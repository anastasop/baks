package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

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
		LongHelp: `Baks is a swiss army knife for bookmarks. It stores bookmarks on an sqlite3
database and supports full text search on title and description.`,
		FlagSet: rootFs,
		Exec: func(ctx context.Context, args []string) error {
			if *rootPath {
				fmt.Println(*rootDB)
				return nil
			}
			return flag.ErrHelp
		},
	}

	addFs := flag.NewFlagSet("addFlags", flag.ExitOnError)
	addIgnoreErrors := addFs.Bool("i", false, "ignore http errors")
	addSkipContent := addFs.Bool("n", false, "don't read content")
	addCmd := &ffcli.Command{
		Name:       "add",
		ShortUsage: "add [flags] <url>...",
		ShortHelp:  "Add the urls to the database",
		LongHelp:   "Add the urls to the database",
		FlagSet:    addFs,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}

			openDatabase(*rootDB)
			defer closeDatabase()

			for _, arg := range args {
				if err := addURL(arg, *addIgnoreErrors, *addSkipContent); err != nil {
					log.Println(err)
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

	serverFs := flag.NewFlagSet("serverFlags", flag.ExitOnError)
	serverAddr := serverFs.String("l", ":8080", "server address")
	serverCmd := &ffcli.Command{
		Name:       "server",
		ShortUsage: "server [flags] <url>...",
		ShortHelp:  "Launch an http server with a search API",
		LongHelp:   "Launch an http server with a search API",
		FlagSet:    serverFs,
		Exec: func(ctx context.Context, args []string) error {
			openDatabase(*rootDB)
			defer closeDatabase()

			return startServer(*serverAddr)
		},
	}

	rootCmd.Subcommands = []*ffcli.Command{addCmd, searchCmd, serverCmd}

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
