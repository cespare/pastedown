all: build styles

clean:
	@rm -rf pastedown

run: build
	./pastedown

build:
	go build -o ./pastedown

styles:
	sassc sass/style.scss public/style.css

tarball: build styles
	tar czf pastedown_built.tgz pastedown view.html vendor public files/about.markdown files/reference.markdown

fmt:
	@gofmt -s -l -w .

watch:
	reflex -d fancy -c Reflexfile
