package kym

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// zipWith builds an in-memory zip holding one CSV file.
func zipWith(t *testing.T, name, body string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

const etag = `"abc123"`

// catalogServer serves the zip with an ETag and honors conditional requests.
// hits counts the number of full (200) downloads served.
func catalogServer(t *testing.T, zipData []byte, hits *int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", etag)
		*hits++
		w.Write(zipData)
	}))
}

func TestLoadFromDownloadsExtractsCaches(t *testing.T) {
	z := zipWith(t, "knowyourmeme_memes.csv", sampleCSV)
	hits := 0
	srv := catalogServer(t, z, &hits)
	defer srv.Close()
	DatasetURL = srv.URL

	dir := t.TempDir()
	c, err := LoadFrom(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.Len() != 2 {
		t.Fatalf("want 2 templates got %d", c.Len())
	}
	if hits != 1 {
		t.Fatalf("want 1 download got %d", hits)
	}
	if _, err := os.Stat(filepath.Join(dir, "catalog.csv")); err != nil {
		t.Fatalf("catalog not cached: %v", err)
	}
}

func TestLoadFromUsesConditionalCache(t *testing.T) {
	z := zipWith(t, "knowyourmeme_memes.csv", sampleCSV)
	hits := 0
	srv := catalogServer(t, z, &hits)
	defer srv.Close()
	DatasetURL = srv.URL

	dir := t.TempDir()
	if _, err := LoadFrom(dir); err != nil { // primes cache + etag
		t.Fatal(err)
	}
	c, err := LoadFrom(dir) // second call should get 304, reuse cache
	if err != nil {
		t.Fatal(err)
	}
	if c.Len() != 2 {
		t.Fatalf("want 2 templates got %d", c.Len())
	}
	if hits != 1 {
		t.Fatalf("conditional cache miss: want 1 download got %d", hits)
	}
}

func TestLoadFromFallsBackToCacheOffline(t *testing.T) {
	z := zipWith(t, "knowyourmeme_memes.csv", sampleCSV)
	hits := 0
	srv := catalogServer(t, z, &hits)
	DatasetURL = srv.URL

	dir := t.TempDir()
	if _, err := LoadFrom(dir); err != nil { // primes cache
		t.Fatal(err)
	}
	srv.Close()                   // simulate offline
	DatasetURL = "http://127.0.0.1:0/gone"

	c, err := LoadFrom(dir)
	if err != nil {
		t.Fatalf("offline with cache should succeed: %v", err)
	}
	if c.Len() != 2 {
		t.Fatalf("want 2 cached templates got %d", c.Len())
	}
}

func TestLoadFromNoCacheNoNetworkErrors(t *testing.T) {
	DatasetURL = "http://127.0.0.1:0/gone"
	if _, err := LoadFrom(t.TempDir()); err == nil {
		t.Fatal("want error when no cache and network down")
	}
}

