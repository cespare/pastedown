package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"flag"
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

	"github.com/cespare/blackfriday"
	"github.com/cespare/go-apachelog"
	"github.com/gorilla/pat"
)

const (
	staticDir                  = "public"
	pygmentize                 = "./vendor/pygments/pygmentize"
	viewFile                   = "view.html"
	expirationCheckPeriodHours = 1
)

// User-configurable values
var (
	listenAddr              string
	pastieDir               string
	mainPastie              string
	markdownReferencePastie string
	expirationTimeHours     int
)

var (
	validLanguages     = make(map[string]struct{})
	markdownRenderer   *blackfriday.Html
	markdownExtensions int
	viewHtml           []byte
	filenameRegex      = regexp.MustCompile(`^[\w\-]{27}\.\w+$`)
)

func init() {
	var err error

	// Set up flags
	flag.StringVar(&listenAddr, "listenaddr", "localhost:8389", "The server address on which to listen")
	flag.StringVar(&pastieDir, "storagedir", "files", "The directory in which to store documents")
	flag.StringVar(&mainPastie, "mainpage", "about.markdown", "The document to display on the front page")
	flag.StringVar(&markdownReferencePastie, "referencepage", "reference.markdown",
		"The document to display at the 'markdown reference' link")
	flag.IntVar(&expirationTimeHours, "expirationhours", 7*24,
		"How long to keep documents before deleting them")

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

	// Set up the renderer.
	flags := 0
	flags |= blackfriday.HTML_GITHUB_BLOCKCODE
	markdownRenderer = blackfriday.HtmlRenderer(flags, "", "")
	markdownRenderer.SetBlockCodeProcessor(syntaxHighlight)

	markdownExtensions = 0
	markdownExtensions |= blackfriday.EXTENSION_FENCED_CODE
	markdownExtensions |= blackfriday.EXTENSION_TABLES
	markdownExtensions |= blackfriday.EXTENSION_NO_INTRA_EMPHASIS
	markdownExtensions |= blackfriday.EXTENSION_SPACE_HEADERS
	markdownExtensions |= blackfriday.EXTENSION_AUTOLINK

	// Check that the main info file exists.
	_, err = os.Stat(pastieDir + "/" + mainPastie)
	if err != nil {
		log.Fatalln("Error with main info file: " + err.Error())
	}

	// Load in the main view template
	viewTemplate, err := template.ParseFiles(viewFile)
	if err != nil {
		log.Fatalln(err)
	}
	b := new(bytes.Buffer)
	templateData := struct{ MainId, MarkdownReferenceId string }{mainPastie, markdownReferencePastie}
	err = viewTemplate.Execute(b, templateData)
	if err != nil {
		log.Fatalln(err)
	}
	viewHtml = b.Bytes()
}

func syntaxHighlight(out io.Writer, in io.Reader, language string) {
	_, ok := validLanguages[language]
	if !ok || language == "" {
		language = "text"
	}
	pygmentsCommand := exec.Command(pygmentize, "-l", language, "-f", "html")
	pygmentsCommand.Stdin = in
	pygmentsCommand.Stdout = out
	pygmentsCommand.Run()
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
		rendered = blackfriday.Markdown(text, markdownRenderer, markdownExtensions)
	default:
		var highlighted bytes.Buffer
		in := bytes.NewBuffer(text)
		syntaxHighlight(&highlighted, in, format)
		rendered = highlighted.Bytes()
	}
	return rendered
}

// Update the mtime
func touch(path string) { os.Chtimes(path, time.Now(), time.Now()) }

func pastieHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(0 * time.Second)
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
	touch(filename)
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		http.Error(w, "No such file.", http.StatusNotFound)
		return
	}

	// Just return the raw contents if ?rendered=true wasn't set.
	if r.URL.Query().Get("rendered") != "true" {
		w.Write(contents)
		return
	}

	// Otherwise, render the proper format according to the extension.
	extension := path.Ext(id)
	if extension == "" {
		extension = "text"
	} else {
		extension = extension[1:]
	}

	w.Write(render(contents, extension))
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
	// - Chop off the last '=' (padding character) of the hash -- all the shas are the same length anyway so
	//   we might as well get rid of the character that they all have in common.
	// - Chop the first two characters off the front of the hash and use this as the directory to limit the
	//	 number files in a single directory (git uses this trick for its object store).
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
		touch(filename)
	} else {
		log.Println("Saving new pastie: " + filename)
		err = ioutil.WriteFile(filename, bytes, 0666)
		if err != nil {
			log.Println("Error writing pastie file: " + err.Error())
			http.Error(w, "Could not save text.", http.StatusInternalServerError)
			return
		}
	}
	// Otherwise file already exists
	w.Write([]byte(logicalName))
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	w.Write(viewHtml)
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

func main() {
	flag.Parse()

	// Start the background file deleter going
	go func() {
		ticker := time.NewTicker(expirationCheckPeriodHours * time.Hour)
		for _ = range ticker.C {
			expire()
		}
	}()

	// Set up the server
	mux := pat.New()

	mux.Add("GET", "/favicon.ico", http.FileServer(http.Dir(staticDir)))
	staticPath := "/" + staticDir + "/"
	mux.Add("GET", staticPath, http.StripPrefix(staticPath, http.FileServer(http.Dir("./"+staticDir))))

	mux.Get(`/files/{id:[\w\.\-]+}`, pastieHandler)
	mux.Post("/preview", previewHandler)
	mux.Put("/file", saveHandler)
	mux.Get("/", viewHandler)

	handler := apachelog.NewHandler(mux, os.Stderr)
	server := &http.Server{
		Addr:    listenAddr,
		Handler: handler,
	}
	log.Println("Now listening on", listenAddr)
	log.Fatalf(server.ListenAndServe().Error())
}
