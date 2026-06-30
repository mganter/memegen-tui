package templates

import (
	"image"
	"image/color"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mganter/memegen-tui/pkg/memecat"
)

var errTest = errImg("boom")

type errImg string

func (e errImg) Error() string { return string(e) }

func swatch() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.RGBA{200, 100, 50, 255})
		}
	}
	return img
}

func cat() memecat.Catalog {
	return memecat.Catalog{Templates: []memecat.Template{
		{Title: "Distracted Boyfriend", ImageURL: "https://x/a.jpg"},
		{Title: "Drakeposting", ImageURL: "https://x/b.jpg"},
		{Title: "One Does Not Simply", ImageURL: "https://x/c.jpg"},
	}}
}

func sized(c memecat.Catalog) Model {
	m := New(c)
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return nm.(Model)
}

func typeStr(t *testing.T, m Model, s string) Model {
	t.Helper()
	for _, r := range s {
		nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = nm.(Model)
	}
	return m
}

func TestTypingFiltersResults(t *testing.T) {
	m := sized(cat())
	m = typeStr(t, m, "drak")
	if len(m.results) != 1 || m.results[0].Title != "Drakeposting" {
		t.Fatalf("want only Drakeposting got %+v", m.results)
	}
}

func TestEmptyQueryShowsAll(t *testing.T) {
	m := sized(cat())
	if len(m.results) != 3 {
		t.Fatalf("want 3 results got %d", len(m.results))
	}
}

func TestDownMovesCursor(t *testing.T) {
	m := sized(cat())
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = nm.(Model)
	if m.cursor != 1 {
		t.Fatalf("want cursor 1 got %d", m.cursor)
	}
}

func TestEnterSelects(t *testing.T) {
	m := sized(cat())
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = nm.(Model)
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if !m.Done() {
		t.Fatal("enter should finish")
	}
	if m.Selected().Title != "Drakeposting" {
		t.Fatalf("want Drakeposting got %q", m.Selected().Title)
	}
}

func TestBackspaceWidensResults(t *testing.T) {
	m := sized(cat())
	m = typeStr(t, m, "drak")
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = nm.(Model)
	// "dra" still matches only Drakeposting; remove more to widen
	m = typeStr(t, m, "")
	for i := 0; i < 3; i++ {
		nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		m = nm.(Model)
	}
	if len(m.results) != 3 {
		t.Fatalf("want all 3 after clearing query got %d", len(m.results))
	}
}

func TestEscCancels(t *testing.T) {
	m := sized(cat())
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = nm.(Model)
	if !m.Cancelled() {
		t.Fatal("esc should cancel")
	}
	if m.Done() {
		t.Fatal("cancel is not a selection")
	}
}

func TestMoveTriggersImageLoadWhenUncached(t *testing.T) {
	m := sized(cat())
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if cmd == nil {
		t.Fatal("moving onto an uncached template should kick off an image load")
	}
}

func TestImgLoadedRendersPreviewForCurrent(t *testing.T) {
	m := sized(cat())
	url := m.results[m.cursor].ImageURL
	nm, _ := m.Update(imgLoadedMsg{url: url, img: swatch()})
	m = nm.(Model)
	if m.pvStr == "" {
		t.Fatal("preview should render after the current image loads")
	}
	if _, ok := m.imgCache[url]; !ok {
		t.Fatal("loaded image should be cached")
	}
}

func TestCachedSelectionDoesNotReload(t *testing.T) {
	m := sized(cat())
	// load image for the second template, then move onto it: no new load cmd.
	url := m.results[1].ImageURL
	nm, _ := m.Update(imgLoadedMsg{url: url, img: swatch()})
	m = nm.(Model)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown}) // cursor 0 → 1
	if cmd != nil {
		t.Fatal("moving onto a cached template should not reload")
	}
}

func TestImgLoadErrorDoesNotPanicOrCache(t *testing.T) {
	m := sized(cat())
	url := m.results[m.cursor].ImageURL
	nm, _ := m.Update(imgLoadedMsg{url: url, err: errTest})
	m = nm.(Model)
	if _, ok := m.imgCache[url]; ok {
		t.Fatal("failed load should not be cached")
	}
}

func TestGraphicsPreviewUsesKitty(t *testing.T) {
	t.Setenv("MEMEGEN_NO_GRAPHICS", "")
	t.Setenv("TERM", "xterm-ghostty")
	m := sized(cat())
	if !m.gfx {
		t.Fatal("expected graphics enabled under ghostty TERM")
	}
	url := m.results[m.cursor].ImageURL
	nm, _ := m.Update(imgLoadedMsg{url: url, img: swatch()})
	m = nm.(Model)
	if !strings.Contains(m.pvStr, "\x1b_G") {
		t.Fatal("graphics preview should emit a Kitty transmit escape")
	}
	if !strings.Contains(m.View(), "\x1b_G") {
		t.Fatal("View should include the graphics preview")
	}
}

func TestHalfBlockPreviewWhenNoGraphics(t *testing.T) {
	t.Setenv("MEMEGEN_NO_GRAPHICS", "1")
	m := sized(cat())
	if m.gfx {
		t.Fatal("graphics must be disabled by MEMEGEN_NO_GRAPHICS")
	}
	url := m.results[m.cursor].ImageURL
	nm, _ := m.Update(imgLoadedMsg{url: url, img: swatch()})
	m = nm.(Model)
	if strings.Contains(m.pvStr, "\x1b_G") {
		t.Fatal("half-block preview must not emit graphics escapes")
	}
	if !strings.Contains(m.pvStr, "▀") {
		t.Fatal("expected half-block preview")
	}
}

func TestClickSelects(t *testing.T) {
	m := sized(cat())
	// row layout: y=0 search line, y=1 hint, list starts at listTop. Click first row.
	nm, _ := m.Update(tea.MouseMsg{X: 2, Y: listTop, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = nm.(Model)
	if !m.Done() || m.Selected().Title != "Distracted Boyfriend" {
		t.Fatalf("click first row should select Distracted Boyfriend; done=%v sel=%q", m.Done(), m.Selected().Title)
	}
}
