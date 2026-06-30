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
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown}) // to "sub/"
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
	// rows: y=0 title; list starts y=1. entries: 0=.. 1=sub/ 2=a.png
	nm, _ := m.Update(tea.MouseMsg{X: 2, Y: 1 + 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = nm.(Model)
	if !m.Done() {
		t.Fatal("clicking image should finish")
	}
	if m.Selected() != filepath.Join(d, "a.png") {
		t.Fatalf("want a.png got %q", m.Selected())
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
