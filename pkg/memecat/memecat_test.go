package memecat

import (
	"strings"
	"testing"
)

func TestAddFiltersAndDedups(t *testing.T) {
	var c Catalog
	cases := []struct {
		title, url string
		want       bool
	}{
		{"Drakeposting", "https://x/a.jpg", true},
		{"", "https://x/b.jpg", false},           // empty title
		{"No URL", "", false},                    // empty url
		{"Bad URL", "ftp://nope", false},         // non-http
		{"Drakeposting", "https://x/dup", false}, // duplicate title
		{"One Does Not Simply", "http://x/c", true},
	}
	for _, tc := range cases {
		if got := c.Add(tc.title, tc.url); got != tc.want {
			t.Fatalf("Add(%q,%q)=%v want %v", tc.title, tc.url, got, tc.want)
		}
	}
	if c.Len() != 2 {
		t.Fatalf("want 2 templates got %d: %+v", c.Len(), c.Templates)
	}
}

func cat(titles ...string) Catalog {
	var c Catalog
	for _, t := range titles {
		c.Add(t, "https://x/"+t)
	}
	return c
}

func TestSearchSubstringCaseInsensitive(t *testing.T) {
	c := cat("Distracted Boyfriend", "Drakeposting", "One Does Not Simply")
	got := c.Search("drak", 10)
	if len(got) != 1 || got[0].Title != "Drakeposting" {
		t.Fatalf("want Drakeposting got %+v", got)
	}
}

func TestSearchEmptyReturnsFirstN(t *testing.T) {
	c := cat("a", "b", "c", "d")
	got := c.Search("  ", 2)
	if len(got) != 2 || got[0].Title != "a" || got[1].Title != "b" {
		t.Fatalf("want first 2 got %+v", got)
	}
}

func TestCSVRoundTrip(t *testing.T) {
	c := cat("Drakeposting", "One Does Not Simply")
	var sb strings.Builder
	if err := EncodeCSV(&sb, c.Templates); err != nil {
		t.Fatal(err)
	}
	got, err := DecodeCSV(strings.NewReader(sb.String()))
	if err != nil {
		t.Fatal(err)
	}
	if got.Len() != 2 || got.Templates[0].Title != "Drakeposting" {
		t.Fatalf("round trip lost data: %+v", got.Templates)
	}
	if got.Templates[0].ImageURL != "https://x/Drakeposting" {
		t.Fatalf("bad url %q", got.Templates[0].ImageURL)
	}
}
