package main

import (
	"errors"
	"fmt"
)

func addURL(URL string, ignoreErrors, skipContent bool) error {
	pg, err := visit(URL, ignoreErrors, skipContent)
	if err != nil {
		var verr VisitURLError
		if errors.As(err, &verr) {
			return err
		} else {
			return NewVisitURLError(URL, fmt.Errorf("add: %w", err))
		}
	}

	if err := insertPage(pg); err != nil {
		return NewVisitURLError(pg.URL, fmt.Errorf("original: %s: db: %w", pg.URLorig, err))
	}

	return nil
}
