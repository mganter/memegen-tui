package canvas

import (
	"image"
	"image/color"
	"testing"
)

func base(w, h int) image.Image {
	return image.NewRGBA(image.Rect(0, 0, w, h))
}

func TestAddBoxReturnsIndex(t *testing.T) {
	m := New(base(100, 100))
	i0 := m.AddBox(TextBox{X: 10, Y: 10, W: 20, H: 10, Text: "a"})
	i1 := m.AddBox(TextBox{X: 40, Y: 40, W: 20, H: 10, Text: "b"})
	if i0 != 0 || i1 != 1 {
		t.Fatalf("want indexes 0,1 got %d,%d", i0, i1)
	}
	if len(m.Boxes) != 2 {
		t.Fatalf("want 2 boxes got %d", len(m.Boxes))
	}
}

func TestHitTestTopmostWins(t *testing.T) {
	m := New(base(100, 100))
	m.AddBox(TextBox{X: 0, Y: 0, W: 50, H: 50})   // index 0
	m.AddBox(TextBox{X: 10, Y: 10, W: 50, H: 50}) // index 1 overlaps
	if got := m.HitTest(20, 20); got != 1 {
		t.Fatalf("overlap should pick topmost (1), got %d", got)
	}
	if got := m.HitTest(5, 5); got != 0 {
		t.Fatalf("only box0 covers (5,5), got %d", got)
	}
	if got := m.HitTest(90, 90); got != -1 {
		t.Fatalf("empty space should be -1, got %d", got)
	}
}

func TestMoveBoxClampsToBounds(t *testing.T) {
	m := New(base(100, 100))
	i := m.AddBox(TextBox{X: 10, Y: 10, W: 20, H: 20})
	m.MoveBox(i, -50, -50) // drag past top-left
	b := m.Boxes[i]
	if b.X != 0 || b.Y != 0 {
		t.Fatalf("want clamp to 0,0 got %d,%d", b.X, b.Y)
	}
	m.MoveBox(i, 1000, 1000) // drag past bottom-right
	b = m.Boxes[i]
	if b.X != 80 || b.Y != 80 { // 100 - W/H
		t.Fatalf("want clamp to 80,80 got %d,%d", b.X, b.Y)
	}
}

func TestResizeBoxWidth(t *testing.T) {
	m := New(base(200, 100))
	i := m.AddBox(TextBox{X: 50, Y: 10, W: 60, H: 20})
	m.ResizeBox(i, 30) // wider
	if m.Boxes[i].W != 90 {
		t.Fatalf("want W 90 got %d", m.Boxes[i].W)
	}
	m.ResizeBox(i, -50) // narrower
	if m.Boxes[i].W != 40 {
		t.Fatalf("want W 40 got %d", m.Boxes[i].W)
	}
}

func TestResizeBoxClampsMinWidth(t *testing.T) {
	m := New(base(200, 100))
	i := m.AddBox(TextBox{X: 0, Y: 0, W: 60, H: 20})
	m.ResizeBox(i, -1000)
	if m.Boxes[i].W != minBoxWidth {
		t.Fatalf("want clamp to min %d got %d", minBoxWidth, m.Boxes[i].W)
	}
}

func TestResizeBoxKeepsCenter(t *testing.T) {
	m := New(base(200, 100))
	i := m.AddBox(TextBox{X: 70, Y: 0, W: 60, H: 20}) // center x = 100
	m.ResizeBox(i, 40)                                // wider by 40
	b := m.Boxes[i]
	if b.W != 100 {
		t.Fatalf("want W 100 got %d", b.W)
	}
	if b.X+b.W/2 != 100 {
		t.Fatalf("center should stay at 100, got %d (X=%d W=%d)", b.X+b.W/2, b.X, b.W)
	}
}

func TestResizeBoxStaysInBounds(t *testing.T) {
	m := New(base(200, 100))
	i := m.AddBox(TextBox{X: 150, Y: 0, W: 40, H: 20})
	m.ResizeBox(i, 1000) // grow huge
	b := m.Boxes[i]
	if b.W > 200 || b.X < 0 || b.X+b.W > 200 {
		t.Fatalf("box left image bounds: X=%d W=%d", b.X, b.W)
	}
}

func TestResizeBoxOutOfRangeNoop(t *testing.T) {
	m := New(base(100, 100))
	m.AddBox(TextBox{X: 0, Y: 0, W: 30, H: 10})
	m.ResizeBox(99, 10) // no panic, no change
	if m.Boxes[0].W != 30 {
		t.Fatalf("oob resize changed box: %d", m.Boxes[0].W)
	}
}

func TestRemoveBox(t *testing.T) {
	m := New(base(100, 100))
	m.AddBox(TextBox{Text: "a"})
	m.AddBox(TextBox{Text: "b"})
	m.AddBox(TextBox{Text: "c"})
	m.RemoveBox(1)
	if len(m.Boxes) != 2 {
		t.Fatalf("want 2 got %d", len(m.Boxes))
	}
	if m.Boxes[0].Text != "a" || m.Boxes[1].Text != "c" {
		t.Fatalf("wrong remain: %q,%q", m.Boxes[0].Text, m.Boxes[1].Text)
	}
	m.RemoveBox(99) // out of range = no-op, no panic
	if len(m.Boxes) != 2 {
		t.Fatal("oob remove changed slice")
	}
}

func TestNewMemeDefaultColor(t *testing.T) {
	m := New(base(10, 10))
	i := m.AddBox(TextBox{Text: "x"})
	if m.Boxes[i].Color == (color.RGBA{}) {
		t.Fatal("zero box should get default (non-transparent) color")
	}
}
