# Pastedown

An easy-to-use markdown-formatting pastebin.

## Features

* Markdown rendering and syntax highlighting
* Doesn't look terrible
* HTTPS built in
* Automatic document deletion

## Dependencies:

Pastedown is written in [Go](http://golang.org) and uses
[Sass](http://sass-lang.com/) for generating CSS. It uses
[Pygments](http://pygments.org/) for syntax highlighting.

To run Pastedown, you will need:

* Python

To build/develop Pastedown you will also require:

* [sassc](https://github.com/sass/sassc)
* [Go](http://golang.org)
* [Reflex](https://github.com/cespare/reflex)

## Installation

For now, the process is:

1. Clone this repo.
1. Run `make`.

This builds the server executable (`pastedown`) and the associated static files.
Run it with:

    $ ./pastedown [OPTIONS]

Use `./pastedown -h` to see all the available options.

## Development

You'll need Reflex and Go as in the installation instructions. Use the following
command to run the server and rebuild/rerun it when files change:

    $ reflex -d fancy -c Reflexfile

This is also available as

    $ make watch

## Deployment

Follow the installation instructions, then run this command:

    $ make tarball

to build a tarball of all the files you'll need. Copy this to your server and
run pastedown with the options you want as before. You may wish to make an init
script or more fleshed-out deployment scripts.
