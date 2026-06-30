// memecat.go — source-agnostic meme template catalog.
// A Template is just a display title plus a remote image URL; a Catalog is a
// de-duplicated, ordered list of them with title search. Different sources
// (KnowYourMeme CSV, imgflip API) build a Catalog via Add and share this type
// so the template browser does not care where templates came from. CSV
// encode/decode here is the on-disk cache format used by those sources.
package memecat

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// Template is one browsable meme: a display title and the remote image URL.
type Template struct {
	Title    string
	ImageURL string
}

// Catalog is a de-duplicated list of templates in insertion order.
type Catalog struct {
	Templates []Template
	seen      map[string]bool
}

// Len reports the number of templates.
func (c Catalog) Len() int { return len(c.Templates) }

// Add appends a template, dropping empty titles, missing/non-http URLs, and
// titles already present. It reports whether the template was added.
func (c *Catalog) Add(title, url string) bool {
	title = strings.TrimSpace(title)
	url = strings.TrimSpace(url)
	if title == "" || !strings.HasPrefix(url, "http") {
		return false
	}
	if c.seen == nil {
		c.seen = make(map[string]bool)
	}
	if c.seen[title] {
		return false
	}
	c.seen[title] = true
	c.Templates = append(c.Templates, Template{Title: title, ImageURL: url})
	return true
}

// Search returns up to limit templates whose title contains query
// (case-insensitive, surrounding space ignored). A blank query returns the
// first limit templates in catalog order.
func (c Catalog) Search(query string, limit int) []Template {
	q := strings.ToLower(strings.TrimSpace(query))
	var out []Template
	for _, t := range c.Templates {
		if len(out) >= limit {
			break
		}
		if q == "" || strings.Contains(strings.ToLower(t.Title), q) {
			out = append(out, t)
		}
	}
	return out
}

// EncodeCSV writes templates as a title,image_url CSV (with header).
func EncodeCSV(w io.Writer, ts []Template) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"title", "image_url"}); err != nil {
		return err
	}
	for _, t := range ts {
		if err := cw.Write([]string{t.Title, t.ImageURL}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// DecodeCSV reads a title,image_url CSV (as written by EncodeCSV) into a
// Catalog, applying the same filtering as Add.
func DecodeCSV(r io.Reader) (Catalog, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	header, err := cr.Read()
	if err != nil {
		return Catalog{}, fmt.Errorf("read header: %w", err)
	}
	titleCol, urlCol := colIndex(header, "title"), colIndex(header, "image_url")
	if titleCol < 0 || urlCol < 0 {
		return Catalog{}, fmt.Errorf("csv missing title/image_url columns: %v", header)
	}
	var c Catalog
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Catalog{}, err
		}
		if titleCol < len(rec) && urlCol < len(rec) {
			c.Add(rec[titleCol], rec[urlCol])
		}
	}
	return c, nil
}

func colIndex(header []string, name string) int {
	for i, h := range header {
		if strings.ToLower(strings.TrimSpace(h)) == name {
			return i
		}
	}
	return -1
}
