# Pastedown

An easy-to-use markdown-formatting pastebin.

## Demo

http://pastedown.ctrl-c.us

## Features

* Markdown rendering and syntax highlighting
* Doesn't look terrible
* HTTPS built in
* Automatic document deletion

## Dependencies:

Pastedown is written in [Go](http://golang.org) and uses [Sass](http://sass-lang.com/) and
[Coffeescript](http://coffeescript.org/) for generating stylesheets and Javascript. It uses
[Pygments](http://pygments.org/) for syntax highlighting.

To run Pastedown, you will need:

* Python

To build/develop Pastedown you will also require:

* Ruby/bundler and all gems in the Gemfile
* [Go](http://golang.org)

## Installation

For now, the process is:

1. Clone this repo.
1. `$ bundle install`
1. `$ bundle exec rake`

This builds the server executable (`pastedown`) and the associated static files. Run it with:

    $ ./pastedown [OPTIONS]

Use `./pastedown -h` to see all the available options.

## Development

You'll need Ruby/Bundler and Go as in the installation instructions. Use the following command to run the
server and rebuild/rerun it when files change:

    $ bundle exec guard

## Deployment

Follow the installation instructions, then run this command:

    $ bundle exec rake build:tarball

to build a tarball of all the files you'll need. Copy this to your server and run pastedown with the options
you want as before. You may wish to make an init script or more fleshed-out deployment scripts.
