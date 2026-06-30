// catalog.go — parse the KnowYourMeme dataset CSV into a shared meme catalog.
// The dataset carries many columns; only title and image_url matter, so parsing
// is column-name based and delegates filtering/dedup to memecat.Catalog. No
// network or filesystem concerns live here — see fetch.go for that.
package kym

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/mganter/memegen-tui/pkg/memecat"
)

// ParseCSV reads the dataset CSV, keeping the title and image_url columns and
// building a memecat.Catalog. It errors if either required column is absent.
func ParseCSV(r io.Reader) (memecat.Catalog, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1 // tolerate ragged rows
	header, err := cr.Read()
	if err != nil {
		return memecat.Catalog{}, fmt.Errorf("read header: %w", err)
	}
	titleCol, urlCol := -1, -1
	for i, h := range header {
		switch strings.ToLower(strings.TrimSpace(h)) {
		case "title":
			titleCol = i
		case "image_url":
			urlCol = i
		}
	}
	if titleCol < 0 || urlCol < 0 {
		return memecat.Catalog{}, fmt.Errorf("csv missing title/image_url columns: %v", header)
	}

	var c memecat.Catalog
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return memecat.Catalog{}, err
		}
		if titleCol < len(rec) && urlCol < len(rec) {
			c.Add(rec[titleCol], rec[urlCol])
		}
	}
	return c, nil
}
