package main

import (
	"fmt"
	"time"
)

func addURL(u string, ignoreErrors, skipContent bool) error {
	pg, err := visit(u, ignoreErrors, skipContent)
	if err != nil {
		return fmt.Errorf("add: %s: %w", u, err)
	}
	pg.AddedAt = time.Now()
	if err := insertPage(pg); err != nil {
		return fmt.Errorf("add: %s: %w", u, err)
	}

	return nil
}
