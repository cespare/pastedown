# Pastedown

An easy-to-use markdown-formatting pastebin.

## Installation

For now, the process is:

1. Clone this repo.
1. Install Ruby/Bundler.
1. Install [Go](http://golang.org)
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
