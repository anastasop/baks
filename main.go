package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func prepareWorkDirectory() string {
	cdir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}

	workDir := filepath.Join(cdir, "baks")
	if err := os.Mkdir(workDir, os.ModePerm); err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}

	return filepath.Join(workDir, "baks.db")
}

func main() {
	log.SetPrefix("")
	log.SetFlags(0)

	rootFs := flag.NewFlagSet("rootFlags", flag.ExitOnError)
	rootDB := rootFs.String("db", "", "database path. Use --path to print default path")
	rootPath := rootFs.Bool("path", false, "print database path")
	rootCmd := &ffcli.Command{
		Name:       "baks",
		ShortUsage: "baks [flags] subcommand [flags] <arguments>...",
		ShortHelp:  "Baks is a swiss army knife for bookmarks",
		LongHelp: `Baks is a swiss army knife for bookmarks.
It stores bookmarks on an sqlite3 database and supports full text search on
title and description. Each bookmark also has two labels: the tag which is the
user defined class of the bookmark (ex news, culture) and the referrer which
is information on how the url was found (ex twitter, google groups). These are
set by the user when adding a url.
Baks also has experimental support for atom feeds.`,
		FlagSet: rootFs,
		Exec: func(ctx context.Context, args []string) error {
			if *rootPath {
				fmt.Println(*rootDB)
				return nil
			}
			return flag.ErrHelp
		},
	}

	visitFs := flag.NewFlagSet("visitFlags", flag.ExitOnError)
	visitIgnoreErrors := visitFs.Bool("i", false, "ignore http errors")
	visitSkipContent := visitFs.Bool("n", false, "don't read content")
	visitCmd := &ffcli.Command{
		Name:       "visit",
		ShortUsage: "visit [flags] <url>...",
		ShortHelp:  "Visit the urls and for each one print the title and description",
		LongHelp:   "Visit the urls and for each one print the title and description.",
		FlagSet:    visitFs,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}

			for _, u := range args {
				pg, err := visit(u, *visitIgnoreErrors, *visitSkipContent)
				if err == nil {
					printPage(pg)
					fmt.Println()
				} else {
					log.Printf("%v\n\n", err)
				}
			}

			return nil
		},
	}

	addFs := flag.NewFlagSet("addFlags", flag.ExitOnError)
	addTag := addFs.String("t", "", "tag")
	addReferrer := addFs.String("r", "", "referrer")
	addIgnoreErrors := addFs.Bool("i", false, "ignore http errors")
	addSkipContent := addFs.Bool("n", false, "don't read content")
	addCmd := &ffcli.Command{
		Name:       "add",
		ShortUsage: "add [flags] <url | file>...",
		ShortHelp:  "Add the urls to the database",
		LongHelp: `Add the urls to the database.
If the argument is a url, it it added to the database.
If the argument is a file it is expected to be a text or html file.
For html, usually an export of bookmarks by a browser, the <a href=...>
tags are extracted and the urls are added to the database.
For text each line is assumed to be a valid url. The db contains a table
pages_tags which has predefined rules to set tags by the urls host. These
can be configured manually with the sqlite3 cli app.`,
		FlagSet: addFs,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}

			openDatabase(*rootDB)
			defer closeDatabase()

			for _, arg := range args {
				if strings.HasPrefix(arg, "http") {
					if err := addURL(arg, *addTag, *addReferrer, *addIgnoreErrors, *addSkipContent); err != nil {
						log.Println(err)
					}
				} else {
					anchors, err := anchorsFromFile(arg)
					if err == nil {
						for _, u := range anchors {
							if err := addURL(u.url, *addTag, *addReferrer, *addIgnoreErrors, *addSkipContent); err != nil {
								log.Println(err)
							}
						}
					} else {
						log.Printf("add \"%s\": %v", arg, err)
					}
				}
			}

			return nil
		},
	}

	listFs := flag.NewFlagSet("listFlags", flag.ExitOnError)
	listTag := listFs.String("t", "", "tag")
	listReferrer := listFs.String("r", "", "referrer")
	listCount := listFs.Bool("c", false, "show counts of each tag")
	listCmd := &ffcli.Command{
		Name:       "list",
		ShortUsage: "list [flags]",
		ShortHelp:  "Lists urls with tag or with referrer",
		LongHelp:   "Lists urls with tag or with referrer.",
		FlagSet:    listFs,
		Exec: func(ctx context.Context, args []string) error {
			openDatabase(*rootDB)
			defer closeDatabase()

			if *listCount {
				if counts, err := tagCounts(); err == nil {
					w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
					for _, c := range counts {
						fmt.Fprintf(w, "%s\t%d\n", c.tag, c.count)
					}
					w.Flush()
				} else {
					log.Fatal(err)
				}

				return nil
			}

			if *listTag == "" && *listReferrer == "" {
				return flag.ErrHelp
			}

			if *listTag != "" {
				if pages, err := listByTag(*listTag); err == nil {
					for _, pg := range pages {
						printPage(pg)
						fmt.Println()
					}
				} else {
					return err
				}
			}

			if *listReferrer != "" {
				if pages, err := listByReferrer(*listReferrer); err == nil {
					for _, pg := range pages {
						printPage(pg)
						fmt.Println()
					}
				} else {
					return err
				}
			}

			return nil
		},
	}

	searchFs := flag.NewFlagSet("searchFlags", flag.ExitOnError)
	searchCountOnly := searchFs.Bool("c", false, "display only the result count")
	searchCmd := &ffcli.Command{
		Name:       "search",
		ShortUsage: "search <query>",
		ShortHelp:  "Search urls with full-text search on title or description",
		LongHelp: `Search urls with full-text search on title or description.
The query is an sqlite3 text search query and is applied verbatim:
    SELECT * FROM pages_fts WHERE pages_fts MATCH ?
It can include near and prefix queries as described in the sqlite3 docs.`,
		FlagSet: searchFs,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}

			openDatabase(*rootDB)
			defer closeDatabase()

			if *searchCountOnly {
				if count, err := searchCount(args[0]); err == nil {
					fmt.Println("Found", count, "results.")
				} else {
					return err
				}
			} else {
				if pages, err := search(args[0]); err == nil {
					for _, pg := range pages {
						printPage(pg)
						fmt.Println()
					}
				} else {
					return err
				}
			}

			return nil
		},
	}

	likeFs := flag.NewFlagSet("likeFlags", flag.ExitOnError)
	likeCountOnly := likeFs.Bool("c", false, "display only the result count")
	likeCmd := &ffcli.Command{
		Name:       "like",
		ShortUsage: "like <query>",
		ShortHelp:  "Search urls with an sql like on titles",
		LongHelp:   "Search urls with an sql like on titles.",
		FlagSet:    likeFs,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}

			openDatabase(*rootDB)
			defer closeDatabase()

			if *likeCountOnly {
				if count, err := likeCount(args[0]); err == nil {
					fmt.Println("Results:", count)
				} else {
					return err
				}
			} else {
				if pages, err := like(args[0]); err == nil {
					for _, pg := range pages {
						printPage(pg)
						fmt.Println()
					}
				} else {
					return err
				}
			}

			return nil
		},
	}

	extractFs := flag.NewFlagSet("extractFlags", flag.ExitOnError)
	extractPrintText := extractFs.Bool("t", false, "print anchor text")
	extractResolveURLs := extractFs.Bool("a", false, "print absolute urls")
	extractCmd := &ffcli.Command{
		Name:       "extract",
		ShortUsage: "extract <url | file>...",
		ShortHelp:  "Extract urls from urls or files",
		LongHelp: `Extract extracts urls from files or urls.
If the argument is a url, it fetces the html and prints the anchors
<a href=...> one per line.
If the argument is a file it is expected to be a text or html file.
Text files should contain one url per line. Html files, are
usually exports of bookmarks from browsers. The anchrors are extracted
and printed one per line.`,
		FlagSet: extractFs,
		Exec: func(ctx context.Context, args []string) error {
			for _, arg := range args {
				urls, err := extractAnchors(arg, *extractResolveURLs)
				if err != nil {
					log.Println(err)
					log.Println()
					continue
				}
				if *extractPrintText {
					for _, u := range urls {
						fmt.Printf("%s %s\n\n", u.text, u.url)
					}
					fmt.Println()
				} else {
					for _, u := range urls {
						fmt.Println(u.url)
					}
					fmt.Println()
				}
			}

			return nil
		},
	}

	feedFs := flag.NewFlagSet("feedFlags", flag.ExitOnError)
	feedSince := feedFs.String("s", "", "a date, formatted like 2020-01-01. Return entries newer than this.")
	feedCmd := &ffcli.Command{
		Name:       "feed",
		ShortUsage: "feed <feed url>...",
		ShortHelp:  "Read the most recent entries of the feeds",
		LongHelp:   "Read the most recent entries of the feeds.",
		FlagSet:    feedFs,
		Exec: func(ctx context.Context, args []string) error {
			if *feedSince == "" || len(args) == 0 {
				return flag.ErrHelp
			}

			since, err := time.Parse("2006-01-02", *feedSince)
			if err != nil {
				return flag.ErrHelp
			}

			for _, feed := range args {
				readFeed(feed, since)
			}

			return nil
		},
	}

	rootCmd.Subcommands = []*ffcli.Command{visitCmd, addCmd, searchCmd, listCmd, likeCmd, extractCmd, feedCmd}

	if err := rootCmd.Parse(os.Args[1:]); err != nil && err != flag.ErrHelp {
		log.Fatal(err)
	}

	if *rootDB == "" {
		*rootDB = prepareWorkDirectory()
	}

	if err := rootCmd.Run(context.Background()); err != nil && err != flag.ErrHelp {
		log.Fatal(err)
	}
}
