package ui

import (
	"image"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mganter/memegen-tui/pkg/canvas"
)

func newTestModel() Model {
	base := image.NewRGBA(image.Rect(0, 0, 400, 200))
	m := New(canvas.New(base), "out.png")
	// give it a terminal size so preview exists
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	return nm.(Model)
}

func TestWindowSizeBuildsPreview(t *testing.T) {
	m := newTestModel()
	if m.pv.Cols == 0 {
		t.Fatal("preview not built on resize")
	}
}

func TestLeftPressEmptyCreatesAndSelectsBox(t *testing.T) {
	m := newTestModel()
	nm, _ := m.Update(tea.MouseMsg{X: 10, Y: 5, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = nm.(Model)
	if len(m.meme.Boxes) != 1 {
		t.Fatalf("want 1 box got %d", len(m.meme.Boxes))
	}
	if m.sel != 0 {
		t.Fatalf("want sel 0 got %d", m.sel)
	}
	if !m.dragging {
		t.Fatal("press should begin drag")
	}
}

func TestDragMovesSelectedBox(t *testing.T) {
	m := newTestModel()
	// create at far left
	nm, _ := m.Update(tea.MouseMsg{X: 2, Y: 10, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = nm.(Model)
	startX := m.meme.Boxes[0].X
	// drag right
	nm, _ = m.Update(tea.MouseMsg{X: 60, Y: 10, Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft})
	m = nm.(Model)
	if m.meme.Boxes[0].X <= startX {
		t.Fatalf("drag right should increase X: start %d now %d", startX, m.meme.Boxes[0].X)
	}
	// release
	nm, _ = m.Update(tea.MouseMsg{X: 60, Y: 10, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	m = nm.(Model)
	if m.dragging {
		t.Fatal("release should stop drag")
	}
}

func TestEnterTogglesEditingAndRunesAppend(t *testing.T) {
	m := newTestModel()
	nm, _ := m.Update(tea.MouseMsg{X: 10, Y: 5, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = nm.(Model)
	m.meme.Boxes[0].Text = "" // start empty
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if !m.editing {
		t.Fatal("enter should enter editing")
	}
	for _, r := range "HI" {
		nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = nm.(Model)
	}
	if m.meme.Boxes[0].Text != "HI" {
		t.Fatalf("want HI got %q", m.meme.Boxes[0].Text)
	}
	// backspace
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = nm.(Model)
	if m.meme.Boxes[0].Text != "H" {
		t.Fatalf("backspace failed: %q", m.meme.Boxes[0].Text)
	}
	// esc exits editing
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = nm.(Model)
	if m.editing {
		t.Fatal("esc should exit editing")
	}
}

func TestDeleteRemovesSelected(t *testing.T) {
	m := newTestModel()
	nm, _ := m.Update(tea.MouseMsg{X: 10, Y: 5, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = nm.(Model)
	nm, _ = m.Update(tea.MouseMsg{X: 10, Y: 5, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	m = nm.(Model)
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = nm.(Model)
	if len(m.meme.Boxes) != 0 {
		t.Fatalf("delete failed, %d boxes", len(m.meme.Boxes))
	}
	if m.sel != -1 {
		t.Fatalf("sel should reset to -1 got %d", m.sel)
	}
}

func TestTypingNotEditingIsIgnored(t *testing.T) {
	m := newTestModel()
	nm, _ := m.Update(tea.MouseMsg{X: 10, Y: 5, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = nm.(Model)
	nm, _ = m.Update(tea.MouseMsg{X: 10, Y: 5, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	m = nm.(Model)
	before := m.meme.Boxes[0].Text
	// 'x' is not a command and we're not editing -> text unchanged
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = nm.(Model)
	if m.meme.Boxes[0].Text != before {
		t.Fatalf("non-editing key mutated text: %q", m.meme.Boxes[0].Text)
	}
}
