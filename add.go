package main

import (
	"fmt"
	"time"
)

func addURL(u, tag, referrer string, ignoreErrors, skipContent bool) error {
	pg, err := visit(u, ignoreErrors, skipContent)
	if err != nil {
		return fmt.Errorf("add: \"%s\": %w", u, err)
	}

	if tag == "" {
		tag = findTag(pg.Host)
	}

	pg.Tag = tag
	pg.Referrer = referrer
	pg.AddedAt = time.Now()
	if err := insertPage(pg); err != nil {
		return fmt.Errorf("add: \"%s\": %w", u, err)
	}

	return nil
}
