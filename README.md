# Baks

Baks is a tool to handle bookmarks. The usual flow is: `add url`, `add url`, `add url`, *time passes*, **you forget all of them**, use a full text index to find the url you once read but only vaguely remember.

It is not a replacement for a bookmarks service like [pocket](https://getpocket.com/), or [tefter](https://tefter.io/) but just a simple program for the common cases.

As a special feature it can launch an http server for an [opensearch](https://developer.mozilla.org/en-US/docs/Web/XML/Guides/OpenSearch) provider.

## Usage

Baks provides online help. Use `--help` with each subcommand

```
DESCRIPTION
  Baks is a swiss army knife for bookmarks

USAGE
  baks [flags] subcommand [flags] <arguments>...

Baks is a swiss army knife for bookmarks. It stores bookmarks on an sqlite3
database and supports full text search on title and description.

SUBCOMMANDS
  add     Add the urls to the database
  search  Search urls with full-text search on title or description
  server  Launch an http server with a search API

FLAGS
  -db string   database path. Use --path to print default path
  -path=false  print database path
```

## Installation

Install with
```
go install --tags=fts5 github.com/anastasop/baks@latest
```

Baks has a dependency on the sqlite3 driver https://github.com/mattn/go-sqlite3 which is a cgo driver.
If the installation of baks fails then you should install the sqlite3 driver manually and then continue with
baks.
