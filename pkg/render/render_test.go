package render

import (
	"image"
	"image/color"
	"testing"

	"github.com/mganter/memegen-tui/pkg/canvas"
)

// blackBase makes an all-black opaque image so any drawn (white) text pixel
// is detectable.
func blackBase(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{0, 0, 0, 255})
		}
	}
	return img
}

func TestBurnKeepsDimensions(t *testing.T) {
	m := canvas.New(blackBase(200, 120))
	out, err := Burn(m)
	if err != nil {
		t.Fatal(err)
	}
	if out.Bounds().Dx() != 200 || out.Bounds().Dy() != 120 {
		t.Fatalf("dims changed: %v", out.Bounds())
	}
}

func TestBurnDrawsText(t *testing.T) {
	m := canvas.New(blackBase(200, 120))
	m.AddBox(canvas.TextBox{X: 0, Y: 0, W: 200, H: 120, Text: "TOP", FontPt: 48})
	out, err := Burn(m)
	if err != nil {
		t.Fatal(err)
	}
	lit := 0
	b := out.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := out.At(x, y).RGBA()
			if r > 0x8000 && g > 0x8000 && bl > 0x8000 {
				lit++
			}
		}
	}
	if lit == 0 {
		t.Fatal("no white text pixels drawn")
	}
}

func TestOutlineDrawsBorderNotInterior(t *testing.T) {
	src := blackBase(20, 20)
	red := color.RGBA{255, 0, 0, 255}
	out := Outline(src, image.Rect(4, 4, 16, 16), red, 1)

	isRed := func(x, y int) bool {
		r, g, b, _ := out.At(x, y).RGBA()
		return r>>8 == 255 && g>>8 == 0 && b>>8 == 0
	}
	// border pixels red
	if !isRed(4, 4) || !isRed(15, 4) || !isRed(4, 15) || !isRed(15, 15) {
		t.Fatal("corners should be red")
	}
	if !isRed(10, 4) || !isRed(10, 15) || !isRed(4, 10) || !isRed(15, 10) {
		t.Fatal("edge midpoints should be red")
	}
	// interior untouched (still black)
	if isRed(10, 10) {
		t.Fatal("interior should not be drawn")
	}
	// source not mutated
	if r, _, _, _ := src.At(4, 4).RGBA(); r>>8 != 0 {
		t.Fatal("Outline must not mutate src")
	}
}

func TestOutlineClampsToBounds(t *testing.T) {
	src := blackBase(10, 10)
	// rect partly outside; must not panic and stay in-image.
	out := Outline(src, image.Rect(-5, -5, 8, 8), color.RGBA{0, 255, 0, 255}, 2)
	if out.Bounds().Dx() != 10 || out.Bounds().Dy() != 10 {
		t.Fatalf("dims changed: %v", out.Bounds())
	}
}

func TestBurnEmptyTextNoPanic(t *testing.T) {
	m := canvas.New(blackBase(50, 50))
	m.AddBox(canvas.TextBox{X: 0, Y: 0, W: 50, H: 50, Text: ""})
	if _, err := Burn(m); err != nil {
		t.Fatal(err)
	}
}
