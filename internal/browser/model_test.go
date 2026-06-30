package browser

import (
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func testModel(t *testing.T, dir string) Model {
	t.Helper()
	m, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return nm.(Model)
}

func TestKeyDownMovesCursor(t *testing.T) {
	m := testModel(t, setup(t))
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = nm.(Model)
	if m.state.Cursor != 1 {
		t.Fatalf("want cursor 1 got %d", m.state.Cursor)
	}
}

func TestEnterDirChangesDir(t *testing.T) {
	d := setup(t)
	m := testModel(t, d)
	// entries: 0,1=templates 2=.. 3=sub/ — three downs to reach "sub/"
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = nm.(Model)
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = nm.(Model)
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = nm.(Model)
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.state.Dir != filepath.Join(d, "sub") {
		t.Fatalf("did not enter sub: %q", m.state.Dir)
	}
	if m.Done() {
		t.Fatal("entering dir should not finish")
	}
}

func TestClickImageSelectsAndFinishes(t *testing.T) {
	d := setup(t)
	m := testModel(t, d)
	// rows: y=0 title; list starts y=1. entries: 0,1=templates 2=.. 3=sub/ 4=a.png
	nm, _ := m.Update(tea.MouseMsg{X: 2, Y: 1 + 4, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = nm.(Model)
	if !m.Done() {
		t.Fatal("clicking image should finish")
	}
	if m.Selected() != filepath.Join(d, "a.png") {
		t.Fatalf("want a.png got %q", m.Selected())
	}
}

func TestClickTemplateEntrySignalsSource(t *testing.T) {
	m := testModel(t, setup(t))
	// click second list row (y=2) = the Imgflip template entry
	nm, _ := m.Update(tea.MouseMsg{X: 2, Y: 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = nm.(Model)
	if !m.Templates() {
		t.Fatal("clicking a template entry should signal templates mode")
	}
	if m.Source() != SourceImgflip {
		t.Fatalf("want imgflip source got %q", m.Source())
	}
	if m.Done() {
		t.Fatal("templates signal is not a file selection")
	}
}

func TestQuitCancels(t *testing.T) {
	m := testModel(t, setup(t))
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = nm.(Model)
	if !m.Cancelled() {
		t.Fatal("q should cancel")
	}
	if m.Done() {
		t.Fatal("cancel is not done-with-selection")
	}
}
