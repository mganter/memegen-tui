// imgflip.go — load the imgflip popular-templates catalog.
// Fetches the public api.imgflip.com/get_memes endpoint (the ~100 most popular
// meme templates: name + direct image URL), maps it onto the shared
// memecat.Catalog, and caches it as CSV under the user cache dir so it works
// offline once primed. Parsing is split from IO so the JSON mapping is
// unit-testable without the network.
package imgflip

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mganter/memegen-tui/pkg/memecat"
)

// APIURL is the imgflip endpoint listing popular templates. It is a var so
// tests can point it at a local server.
var APIURL = "https://api.imgflip.com/get_memes"

const cacheFile = "imgflip.csv"

// apiResponse models the get_memes JSON envelope.
type apiResponse struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"error_message"`
	Data         struct {
		Memes []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"memes"`
	} `json:"data"`
}

// ParseMemesJSON maps a get_memes response onto a memecat.Catalog, dropping
// entries with an empty name or URL. It errors if the API reports failure.
func ParseMemesJSON(body []byte) (memecat.Catalog, error) {
	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return memecat.Catalog{}, fmt.Errorf("decode get_memes: %w", err)
	}
	if !resp.Success {
		return memecat.Catalog{}, fmt.Errorf("imgflip api error: %s", resp.ErrorMessage)
	}
	var c memecat.Catalog
	for _, m := range resp.Data.Memes {
		c.Add(m.Name, m.URL)
	}
	return c, nil
}

// Load resolves the user cache dir and loads the catalog from it.
func Load() (memecat.Catalog, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return memecat.Catalog{}, err
	}
	return LoadFrom(filepath.Join(base, "memegen"))
}

// LoadFrom fetches the catalog and caches it as CSV under cacheDir. On a network
// failure it falls back to any cached CSV, else errors.
func LoadFrom(cacheDir string) (memecat.Catalog, error) {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return memecat.Catalog{}, err
	}
	csvPath := filepath.Join(cacheDir, cacheFile)

	resp, err := http.Get(APIURL)
	if err != nil {
		if c, cerr := loadCache(csvPath); cerr == nil {
			return c, nil // offline: use what we have
		}
		return memecat.Catalog{}, fmt.Errorf("fetch imgflip catalog: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if c, cerr := loadCache(csvPath); cerr == nil {
			return c, nil
		}
		return memecat.Catalog{}, fmt.Errorf("fetch imgflip catalog: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return memecat.Catalog{}, err
	}
	c, err := ParseMemesJSON(body)
	if err != nil {
		return memecat.Catalog{}, err
	}
	writeCache(csvPath, c.Templates)
	return c, nil
}

func loadCache(path string) (memecat.Catalog, error) {
	f, err := os.Open(path)
	if err != nil {
		return memecat.Catalog{}, err
	}
	defer f.Close()
	return memecat.DecodeCSV(f)
}

func writeCache(path string, ts []memecat.Template) {
	if f, err := os.Create(path); err == nil {
		defer f.Close()
		memecat.EncodeCSV(f, ts)
	}
}
