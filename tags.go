package main

import "strings"

type tag struct {
	Host string
	tag  string
}

type tagCount struct {
	tag   string
	count int
}

var predefinedTags []tag

func findTag(host string) string {
	for _, pre := range predefinedTags {
		if strings.HasSuffix(host, pre.Host) {
			return pre.tag
		}
	}

	return ""
}
