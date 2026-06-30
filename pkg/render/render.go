// render.go — burns captions onto the base image.
// Takes a canvas.Meme and draws each text box with a black outline + fill color
// using the bundled gobold font (no external font asset), returning a flattened
// image.Image ready for PNG export. This is the same pipeline the preview and
// the save/copy paths use, so what you see matches what you get.

// Package render burns a meme's text boxes onto its base image, producing a
// flattened image suitable for PNG export.
package render

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"

	"github.com/mganter/memegen-tui/pkg/canvas"
)

var parsedFont = mustParse()

func mustParse() *opentype.Font {
	f, err := opentype.Parse(gobold.TTF)
	if err != nil {
		panic(err) // bundled font; parse failure is a build-time bug
	}
	return f
}

func face(pt float64) (font.Face, error) {
	return opentype.NewFace(parsedFont, &opentype.FaceOptions{
		Size: pt, DPI: 72, Hinting: font.HintingFull,
	})
}

// Burn flattens the meme: copies the base image then draws each text box with a
// black outline and the box's fill color, returning the result.
func Burn(m *canvas.Meme) (image.Image, error) {
	dc := gg.NewContextForImage(m.Base)
	for _, b := range m.Boxes {
		if b.Text == "" {
			continue
		}
		fc, err := face(b.FontPt)
		if err != nil {
			return nil, err
		}
		dc.SetFontFace(fc)

		cx := float64(b.X) + float64(b.W)/2
		cy := float64(b.Y) + float64(b.H)/2
		maxW := float64(b.W)
		const lineSpacing = 1.2

		ax, ay := anchor(b.Align)
		// Outline: draw text in black at 8 offsets around center.
		dc.SetColor(color.Black)
		outline := b.FontPt / 12
		if outline < 1 {
			outline = 1
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				dc.DrawStringWrapped(b.Text,
					cx+float64(dx)*outline, cy+float64(dy)*outline,
					ax, ay, maxW, lineSpacing, ggAlign(b.Align))
			}
		}
		// Fill on top.
		dc.SetColor(b.Color)
		dc.DrawStringWrapped(b.Text, cx, cy, ax, ay, maxW, lineSpacing, ggAlign(b.Align))
	}
	return dc.Image(), nil
}

// Outline returns a copy of src with a thick-pixel rectangle border drawn along
// r in col, clamped to the image bounds. Used to mark the selected text box in
// the preview; it does not touch the saved meme.
func Outline(src image.Image, r image.Rectangle, col color.Color, thick int) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)
	if thick < 1 {
		thick = 1
	}
	r = r.Intersect(b)
	if r.Empty() {
		return dst
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			onBorder := x < r.Min.X+thick || x >= r.Max.X-thick ||
				y < r.Min.Y+thick || y >= r.Max.Y-thick
			if onBorder {
				dst.Set(x, y, col)
			}
		}
	}
	return dst
}

func anchor(a canvas.Align) (ax, ay float64) {
	switch a {
	case canvas.AlignLeft:
		return 0, 0.5
	case canvas.AlignRight:
		return 1, 0.5
	default:
		return 0.5, 0.5
	}
}

func ggAlign(a canvas.Align) gg.Align {
	switch a {
	case canvas.AlignLeft:
		return gg.AlignLeft
	case canvas.AlignRight:
		return gg.AlignRight
	default:
		return gg.AlignCenter
	}
}
