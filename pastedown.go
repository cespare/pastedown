package main

import (
	"github.com/cespare/blackfriday"
	"io/ioutil"
	"os"
)

func main() {
	contents, err := ioutil.ReadFile("test.md")
	if err != nil {
		panic(err)
	}
	flags := 0
	flags |= blackfriday.HTML_COMPLETE_PAGE
	flags |= blackfriday.HTML_GITHUB_BLOCKCODE
	renderer := blackfriday.HtmlRenderer(flags, "Title!", "")

	extensions := 0
	extensions |= blackfriday.EXTENSION_FENCED_CODE
	extensions |= blackfriday.EXTENSION_TABLES
	extensions |= blackfriday.EXTENSION_NO_INTRA_EMPHASIS
	extensions |= blackfriday.EXTENSION_SPACE_HEADERS
	output := blackfriday.Markdown(contents, renderer, extensions)
	os.Stdout.Write(output)
}
