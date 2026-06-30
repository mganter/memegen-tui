package imgflip

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const sampleJSON = `{"success":true,"data":{"memes":[
{"id":"1","name":"Drake Hotline Bling","url":"https://i.imgflip.com/30b1gx.jpg","box_count":2},
{"id":"2","name":"Two Buttons","url":"https://i.imgflip.com/1g8my4.jpg","box_count":3},
{"id":"3","name":"","url":"https://i.imgflip.com/x.jpg","box_count":1},
{"id":"4","name":"No URL","url":"","box_count":1}
]}}`

func TestParseMemesJSONFilters(t *testing.T) {
	c, err := ParseMemesJSON([]byte(sampleJSON))
	if err != nil {
		t.Fatal(err)
	}
	// empty name and empty url dropped.
	if c.Len() != 2 {
		t.Fatalf("want 2 got %d: %+v", c.Len(), c.Templates)
	}
	if c.Templates[0].Title != "Drake Hotline Bling" || c.Templates[0].ImageURL != "https://i.imgflip.com/30b1gx.jpg" {
		t.Fatalf("bad first template: %+v", c.Templates[0])
	}
}

func TestParseMemesJSONApiFailure(t *testing.T) {
	if _, err := ParseMemesJSON([]byte(`{"success":false,"error_message":"boom"}`)); err == nil {
		t.Fatal("want error when success=false")
	}
}

func TestLoadFromFetchesAndCaches(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Write([]byte(sampleJSON))
	}))
	defer srv.Close()
	APIURL = srv.URL

	dir := t.TempDir()
	c, err := LoadFrom(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.Len() != 2 {
		t.Fatalf("want 2 got %d", c.Len())
	}
	if _, err := os.Stat(filepath.Join(dir, "imgflip.csv")); err != nil {
		t.Fatalf("catalog not cached: %v", err)
	}
	if hits != 1 {
		t.Fatalf("want 1 fetch got %d", hits)
	}
}

func TestLoadFromFallsBackToCacheOffline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleJSON))
	}))
	APIURL = srv.URL
	dir := t.TempDir()
	if _, err := LoadFrom(dir); err != nil { // prime cache
		t.Fatal(err)
	}
	srv.Close()
	APIURL = "http://127.0.0.1:0/gone"

	c, err := LoadFrom(dir)
	if err != nil {
		t.Fatalf("offline with cache should succeed: %v", err)
	}
	if c.Len() != 2 {
		t.Fatalf("want 2 cached got %d", c.Len())
	}
}

func TestLoadFromNoCacheNoNetworkErrors(t *testing.T) {
	APIURL = "http://127.0.0.1:0/gone"
	if _, err := LoadFrom(t.TempDir()); err == nil {
		t.Fatal("want error when no cache and network down")
	}
}
