package main

import (
	"bytes"
	"context"
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
	ctx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancelFunc()

	req, err := http.NewRequestWithContext(ctx, "GET", URL, nil)
	if err != nil {
		return nil, NewVisitURLError(URL, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, NewVisitURLError(URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && !ignoreErrors {
		return nil, NewVisitURLError(URL, fmt.Errorf("HTTP status %d", resp.StatusCode))
	}

	if redirectedToHostRoot(req.URL.Path, resp.Request.URL.Path) {
		return nil, NewVisitURLError(URL, fmt.Errorf("redirection %s: redirection to host", resp.Request.URL))
	}

	page := new(page)
	page.URL = resp.Request.URL.String()
	page.URLorig = URL
	page.AddedAt = time.Now()

	if skipContent {
		return page, nil
	}

	body, err := io.ReadAll(&io.LimitedReader{R: resp.Body, N: maxPageSize})
	if err != nil {
		return nil, NewVisitURLError(URL, fmt.Errorf("read failed: %w", err))
	}

	page.MimeType = http.DetectContentType(body)
	if strings.HasPrefix(page.MimeType, "text/html") {
		if err := setContentAttributes(ctx, page, body, resp.Request.URL); err != nil {
			return nil, NewVisitURLError(URL, fmt.Errorf("set contents failed: %w", err))
		}
	}

	return page, nil
}

func setContentAttributes(ctx context.Context, pg *page, body []byte, base *url.URL) error {
	type pageWithErr struct {
		page *page
		err  error
	}
	rc := make(chan pageWithErr, 1)

	// TODO: if the parser falls in an infinite loop, this is a goroutine leak
	go func(body []byte) {
		doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(body))
		if err != nil {
			rc <- pageWithErr{nil, err}
		}

		dpg := new(page)
		dpg.Title = strings.TrimSpace(doc.Find("head title").Text())

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
			dpg.Description = meta
		} else if twitter != "" {
			dpg.Description = twitter
		} else if og != "" {
			dpg.Description = og
		}

		rc <- pageWithErr{dpg, nil}
	}(body)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case npg := <-rc:
		if npg.err != nil {
			return npg.err
		}
		pg.Title = npg.page.Title
		pg.Description = npg.page.Description
		return nil
	}
}

func redirectedToHostRoot(askPath, gotPath string) bool {
	return isPathEmpty(gotPath) && !isPathEmpty(askPath)
}

func isPathEmpty(p string) bool {
	return p == "/" || p == ""
}
