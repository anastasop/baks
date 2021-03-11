package main

import "time"

const maxPageSize = 10 * 1048576

type page struct {
	URL         string
	Title       string
	Description string
	MimeType    string
	AtomURL     string
	Tag         string
	Referrer    string
	Host        string
	IsRootPage  bool
	AddedAt     time.Time
}
