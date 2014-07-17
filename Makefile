all: build styles javascript

clean:
	@rm -rf pastedown

run: build
	./pastedown

build: styles javascript
	go build -o ./pastedown

styles:
	sass sass/style.scss public/style.css

javascript:
	coffee -c -o public coffee/*.coffee

tarball:
	tar czf pastedown_built.tgz pastedown view.html vendor public files/about.markdown files/reference.markdown

fmt:
	@gofmt -s -l -w .

watch:
	reflex -d fancy -c Reflexfile
