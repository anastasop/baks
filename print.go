package main

import (
	"html"
	"log"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/kr/text"
	"github.com/mmcdole/gofeed"
)

const tmplPageText = `
{{- .URL}}{{labels .Tag .Referrer}}
{{- if isHTML .}}
{{unescape .Title}}

{{contentFmt 4 70 .Description}}
{{- else}} {{if .MimeType}} MIME Type {{.MimeType}}{{end}}{{end}}
`

const tmplFeedText = `
{{- .Feed.Title}}
{{range recentItems .Since .Feed.Items}}
{{indent 2}}{{dateFmt .PublishedParsed}} {{unescape .Title}}
{{indent 9}}{{.Link}}

{{contentFmt 9 70 .Content}}
{{end}}
`

var spaces = strings.Repeat(" ", 120)

var tmplPage = template.Must(template.New("page").Funcs(template.FuncMap{
	"unescape":   html.UnescapeString,
	"contentFmt": contentFmt,
	"dateFmt":    dateFmt,
	"indent":     indent,
	"labels":     labels,
	"isHTML":     isHTML,
}).Parse(tmplPageText))

var tmplFeed = template.Must(template.New("feed").Funcs(template.FuncMap{
	"unescape":    html.UnescapeString,
	"recentItems": recentItems,
	"contentFmt":  contentFmt,
	"dateFmt":     dateFmt,
	"indent":      indent,
	"labels":      labels,
	"isHTML":      isHTML,
}).Parse(tmplFeedText))

func printPage(pg *page) {
	if err := tmplPage.Execute(os.Stdout, pg); err != nil {
		log.Fatal(err)
	}
}

func printFeed(feed limitedFeed) {
	if err := tmplFeed.Execute(os.Stdout, feed); err != nil {
		log.Fatal(err)
	}
}

func isHTML(pg *page) bool {
	return strings.HasPrefix(pg.MimeType, "text/html")
}

func indent(i int) string {
	return spaces[0:i]
}

func dateFmt(t *time.Time) string {
	return t.Format("02 Jan")
}

func contentFmt(ind int, wid int, content string) string {
	cont := html.UnescapeString(content)
	trimmed := cont
	if len(trimmed) > 300 {
		trimmed = trimmed[0:300]
	}
	return text.Indent(text.Wrap(trimmed, wid), indent(ind))
}

func labels(labels ...string) string {
	if labels == nil || len(labels) == 0 {
		return ""
	}

	s := ""
	for _, t := range labels {
		if t != "" {
			s += t + ":"
		}
	}

	if s == "" {
		return ""
	}
	return " -- :" + s
}

func recentItems(since time.Time, items []*gofeed.Item) []*gofeed.Item {
	its := make([]*gofeed.Item, 0, len(items))
	for _, item := range items {
		if when := item.PublishedParsed; when != nil && when.After(since) {
			its = append(its, item)
		}
	}
	return its
}
