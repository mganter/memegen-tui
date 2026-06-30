package kym

import (
	"strings"
	"testing"

	"github.com/mganter/memegen-tui/pkg/memecat"
)

const sampleCSV = `"page_number","title","image_url","is_photo"
"1","Strait of Hormuz","https://i.kym-cdn.com/a.jpg","is_photo"
"1","Foodrot","https://i.kym-cdn.com/b.png","is_photo"
"1","","https://i.kym-cdn.com/empty-title.png","is_photo"
"1","No URL",""," "
"1","Bad URL","ftp://nope.png","x"
"1","Foodrot","https://i.kym-cdn.com/dup.png","x"
`

func TestParseCSVFiltersAndDedups(t *testing.T) {
	c, err := ParseCSV(strings.NewReader(sampleCSV))
	if err != nil {
		t.Fatal(err)
	}
	// empty title, empty url, non-http url, and the duplicate title are dropped.
	want := []memecat.Template{
		{Title: "Strait of Hormuz", ImageURL: "https://i.kym-cdn.com/a.jpg"},
		{Title: "Foodrot", ImageURL: "https://i.kym-cdn.com/b.png"},
	}
	if c.Len() != len(want) {
		t.Fatalf("want %d templates got %d: %+v", len(want), c.Len(), c.Templates)
	}
	for i, w := range want {
		if c.Templates[i] != w {
			t.Fatalf("at %d want %+v got %+v", i, w, c.Templates[i])
		}
	}
}

func TestParseCSVMissingColumns(t *testing.T) {
	if _, err := ParseCSV(strings.NewReader("a,b,c\n1,2,3\n")); err == nil {
		t.Fatal("want error when title/image_url columns absent")
	}
}
