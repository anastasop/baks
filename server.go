package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"time"
)

type SearchResults struct {
	Query string
	Count int
	Pages []*page
}

//go:embed static
var content embed.FS

var resultsTemplate = template.Must(template.New("results.tmpl").Funcs(templateFuncs).ParseFS(content, "static/results.tmpl"))

var templateFuncs = template.FuncMap(map[string]any{
	"truncate": func(l int, s string) string { return s[0:min(len(s), l)] },
	"timefmt":  func(t time.Time) string { return t.Format(time.DateOnly) },
	"ifempty": func(dflt, s string) string {
		if s == "" {
			return dflt
		}
		return s
	},
})

func searchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("q")

	count, err := searchCount(query)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "internal error: %v", err)
		return
	}

	pages, err := search(query)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "internal error: %v", err)
		return
	}

	w.Header().Set("Cache-Control", "no-cache")
	resultsTemplate.Execute(w, SearchResults{Query: query, Count: count, Pages: pages})
}

func recentHandler(w http.ResponseWriter, r *http.Request) {
	n := 30
	if s := r.FormValue("n"); s != "" {
		n, _ = strconv.Atoi(s)
	}

	pages, err := recent(n)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "internal error: %v", err)
		return
	}

	w.Header().Set("Cache-Control", "no-cache")
	resultsTemplate.Execute(w, SearchResults{Query: "Recent", Count: len(pages), Pages: pages})
}

func randomHandler(w http.ResponseWriter, r *http.Request) {
	n := 30
	if s := r.FormValue("n"); s != "" {
		n, _ = strconv.Atoi(s)
	}

	pages, err := random(n)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "internal error: %v", err)
		return
	}

	w.Header().Set("Cache-Control", "no-cache")
	resultsTemplate.Execute(w, SearchResults{Query: "Random", Count: len(pages), Pages: pages})
}

func startServer(addr string) error {
	cnt, err := fs.Sub(content, "static")
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("GET /search", searchHandler)
	http.HandleFunc("GET /recent", recentHandler)
	http.HandleFunc("GET /random", randomHandler)
	http.Handle("GET /", http.FileServerFS(cnt))
	return http.ListenAndServe(addr, nil)
}
