package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func visit(url string, ignoreErrors, skipContent bool) (*page, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("visit \"%s\": %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && !ignoreErrors {
		return nil, fmt.Errorf("visit \"%s\": HTTP status %d", url, resp.StatusCode)
	}

	page := new(page)
	page.URL = resp.Request.URL.String()

	if !skipContent {
		if typ, doc, err := readContent(resp); err == nil {
			page.MimeType = typ
			if doc != nil {
				setContentAttributes(page, doc, resp.Request.URL)
			}
		} else {
			log.Printf("visit \"%s\": ignoring content: error: %v", url, err)
		}
	}

	return page, nil
}

func readContent(resp *http.Response) (string, *goquery.Document, error) {
	body, err := io.ReadAll(&io.LimitedReader{R: resp.Body, N: maxPageSize})
	if err != nil {
		return "", nil, err
	}

	if typ := http.DetectContentType(body); !strings.HasPrefix(typ, "text/html") {
		return typ, nil, nil
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(body))
	return "text/html", doc, err
}

func setContentAttributes(pg *page, doc *goquery.Document, base *url.URL) {
	pg.Title = strings.TrimSpace(doc.Find("head title").Text())

	// fill page description
	var meta, og, twitter string

	doc.Find("head meta").Each(func(i int, s *goquery.Selection) {
		if val, exists := s.Attr("name"); exists && val == "description" {
			meta = strings.TrimSpace(s.AttrOr("content", ""))
		}
		if val, exists := s.Attr("property"); exists && val == "og:description" {
			og = strings.TrimSpace(s.AttrOr("content", ""))
		}
		if val, exists := s.Attr("property"); exists && val == "twitter:description" {
			twitter = strings.TrimSpace(s.AttrOr("content", ""))
		}
	})

	if meta != "" {
		pg.Description = meta
	} else if twitter != "" {
		pg.Description = twitter
	} else if og != "" {
		pg.Description = og
	}
}
