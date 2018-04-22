package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"time"

	"github.com/cespare/hutil/apachelog"
	"github.com/gorilla/pat"
	blackfriday "gopkg.in/russross/blackfriday.v2"

	"github.com/cespare/pastedown/lru"
)

const (
	staticDir                  = "public"
	pygmentize                 = "./vendor/pygments/pygmentize"
	viewFile                   = "view.html"
	expirationCheckPeriodHours = 1
	renderCacheSizeBytes       = 20e6 // 20 MB
)

// User-configurable values
var (
	listenAddr          string
	pastieDir           string
	mainPastie          string
	markdownRefPastie   string
	expirationTimeHours int
	useTls              bool
	tlsCertFile         string
	tlsKeyFile          string
)

var (
	validLanguages = make(map[string]struct{})
	viewHTML       []byte
	filenameRegex  = regexp.MustCompile(`^[\w\-]{27}\.\w+$`)
	expiryMsg      string
	renderCache    = lru.New(renderCacheSizeBytes)
)

func init() {
	var err error

	// Set up flags
	flag.StringVar(&listenAddr, "listenaddr", "localhost:8389", "The server address on which to listen")
	flag.StringVar(&pastieDir, "storagedir", "files", "The directory in which to store documents")
	flag.StringVar(&mainPastie, "mainpage", "about.markdown", "The document to display on the front page")
	flag.StringVar(&markdownRefPastie, "referencepage", "reference.markdown",
		"The document to display at the 'markdown reference' link")
	flag.IntVar(&expirationTimeHours, "expirationhours", 7*24,
		"How long to keep documents before deleting them (0 for 'never delete')")
	flag.BoolVar(&useTls, "tls", false, "Whether to serve over HTTPS.")
	flag.StringVar(&tlsCertFile, "certfile", "cert.pem", "TLS certificate file to use")
	flag.StringVar(&tlsKeyFile, "keyfile", "key.pem", "TLS private key file to use")

	// Get the list of valid lexers from pygments.
	rawLexerList, err := exec.Command(pygmentize, "-L", "lexers").Output()
	if err != nil {
		log.Fatalln(err)
	}
	for _, line := range bytes.Split(rawLexerList, []byte("\n")) {
		if len(line) == 0 || line[0] != '*' {
			continue
		}
		for _, l := range bytes.Split(bytes.Trim(line, "* :"), []byte(",")) {
			lexer := string(bytes.TrimSpace(l))
			if len(lexer) != 0 {
				validLanguages[lexer] = struct{}{}
			}
		}
	}

	// Check that the main info file exists.
	_, err = os.Stat(pastieDir + "/" + mainPastie)
	if err != nil {
		log.Fatalln("Error with main info file: " + err.Error())
	}
}

func syntaxHighlight(out io.Writer, in io.Reader, language string) {
	_, ok := validLanguages[language]
	if !ok || language == "" {
		language = "text"
	}
	pygmentsCommand := exec.Command(pygmentize, "-l", language, "-f", "html", "-P", "encoding=utf-8")
	pygmentsCommand.Stdin = in
	pygmentsCommand.Stdout = out
	var stderr bytes.Buffer
	pygmentsCommand.Stderr = &stderr
	if err := pygmentsCommand.Run(); err != nil {
		log.Println("Error with syntax highlighting:", err)
		log.Println("Stderr:\n", stderr.String())
	}
}

type Pastie struct {
	Text   string `json:"text"`
	Format string `json:"format"`
}

func render(text []byte, format string) []byte {
	var rendered []byte
	switch format {
	case "text":
		rendered = text
	case "markdown":
		html := blackfriday.Run(text)
		rendered = html
	default:
		var highlighted bytes.Buffer
		in := bytes.NewBuffer(text)
		syntaxHighlight(&highlighted, in, format)
		rendered = highlighted.Bytes()
	}
	return rendered
}

func pastieHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get(":id")
	if len(id) == 0 {
		http.Error(w, "No such file.", http.StatusInternalServerError)
		return
	}
	var filename string
	// If the filename is one made by Pastedown, then look in the directory structure we expect; otherwise, just
	// try to find such a file directly.
	if filenameRegex.MatchString(id) {
		filename = path.Join(pastieDir, id[:2], id[2:])
	} else {
		filename = path.Join(pastieDir, id)
	}

	// Render the proper format according to the extension if ?rendered=true.
	if r.URL.Query().Get("rendered") == "true" {
		// First try to find the rendered document in cache
		rendered, ok := renderCache.Get(filename)
		if ok {
			w.Write(rendered)
			return
		}

		// Wasn't in cache; render it from disk.
		contents, err := ioutil.ReadFile(filename)
		if err != nil {
			http.Error(w, "No such file.", http.StatusNotFound)
			return
		}
		extension := path.Ext(id)
		if extension == "" {
			extension = "text"
		} else {
			extension = extension[1:]
		}

		rendered = render(contents, extension)
		renderCache.Insert(filename, rendered)
		w.Write(rendered)
		return
	}

	// Return the raw contents if ?rendered=true wasn't set.
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		http.Error(w, "No such file.", http.StatusNotFound)
		return
	}
	// Fetch the value to mark it as recently used.
	_, _ = renderCache.Get(filename)
	w.Write(contents)
}

func decodePastie(r *http.Request) (*Pastie, error) {
	text, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	pastie := &Pastie{}
	err = json.Unmarshal(text, pastie)
	if err != nil {
		return nil, err
	}
	return pastie, nil
}

