package memeio

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func writePNG(t *testing.T, path string, c color.RGBA) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, c)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestLoadPNG(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "in.png")
	writePNG(t, p, color.RGBA{10, 20, 30, 255})
	img, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 8 {
		t.Fatalf("bad width %d", img.Bounds().Dx())
	}
}

func TestLoadMissingFileErrors(t *testing.T) {
	if _, err := Load("/no/such/file.png"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestSavePNGRoundTrip(t *testing.T) {
	dir := t.TempDir()
	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	src.Set(0, 0, color.RGBA{1, 2, 3, 255})
	out := filepath.Join(dir, "out.png")
	if err := SavePNG(out, src); err != nil {
		t.Fatal(err)
	}
	got, err := Load(out)
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := got.At(0, 0).RGBA()
	if r>>8 != 1 || g>>8 != 2 || b>>8 != 3 {
		t.Fatalf("pixel changed: %d,%d,%d", r>>8, g>>8, b>>8)
	}
}

func TestEncodePNGNonEmpty(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	data, err := EncodePNG(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 8 || string(data[1:4]) != "PNG" {
		t.Fatalf("not a PNG, len=%d", len(data))
	}
}
