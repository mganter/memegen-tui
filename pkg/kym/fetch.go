// fetch.go — network + cache layer for the KnowYourMeme catalog.
// Downloads the dataset zip from Kaggle, extracts the CSV, and caches it under
// the user cache dir. Re-uses the cache via HTTP conditional requests: the
// ETag/Last-Modified from the last download are replayed, so an unchanged
// dataset returns 304 and the cached CSV is parsed without re-downloading. If
// the network is unreachable but a cache exists, the cache is used (offline).
package kym

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mganter/memegen-tui/pkg/memecat"
)

// DatasetURL is the Kaggle public download endpoint for the catalog zip.
// It is a var so tests can point it at a local server.
var DatasetURL = "https://www.kaggle.com/api/v1/datasets/download/adityaaggarwal27/knowyourmeme-memes"

const (
	catalogFile = "catalog.csv"
	metaFile    = "catalog.meta.json"
)

// cacheMeta records the validators from the last successful download so the
// next request can be made conditional.
type cacheMeta struct {
	ETag         string `json:"etag"`
	LastModified string `json:"last_modified"`
}

// Load resolves the user cache dir and loads the catalog from it.
func Load() (memecat.Catalog, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return memecat.Catalog{}, err
	}
	return LoadFrom(filepath.Join(base, "memegen"))
}

// LoadFrom loads the catalog using cacheDir for persistence. It issues a
// conditional GET when a cache exists: 304 reuses the cached CSV, 200 refreshes
// it. On network failure it falls back to any cached CSV, else errors.
func LoadFrom(cacheDir string) (memecat.Catalog, error) {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return memecat.Catalog{}, err
	}
	csvPath := filepath.Join(cacheDir, catalogFile)
	meta := readMeta(filepath.Join(cacheDir, metaFile))
	_, cacheErr := os.Stat(csvPath)
	haveCache := cacheErr == nil

	req, err := http.NewRequest(http.MethodGet, DatasetURL, nil)
	if err != nil {
		return memecat.Catalog{}, err
	}
	if haveCache {
		if meta.ETag != "" {
			req.Header.Set("If-None-Match", meta.ETag)
		}
		if meta.LastModified != "" {
			req.Header.Set("If-Modified-Since", meta.LastModified)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if haveCache {
			return parseFile(csvPath) // offline: use what we have
		}
		return memecat.Catalog{}, fmt.Errorf("download catalog: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotModified && haveCache:
		return parseFile(csvPath)
	case resp.StatusCode == http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return memecat.Catalog{}, err
		}
		csv, err := extractCSV(body)
		if err != nil {
			return memecat.Catalog{}, err
		}
		if err := os.WriteFile(csvPath, csv, 0o644); err != nil {
			return memecat.Catalog{}, err
		}
		writeMeta(filepath.Join(cacheDir, metaFile), cacheMeta{
			ETag:         resp.Header.Get("ETag"),
			LastModified: resp.Header.Get("Last-Modified"),
		})
		return ParseCSV(bytes.NewReader(csv))
	case haveCache:
		return parseFile(csvPath) // unexpected status, but cache covers us
	default:
		return memecat.Catalog{}, fmt.Errorf("download catalog: status %d", resp.StatusCode)
	}
}

// extractCSV returns the first .csv file's bytes from a zip archive.
func extractCSV(zipData []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("open catalog zip: %w", err)
	}
	for _, f := range zr.File {
		if filepath.Ext(f.Name) != ".csv" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(rc)
	}
	return nil, fmt.Errorf("no csv in catalog zip")
}

func parseFile(path string) (memecat.Catalog, error) {
	f, err := os.Open(path)
	if err != nil {
		return memecat.Catalog{}, err
	}
	defer f.Close()
	return ParseCSV(f)
}

func readMeta(path string) cacheMeta {
	var m cacheMeta
	if b, err := os.ReadFile(path); err == nil {
		json.Unmarshal(b, &m)
	}
	return m
}

func writeMeta(path string, m cacheMeta) {
	if b, err := json.Marshal(m); err == nil {
		os.WriteFile(path, b, 0o644)
	}
}