func previewHandler(w http.ResponseWriter, r *http.Request) {
	preview, err := decodePastie(r)
	if err != nil {
		http.Error(w, "Could not render preview text.", http.StatusInternalServerError)
		return
	}
	w.Write(render([]byte(preview.Text), preview.Format))
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	preview, err := decodePastie(r)
	bytes := []byte(preview.Text)
	if err != nil {
		log.Println("Error decoding pastie for saving: " + err.Error())
		http.Error(w, "Could not save text.", http.StatusInternalServerError)
		return
	}
	sha := sha1.New()
	sha.Write(bytes)
	hash := base64.URLEncoding.EncodeToString(sha.Sum(nil))

	// The filename is constructed in the following manner:
	//
	// - Chop off the last '=' (padding character) of the hash -- all the
	//   shas are the same length anyway so we might as well get rid of the
	//   character that they all have in common.
	// - Chop the first two characters off the front of the hash and use
	//   this as the directory to limit the number of files in a single
	//   directory (git uses this trick for its object store).
	// - The full file format name is used as the extension.
	//
	// So for example:
	// { sha: jBEtyBOnX_M2rp7DNp3mQskWqwg=, filetype: markdown } => jB/EtyBOnX_M2rp7DNp3mQskWqwg.markdown
	directory := path.Join(pastieDir, hash[0:2])
	logicalName := hash[:len(hash)-1] + "." + preview.Format
	filename := path.Join(directory, hash[2:len(hash)-1]+"."+preview.Format)
	err = os.MkdirAll(directory, 0771)
	if err != nil {
		log.Println("Error creating new directory: " + err.Error())
		http.Error(w, "Could not save text.", http.StatusInternalServerError)
		return
	}
	_, err = os.Stat(filename)
	if err == nil {
		log.Println("Request for existing pastie: " + filename)
	} else {
		log.Println("Saving new pastie: " + filename)
		err = ioutil.WriteFile(filename, bytes, 0666)
		if err != nil {
			log.Println("Error writing pastie file: " + err.Error())
			http.Error(w, "Could not save text.", http.StatusInternalServerError)
			return
		}
	}
	// Otherwise file already exists.
	w.Write([]byte(logicalName))
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	w.Write(viewHTML)
}

func expire() {
	log.Println("Expiring files...")
	dirs, err := ioutil.ReadDir(pastieDir)
	if err != nil {
		log.Println("Error reading file directory: " + err.Error())
		return
	}
	expiredFiles := 0
	expiredDirs := 0
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		dirPath := path.Join(pastieDir, d.Name())
		files, err := ioutil.ReadDir(dirPath)
		if err != nil {
			log.Println("Error reading file directory: " + err.Error())
			continue
		}
		unexpired := false
		for _, f := range files {
			if time.Now().Sub(f.ModTime()).Hours() <= float64(expirationTimeHours) {
				unexpired = true
				continue
			}
			filepath := path.Join(dirPath, f.Name())
			// Remove from cache
			renderCache.Delete(filepath)
			// Remove from disk
			if err := os.Remove(filepath); err != nil {
				log.Println("Error deleting file: " + err.Error())
				continue
			}
			expiredFiles++
		}
		if !unexpired {
			if err := os.Remove(dirPath); err != nil {
				log.Println("Error deleting empty directory: " + err.Error())
				continue
			}
			expiredDirs++
		}
	}
	log.Printf("Removed %d expired files and %d empty directories.\n", expiredFiles, expiredDirs)
}

// Creates a string describing the expiry time.
func createExpiryMsg() string {
	switch {
	case expirationTimeHours == 0:
		return "documents are not deleted"
	case expirationTimeHours > 48:
		return fmt.Sprintf("documents are deleted after %d days", expirationTimeHours/24)
	}
	return fmt.Sprintf("documents are deleted after %d hours", expirationTimeHours)
}

func main() {
	flag.Parse()

	// Start the background file deleter going
	if expirationTimeHours > 0 {
		go func() {
			ticker := time.NewTicker(expirationCheckPeriodHours * time.Hour)
			for _ = range ticker.C {
				expire()
			}
		}()
	}

	// Load in the main view template
	viewTemplate, err := template.ParseFiles(viewFile)
	if err != nil {
		log.Fatalln(err)
	}
	b := new(bytes.Buffer)
	templateData := struct {
		ExpiryMsg, MainId, MarkdownRefId string
	}{createExpiryMsg(), mainPastie, markdownRefPastie}
	err = viewTemplate.Execute(b, templateData)
	if err != nil {
		log.Fatalln(err)
	}
	viewHTML = b.Bytes()

	// Set up the server
	mux := pat.New()

	mux.Add("GET", "/favicon.ico", http.FileServer(http.Dir(staticDir)))
	staticPath := "/" + staticDir + "/"
	mux.Add("GET", staticPath, http.StripPrefix(staticPath, http.FileServer(http.Dir("./"+staticDir))))

	mux.Get(`/files/{id:[\w\.\-]+}`, pastieHandler)
	mux.Post("/preview", previewHandler)
	mux.Put("/file", saveHandler)
	mux.Get("/", viewHandler)

	handler := apachelog.NewDefaultHandler(mux)
	server := &http.Server{
		Addr:    listenAddr,
		Handler: handler,
	}
	log.Println("Now listening on", listenAddr, " TLS =", useTls)
	if useTls {
		log.Fatal(server.ListenAndServeTLS(tlsCertFile, tlsKeyFile))
	} else {
		log.Fatal(server.ListenAndServe())
	}
}
