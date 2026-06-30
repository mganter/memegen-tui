package preview

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func TestFitNativeWhenFits(t *testing.T) {
	p := Fit(image.Rect(0, 0, 200, 100), 1000, 1000)
	// half-block packs 2 px rows per cell, so native = 200 cols x 50 rows.
	if p.Cols != 200 || p.Rows != 50 {
		t.Fatalf("want 200x50 got %dx%d", p.Cols, p.Rows)
	}
}

func TestFitScalesDownToMaxCols(t *testing.T) {
	p := Fit(image.Rect(0, 0, 200, 100), 50, 1000)
	if p.Cols != 50 {
		t.Fatalf("want 50 cols got %d", p.Cols)
	}
	if p.Rows != 13 { // round(50 * 0.25)
		t.Fatalf("want 13 rows got %d", p.Rows)
	}
}

func TestCellImageRoundTrip(t *testing.T) {
	p := Fit(image.Rect(0, 0, 200, 100), 50, 1000)
	for _, c := range []struct{ col, row int }{{0, 0}, {10, 5}, {49, 12}} {
		x, y := p.CellToImage(c.col, c.row)
		gc, gr := p.ImageToCell(x, y)
		if gc != c.col || gr != c.row {
			t.Fatalf("roundtrip cell(%d,%d)->px(%d,%d)->cell(%d,%d)", c.col, c.row, x, y, gc, gr)
		}
	}
}

func TestCellToImageStaysInBounds(t *testing.T) {
	r := image.Rect(0, 0, 200, 100)
	p := Fit(r, 50, 1000)
	x, y := p.CellToImage(p.Cols-1, p.Rows-1)
	if x < 0 || x >= 200 || y < 0 || y >= 100 {
		t.Fatalf("px out of bounds: %d,%d", x, y)
	}
}

func TestGridBuildsExactCellGrid(t *testing.T) {
	p := Grid(image.Rect(0, 0, 720, 709), 30, 18)
	if p.Cols != 30 || p.Rows != 18 {
		t.Fatalf("want 30x18 got %dx%d", p.Cols, p.Rows)
	}
	// cell→pixel mapping must stay in bounds and round-trip.
	x, y := p.CellToImage(29, 17)
	if x < 0 || x >= 720 || y < 0 || y >= 709 {
		t.Fatalf("px out of bounds: %d,%d", x, y)
	}
	if c, r := p.ImageToCell(x, y); c != 29 || r != 17 {
		t.Fatalf("roundtrip failed: got cell %d,%d", c, r)
	}
}

func TestRenderShapeAndContent(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	p := Fit(img.Bounds(), 100, 100) // 4 cols x 2 rows
	s := p.Render(img)
	if p.Cols != 4 || p.Rows != 2 {
		t.Fatalf("want 4x2 got %dx%d", p.Cols, p.Rows)
	}
	if lines := strings.Count(s, "\n"); lines != p.Rows-1 {
		t.Fatalf("want %d newlines got %d", p.Rows-1, lines)
	}
	if !strings.Contains(s, "▀") {
		t.Fatal("expected upper-half-block glyph")
	}
}
