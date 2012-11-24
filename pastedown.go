package main

import (
	"bytes"
	"encoding/json"
	"github.com/cespare/blackfriday"
	"github.com/cespare/go-apachelog"
	"github.com/gorilla/pat"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"time"
)

const (
	listenAddr = "localhost:8389"
	staticDir  = "public"
	pastieDir  = "files"
	mainPastie = "about.markdown"
	pygmentize = "./vendor/pygments/pygmentize"
	viewFile   = "view.html"
)

var (
	validLanguages     = make(map[string]struct{})
	markdownRenderer   *blackfriday.Html
	markdownExtensions int
	viewHtml           []byte
)

type Pastie struct {
	Contents string `json:"contents"`
	Format   string `json:"format"`
}

func init() {
	var err error

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
	err = viewTemplate.Execute(b, struct{ MainId string }{mainPastie})
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

func pastieHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(0 * time.Second)
	id := r.URL.Query().Get(":id")
	if len(id) == 0 {
		http.Error(w, "No such file.", http.StatusNotFound)
		return
	}
	contents, err := ioutil.ReadFile(pastieDir + "/" + id)
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

	var rendered string
	switch extension {
	case "text":
		rendered = string(contents)
	case "markdown":
		rendered = string(blackfriday.Markdown(contents, markdownRenderer, markdownExtensions))
	default:
		var highlighted bytes.Buffer
		in := bytes.NewBuffer(contents)
		syntaxHighlight(&highlighted, in, extension)
		rendered = highlighted.String()
	}
	message := &Pastie{rendered, extension}
	encoded, err := json.Marshal(message)
	if err != nil {
		http.Error(w, "Error encoding message.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(encoded)
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	w.Write(viewHtml)
}

func main() {
	mux := pat.New()

	mux.Add("GET", "/favicon.ico", http.FileServer(http.Dir(staticDir)))
	staticPath := "/" + staticDir + "/"
	mux.Add("GET", staticPath, http.StripPrefix(staticPath, http.FileServer(http.Dir("./"+staticDir))))

	mux.Get("/files/{id:[\\w\\.]+}", pastieHandler)
	mux.Get("/", viewHandler)

	handler := apachelog.NewHandler(mux, os.Stderr)
	server := &http.Server{
		Addr:    listenAddr,
		Handler: handler,
	}
	log.Println("Now listening on", listenAddr)
	log.Fatalf(server.ListenAndServe().Error())
}
