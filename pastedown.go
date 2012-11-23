package main

import (
	"bytes"
	"github.com/cespare/blackfriday"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

var (
	pygmentize         = "./vendor/pygments/pygmentize"
	validLanguages     = make(map[string]struct{})
	markdownRenderer   *blackfriday.Html
	markdownExtensions int
)

func init() {
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

func main() {
	contents, err := ioutil.ReadFile("test.md")
	if err != nil {
		panic(err)
	}
	output := blackfriday.Markdown(contents, markdownRenderer, markdownExtensions)
	os.Stdout.Write(output)
}
