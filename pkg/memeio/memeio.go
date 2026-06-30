// memeio.go — all filesystem and clipboard IO for memegen.
// Loads base images (png/jpeg/gif), encodes/saves the flattened meme as PNG, and
// copies it to the system clipboard. Clipboard init is lazy and its error is
// surfaced (not fatal) because clipboard access depends on the platform/display.

// Package memeio handles image loading, PNG export, and clipboard copy.
package memeio

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	_ "image/gif"  // decode support
	_ "image/jpeg" // decode support

	"golang.design/x/clipboard"
	xdraw "golang.org/x/image/draw"
)

// Max base-image dimensions (1080p). Larger images are downscaled on load so the
// editor, preview, and saved meme stay at a sane resolution.
const (
	maxWidth  = 1920
	maxHeight = 1080
)

// Load decodes an image file (png/jpeg/gif) from path, downscaling it to fit
// within 1080p (preserving aspect) when larger.
func Load(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return ScaleToFit(img, maxWidth, maxHeight), nil
}

// ScaleToFit returns img downscaled to fit within maxW x maxH, preserving
// aspect. Images already within bounds are returned unchanged (no upscaling).
func ScaleToFit(img image.Image, maxW, maxH int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxW && h <= maxH || w <= 0 || h <= 0 {
		return img
	}
	scale := math.Min(float64(maxW)/float64(w), float64(maxH)/float64(h))
	nw, nh := int(math.Round(float64(w)*scale)), int(math.Round(float64(h)*scale))
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Over, nil)
	return dst
}

// LoadURL fetches an image over HTTP and decodes it (png/jpeg/gif), without
// touching disk. Used to preview remote meme templates.
func LoadURL(url string) (image.Image, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: status %d", url, resp.StatusCode)
	}
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", url, err)
	}
	return img, nil
}

// DownloadImage fetches imageURL into dir, naming the file after the URL's base,
// and returns the local path. Used to fetch a chosen meme template for editing.
func DownloadImage(imageURL, dir string) (string, error) {
	resp, err := http.Get(imageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download image: status %d", resp.StatusCode)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := filepath.Base(imageURL)
	if name == "" || name == "." || name == "/" {
		name = "template"
	}
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", err
	}
	return path, nil
}

// EncodePNG encodes img to PNG bytes.
func EncodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// SavePNG writes img to path as PNG.
func SavePNG(path string, img image.Image) error {
	data, err := EncodePNG(img)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

var clipOnce struct {
	sync.Once
	err error
}

// CopyImage copies img to the system clipboard as PNG. Clipboard support is
// platform/display dependent; an error is returned if it is unavailable.
func CopyImage(img image.Image) error {
	clipOnce.Do(func() { clipOnce.err = clipboard.Init() })
	if clipOnce.err != nil {
		return fmt.Errorf("clipboard unavailable: %w", clipOnce.err)
	}
	data, err := EncodePNG(img)
	if err != nil {
		return err
	}
	clipboard.Write(clipboard.FmtImage, data)
	return nil
}
