package main

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/mmcdole/gofeed"
)

type limitedFeed struct {
	Since time.Time
	Feed  *gofeed.Feed
}

const maxFeedSize = 10 * 1048576

var client = &http.Client{Timeout: time.Second * 30}

func readFeed(link string, since time.Time) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("feed: \"%s\": panic: %v", link, r)
		}
	}()

	resp, err := client.Get(link)
	if err != nil {
		log.Printf("feed: \"%s\": %v", link, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("feed: \"%s\": HTTP status %d", link, resp.StatusCode)
		return
	}

	parser := gofeed.NewParser()
	feed, err := parser.Parse(&io.LimitedReader{R: resp.Body, N: maxFeedSize})
	if err != nil {
		log.Printf("feed: \"%s\": %v", link, err)
		return
	}

	printFeed(limitedFeed{since, feed})
}
