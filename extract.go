package main

import (
	"bufio"
	"bytes"
	"html"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type anchor struct {
	url  string
	text string
}

func anchorsFromFile(fname string) ([]anchor, error) {
	r, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return anchorsFromReader(r, nil, false)
}

func anchorsFromReader(content io.Reader, base *url.URL, resolve bool) ([]anchor, error) {
	buf := bufio.NewReader(content)

	line, err := buf.Peek(512)
	if err != nil {
		return nil, err
	}

	if bytes.HasPrefix(line, []byte("http")) {
		return anchorsFromTextFile(buf)
	}
	return anchorsFromHTMLFile(buf, base, resolve)
}

func anchorsFromTextFile(content io.Reader) ([]anchor, error) {
	anchors := make([]anchor, 0, 32)

	scanner := bufio.NewScanner(content)
	for scanner.Scan() {
		anchors = append(anchors, anchor{scanner.Text(), ""})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return anchors, nil
}

func anchorsFromHTMLFile(content io.Reader, base *url.URL, resolve bool) ([]anchor, error) {
	anchors := make([]anchor, 0, 32)

	doc, err := goquery.NewDocumentFromReader(content)
	if err != nil {
		return nil, err
	}

	if resolve {
		if sel := doc.Find("base"); sel != nil {
			if u, exists := sel.Attr("href"); exists {
				if r, err := url.Parse(strings.TrimSpace(u)); err != nil {
					base = r
				}
			}
		}
	}

	doc.Find("a").Each(func(_ int, sel *goquery.Selection) {
		if text, exists := sel.Attr("href"); exists {
			u := strings.TrimSpace(text)
			if base != nil {
				if ru, err := url.Parse(u); err == nil {
					u = base.ResolveReference(ru).String()
				}
			}
			anchors = append(anchors, anchor{
				u,
				strings.TrimSpace(html.UnescapeString(sel.Text())),
			})
		}
	})

	return anchors, nil
}
