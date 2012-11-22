package main

import (
	"github.com/russross/blackfriday"
	"io/ioutil"
)

func main() {
	contents, err := ioutil.ReadFile("test.md")
	if err != nil {
		panic(err)
	}
	flags := 0
	renderer = blackfriday.HtmlRenderer(flags, "Title!", "")
}
