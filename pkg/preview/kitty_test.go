package preview

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func solid() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 20, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 20; x++ {
			img.Set(x, y, color.RGBA{120, 60, 200, 255})
		}
	}
	return img
}

func TestKittyImageDirectStructure(t *testing.T) {
	out := KittyImageDirect(solid(), 4, 3)
	if !strings.Contains(out, "\x1b_Ga=T") {
		t.Fatal("missing kitty direct placement escape")
	}
	if !strings.Contains(out, "q=2") {
		t.Fatal("must suppress responses with q=2 (else terminal replies corrupt input)")
	}
	if !strings.Contains(out, "C=1") {
		t.Fatal("must not move the cursor (C=1) so the TUI layout stays correct")
	}
	if !strings.Contains(out, "c=4") || !strings.Contains(out, "r=3") {
		t.Fatalf("placement size not declared: %q", firstLine(out))
	}
	// region reservation: 3 rows of 4 spaces each follow the placement escape.
	body := out[strings.LastIndex(out, "\x1b\\")+2:]
	lines := strings.Split(body, "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 reserved rows got %d: %q", len(lines), lines)
	}
	for i, ln := range lines {
		if ln != "    " {
			t.Fatalf("row %d: want 4 spaces got %q", i, ln)
		}
	}
}

func TestKittyGridPreservesAspect(t *testing.T) {
	// Square image, cells 10 wide x 24 tall (taller than 2:1, like Ghostty).
	// A box of cols x rows cells must have ~square pixel dimensions.
	cols, rows := KittyGrid(720, 709, 200, 100, 10, 24)
	boxAspect := float64(cols*10) / float64(rows*24)
	if boxAspect < 0.9 || boxAspect > 1.1 {
		t.Fatalf("box aspect %.2f not ~square for square image (cols=%d rows=%d)", boxAspect, cols, rows)
	}
}

func TestKittyGridFitsWithinBounds(t *testing.T) {
	cols, rows := KittyGrid(720, 709, 80, 40, 9, 18)
	if cols < 1 || rows < 1 || cols > 80 || rows > 40 {
		t.Fatalf("grid out of bounds: %dx%d (max 80x40)", cols, rows)
	}
}

func TestKittyGridLandscapeLimitedByWidth(t *testing.T) {
	// Wide image in a square-ish cell budget should hit the column cap.
	cols, rows := KittyGrid(1600, 400, 80, 80, 10, 20)
	if cols != 80 {
		t.Fatalf("wide image should use full width 80 cols, got %d", cols)
	}
	if rows >= cols {
		t.Fatalf("landscape image should have fewer rows than cols: %dx%d", cols, rows)
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func TestGraphicsSupported(t *testing.T) {
	cases := []struct {
		name              string
		term, prog, kitty string
		noGfx             string
		want              bool
	}{
		{"ghostty term", "xterm-ghostty", "ghostty", "", "", true},
		{"kitty term", "xterm-kitty", "", "", "", true},
		{"kitty window id", "xterm-256color", "", "1", "", true},
		{"plain xterm", "xterm-256color", "", "", "", false},
		{"disabled override", "xterm-ghostty", "ghostty", "", "1", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TERM", tc.term)
			t.Setenv("TERM_PROGRAM", tc.prog)
			t.Setenv("KITTY_WINDOW_ID", tc.kitty)
			t.Setenv("MEMEGEN_NO_GRAPHICS", tc.noGfx)
			if got := GraphicsSupported(); got != tc.want {
				t.Fatalf("GraphicsSupported()=%v want %v", got, tc.want)
			}
		})
	}
}
