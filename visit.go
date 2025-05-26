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

const userAgent = `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36 Edg/135.0.0.0`

type VisitError struct {
	URL        string
	HttpError  error
	OtherError error
}

func (e VisitError) Error() string {
	if e.HttpError != nil {
		return fmt.Sprintf("visit: %s proto: %v", e.URL, e.HttpError)
	}
	return fmt.Sprintf("visit: %s other: %v", e.URL, e.OtherError)
}

func (e VisitError) Unwrap() error {
	if e.HttpError != nil {
		return e.HttpError
	}
	return e.OtherError
}

func NewVisitHttpError(URL string, err error) error {
	return VisitError{URL: URL, HttpError: err}
}

func NewVisitOtherError(URL string, err error) error {
	return VisitError{URL: URL, OtherError: err}
}

func visit(URL string, ignoreErrors, skipContent bool) (*page, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancelFunc()

	req, err := http.NewRequestWithContext(ctx, "GET", URL, nil)
	if err != nil {
		return nil, NewVisitOtherError(URL, err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, NewVisitOtherError(URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && !ignoreErrors {
		return nil, NewVisitHttpError(URL, fmt.Errorf("HTTP status %d", resp.StatusCode))
	}

	if redirectedToHostRoot(req.URL.Path, resp.Request.URL.Path) {
		return nil, NewVisitHttpError(URL, fmt.Errorf("redirection to host", resp.Request.URL))
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
		return nil, NewVisitOtherError(URL, fmt.Errorf("read failed: %w", err))
	}

	page.MimeType = http.DetectContentType(body)
	if strings.HasPrefix(page.MimeType, "text/html") {
		if err := setContentAttributes(ctx, page, body, resp.Request.URL); err != nil {
			return nil, NewVisitOtherError(URL, fmt.Errorf("failed to extract page content: %w", err))
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
