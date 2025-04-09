package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type VisitURLError struct {
	URL string
	Err error
}

func (e VisitURLError) Error() string {
	return fmt.Sprintf("visit: %s: %v", e.URL, e.Err)
}

func (e VisitURLError) Unwrap() error {
	return e.Err
}

func NewVisitURLError(URL string, err error) error {
	return VisitURLError{URL, err}
}

func visit(URL string, ignoreErrors, skipContent bool) (*page, error) {
	resp, err := http.Get(URL)
	if err != nil {
		return nil, NewVisitURLError(URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && !ignoreErrors {
		return nil, NewVisitURLError(URL, fmt.Errorf("HTTP status %d", resp.StatusCode))
	}

	page := new(page)
	page.URL = resp.Request.URL.String()
	page.AddedAt = time.Now()

	if skipContent {
		return page, nil
	}

	typ, doc, err := readContent(resp)
	if err != nil {
		return nil, NewVisitURLError(URL, fmt.Errorf("read content failed: %w", err))
	}

	page.MimeType = typ
	if doc != nil {
		setContentAttributes(page, doc, resp.Request.URL)
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
