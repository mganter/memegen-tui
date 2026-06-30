package memeio

import (
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
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

func writeSizedPNG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDownscalesLargeImage(t *testing.T) {
	p := filepath.Join(t.TempDir(), "big.png")
	writeSizedPNG(t, p, 3840, 2160) // 4K, 16:9
	img, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	if b.Dx() > 1920 || b.Dy() > 1080 {
		t.Fatalf("not capped to 1080p: %dx%d", b.Dx(), b.Dy())
	}
	// 16:9 capped at 1920x1080 should be exactly that.
	if b.Dx() != 1920 || b.Dy() != 1080 {
		t.Fatalf("want 1920x1080 got %dx%d", b.Dx(), b.Dy())
	}
}

func TestLoadKeepsAspectForTallImage(t *testing.T) {
	p := filepath.Join(t.TempDir(), "tall.png")
	writeSizedPNG(t, p, 2000, 4000) // portrait, capped by height
	b := mustLoad(t, p).Bounds()
	if b.Dy() > 1080 || b.Dx() > 1920 {
		t.Fatalf("not capped: %dx%d", b.Dx(), b.Dy())
	}
	if b.Dy() != 1080 || b.Dx() != 540 { // 2000:4000 → 540:1080
		t.Fatalf("aspect not preserved: want 540x1080 got %dx%d", b.Dx(), b.Dy())
	}
}

func TestLoadLeavesSmallImageUnchanged(t *testing.T) {
	p := filepath.Join(t.TempDir(), "small.png")
	writeSizedPNG(t, p, 800, 600)
	b := mustLoad(t, p).Bounds()
	if b.Dx() != 800 || b.Dy() != 600 {
		t.Fatalf("small image must not be scaled: %dx%d", b.Dx(), b.Dy())
	}
}

func mustLoad(t *testing.T, p string) image.Image {
	t.Helper()
	img, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	return img
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

func TestLoadURL(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 6, 6))
	img.Set(0, 0, color.RGBA{9, 8, 7, 255})
	png, err := EncodePNG(img)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(png)
	}))
	defer srv.Close()

	got, err := LoadURL(srv.URL + "/x.png")
	if err != nil {
		t.Fatal(err)
	}
	if got.Bounds().Dx() != 6 {
		t.Fatalf("bad width %d", got.Bounds().Dx())
	}
}

func TestLoadURLBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()
	if _, err := LoadURL(srv.URL); err == nil {
		t.Fatal("want error on 404")
	}
}

func TestLoadURLNotAnImage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not an image"))
	}))
	defer srv.Close()
	if _, err := LoadURL(srv.URL); err == nil {
		t.Fatal("want decode error")
	}
}

func TestDownloadImage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("\x89PNGfake"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	path, err := DownloadImage(srv.URL+"/some/Cool_Meme.png", dir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "Cool_Meme.png" {
		t.Fatalf("want Cool_Meme.png got %q", path)
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b[:4]) != "\x89PNG" {
		t.Fatalf("image not written: %v %q", err, b)
	}
}

func TestDownloadImageBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()
	if _, err := DownloadImage(srv.URL+"/x.png", t.TempDir()); err == nil {
		t.Fatal("want error on 404")
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
