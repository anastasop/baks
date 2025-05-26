package main

import (
	"fmt"
)

func addURL(URL string, ignoreErrors, skipContent bool) error {
	pg, err := visit(URL, ignoreErrors, skipContent)
	if err != nil {
		return err
	}

	if err := insertPage(pg); err != nil {
		return fmt.Errorf("add: %s orig: %s db: %w", pg.URL, pg.URLorig, err)
	}

	return nil
}
