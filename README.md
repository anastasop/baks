# Baks

Baks is a tool to handle bookmarks. The usual flow is: `add url`, `add url`, `add url`, *time passes*, **you forget all of them**, use a full text index to find the url you once read but only vaguely remember.

Baks also provides some useful tools to extracts urls from pages, read atoms feeds etc.

It is not a replacement for a bookmarks service like [pocket](https://getpocket.com/), or [tefter](https://tefter.io/) but just a simple program for the common cases.

## Usage

Baks provides online help. Use `--help` with each subcommand

```
USAGE
  baks [flags] subcommand [flags] <arguments>...

Baks is a swiss army knife for bookmarks.
It stores bookmarks on an sqlite3 database and supports full text search on
title and description. Each bookmark also has two labels: the tag which is the
user defined class of the bookmark (ex news, culture) and the referrer which
is information on how the url was found (ex twitter, google groups). These are
set by the user when adding a url.
Baks also has experimental support for atom feeds.

SUBCOMMANDS
  visit    Visit the urls and for each one print the title and description
  add      Add the urls to the database
  search   Search urls with full-text search on title or description
  list     Lists urls with tag or with referrer
  like     Search urls with an sql like on titles
  extract  Extract urls from urls or files
  feed     Read the most recent entries of the feeds

FLAGS
  -db ...      database path. Use --path to print default path
  -path false  print database path
```

## Installation

Baks is tested with go 1.16 on windows and debian linux, including WSL and crostini.

Install with
```
go get --tags=fts5 github.com/anastasop/baks
```

Baks has a dependency on the sqlite3 driver https://github.com/mattn/go-sqlite3 which is a cgo driver.
If the installation of baks fails then you should install the sqlite3 driver manually and then continue with
baks.

## Bugs

Automate more things, that now need the sqlite3 cli.
