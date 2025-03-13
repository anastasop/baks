package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

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
		LongHelp:   `Baks is a swiss army knife for bookmarks. It stores bookmarks on an sqlite3 database and supports full text search on title and description.`,
		FlagSet:    rootFs,
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
For text each line is assumed to be a valid url.`,
		FlagSet: addFs,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}

			openDatabase(*rootDB)
			defer closeDatabase()

			for _, arg := range args {
				if strings.HasPrefix(arg, "http") {
					if err := addURL(arg, *addIgnoreErrors, *addSkipContent); err != nil {
						log.Println(err)
					}
				} else {
					anchors, err := anchorsFromFile(arg)
					if err == nil {
						for _, u := range anchors {
							if err := addURL(u.url, *addIgnoreErrors, *addSkipContent); err != nil {
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

	rootCmd.Subcommands = []*ffcli.Command{visitCmd, addCmd, searchCmd}

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
