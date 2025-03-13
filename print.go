package main

import (
	"html"
	"log"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/kr/text"
)

const tmplPageText = `
{{- .URL}}
{{- if isHTML .}}
{{unescape .Title}}

{{contentFmt 4 70 .Description}}
{{- else}} {{if .MimeType}} MIME Type {{.MimeType}}{{end}}{{end}}
`

var spaces = strings.Repeat(" ", 120)

var tmplPage = template.Must(template.New("page").Funcs(template.FuncMap{
	"unescape":   html.UnescapeString,
	"contentFmt": contentFmt,
	"dateFmt":    dateFmt,
	"indent":     indent,
	"isHTML":     isHTML,
}).Parse(tmplPageText))

func printPage(pg *page) {
	if err := tmplPage.Execute(os.Stdout, pg); err != nil {
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
