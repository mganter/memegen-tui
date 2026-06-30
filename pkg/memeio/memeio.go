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
	"os"
	"sync"

	_ "image/gif"  // decode support
	_ "image/jpeg" // decode support

	"golang.design/x/clipboard"
)

// Load decodes an image file (png/jpeg/gif) from path.
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
	return img, nil
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
