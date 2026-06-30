package menu

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func sized() Model {
	m := New()
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return nm.(Model)
}

func TestFirstEntryIsLocalFiles(t *testing.T) {
	m := sized()
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if !m.Done() {
		t.Fatal("enter should finish")
	}
	if m.Selected() != SourceLocal {
		t.Fatalf("first entry should be local files, got %q", m.Selected())
	}
}

func TestDownThenEnterSelectsTemplateSource(t *testing.T) {
	m := sized()
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = nm.(Model)
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.Selected() != SourceKnowYourMeme {
		t.Fatalf("want knowyourmeme got %q", m.Selected())
	}
}

func TestClickSelects(t *testing.T) {
	m := sized()
	// title at y=0, list starts at y=1; click third row (Imgflip).
	nm, _ := m.Update(tea.MouseMsg{X: 2, Y: 1 + 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = nm.(Model)
	if !m.Done() || m.Selected() != SourceImgflip {
		t.Fatalf("click row 3 should select imgflip; done=%v sel=%q", m.Done(), m.Selected())
	}
}

func TestEscCancels(t *testing.T) {
	m := sized()
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = nm.(Model)
	if !m.Cancelled() {
		t.Fatal("esc should cancel")
	}
	if m.Done() {
		t.Fatal("cancel is not a selection")
	}
}

func TestCursorClamps(t *testing.T) {
	m := sized()
	for i := 0; i < 10; i++ { // run past the end
		nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = nm.(Model)
	}
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.Selected() != SourceImgflip { // last entry
		t.Fatalf("cursor should clamp to last entry, got %q", m.Selected())
	}
}
