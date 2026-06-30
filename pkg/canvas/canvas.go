// canvas.go — meme domain model and pure geometry.
// Holds the base image + text boxes and the box operations (add/remove/move/
// hit-test) the UI drives via mouse. No rendering or IO lives here so the model
// stays trivially testable.

// Package canvas holds the meme domain model: a base image plus positioned
// text boxes. Pure logic, no rendering or IO.
package canvas

import (
	"image"
	"image/color"
)

// Align is horizontal text alignment within a box.
type Align int

const (
	AlignCenter Align = iota
	AlignLeft
	AlignRight
)

// TextBox is one caption placed on the meme, in base-image pixel coordinates.
type TextBox struct {
	X, Y   int // top-left
	W, H   int // size
	Text   string
	FontPt float64
	Color  color.RGBA
	Align  Align
}

// Meme is a base image with zero or more text boxes painted on top.
type Meme struct {
	Base  image.Image
	Boxes []TextBox
}

// New builds a meme over base.
func New(base image.Image) *Meme {
	return &Meme{Base: base}
}

// Bounds returns the base image bounds.
func (m *Meme) Bounds() image.Rectangle { return m.Base.Bounds() }

// AddBox appends b and returns its index. Sane defaults fill zero fields.
func (m *Meme) AddBox(b TextBox) int {
	if b.Color == (color.RGBA{}) {
		b.Color = color.RGBA{255, 255, 255, 255} // white
	}
	if b.FontPt == 0 {
		b.FontPt = 32
	}
	m.Boxes = append(m.Boxes, b)
	return len(m.Boxes) - 1
}

// RemoveBox deletes box i. Out-of-range indexes are ignored.
func (m *Meme) RemoveBox(i int) {
	if i < 0 || i >= len(m.Boxes) {
		return
	}
	m.Boxes = append(m.Boxes[:i], m.Boxes[i+1:]...)
}

// HitTest returns the index of the topmost box covering (x,y), or -1.
func (m *Meme) HitTest(x, y int) int {
	for i := len(m.Boxes) - 1; i >= 0; i-- {
		b := m.Boxes[i]
		if x >= b.X && x < b.X+b.W && y >= b.Y && y < b.Y+b.H {
			return i
		}
	}
	return -1
}

// MoveBox shifts box i by (dx,dy), clamped so it stays inside the base bounds.
func (m *Meme) MoveBox(i, dx, dy int) {
	if i < 0 || i >= len(m.Boxes) {
		return
	}
	bd := m.Bounds()
	b := &m.Boxes[i]
	b.X = clamp(b.X+dx, bd.Min.X, bd.Max.X-b.W)
	b.Y = clamp(b.Y+dy, bd.Min.Y, bd.Max.Y-b.H)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
