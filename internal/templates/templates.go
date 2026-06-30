// templates.go — bubbletea browser for the online meme template catalog.
// Shows a type-to-filter search over meme titles fetched from the KnowYourMeme
// dataset; up/down or a click moves the highlight, Enter or a click picks a
// template. All filtering delegates to memecat.Catalog.Search; this file only maps
// terminal events onto it and tracks the chosen template / cancelled result for
// main, which then downloads the picked image and opens the editor.
package templates

import (
	"fmt"
	"image"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mganter/memegen-tui/pkg/memecat"
	"github.com/mganter/memegen-tui/pkg/memeio"
	"github.com/mganter/memegen-tui/pkg/preview"
)

const (
	listTop    = 2   // rows above the result list (search line + hint)
	maxResults = 500 // cap on filtered results held in memory
	listWidth  = 38  // columns reserved for the result list; rest is preview
)

// imgLoadedMsg reports the outcome of an async preview-image fetch.
type imgLoadedMsg struct {
	url string
	img image.Image
	err error
}

// Model is the bubbletea model for picking a meme template.
type Model struct {
	cat       memecat.Catalog
	query     string
	results   []memecat.Template
	cursor    int
	top       int // first visible result index (scroll)
	w, h      int
	selected  memecat.Template
	done      bool
	cancelled bool

	imgCache map[string]image.Image // decoded preview images keyed by URL
	pvStr    string                 // rendered preview of the current selection
	gfx      bool                   // terminal supports Kitty graphics (real pixels)
}

// New builds a template browser over cat, showing all entries until filtered.
func New(cat memecat.Catalog) Model {
	m := Model{cat: cat, imgCache: map[string]image.Image{}, gfx: preview.GraphicsSupported()}
	m.refilter()
	return m
}

// Selected returns the chosen template (valid once Done).
func (m Model) Selected() memecat.Template { return m.selected }

// Done reports a template was picked.
func (m Model) Done() bool { return m.done }

// Cancelled reports the user quit without picking.
func (m Model) Cancelled() bool { return m.cancelled }

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// refilter recomputes the visible results for the current query and clamps the
// cursor/scroll to the new list.
func (m *Model) refilter() {
	m.results = m.cat.Search(m.query, maxResults)
	if m.cursor >= len(m.results) {
		m.cursor = len(m.results) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.scrollToCursor()
}

func (m Model) visibleRows() int {
	r := m.h - listTop
	if r < 1 {
		r = 1
	}
	return r
}

func (m *Model) scrollToCursor() {
	rows := m.visibleRows()
	if m.cursor < m.top {
		m.top = m.cursor
	}
	if m.cursor >= m.top+rows {
		m.top = m.cursor - rows + 1
	}
	if m.top < 0 {
		m.top = 0
	}
}

func (m *Model) move(d int) {
	m.cursor += d
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.results) {
		m.cursor = len(m.results) - 1
	}
	m.scrollToCursor()
}

// currentURL returns the highlighted template's image URL, or "" if none.
func (m Model) currentURL() string {
	if m.cursor < 0 || m.cursor >= len(m.results) {
		return ""
	}
	return m.results[m.cursor].ImageURL
}

// previewCols/previewRows are the cell dimensions of the preview pane.
func (m Model) previewCols() int {
	c := m.w - listWidth - 1
	if c < 1 {
		c = 1
	}
	return c
}

func (m Model) previewRows() int {
	r := m.h - listTop
	if r < 1 {
		r = 1
	}
	return r
}

// renderPreview rebuilds pvStr from img fitted to the preview pane, using real
// pixels via the Kitty graphics protocol when the terminal supports it, else
// Unicode half-blocks.
func (m *Model) renderPreview(img image.Image) {
	if m.w == 0 || m.h == 0 {
		return
	}
	if m.gfx {
		b := img.Bounds()
		cw, ch := preview.CellPixels()
		cols, rows := preview.KittyGrid(b.Dx(), b.Dy(), m.previewCols(), m.previewRows(), cw, ch)
		m.pvStr = preview.KittyImageDirect(img, cols, rows)
		return
	}
	pv := preview.Fit(img.Bounds(), m.previewCols(), m.previewRows())
	m.pvStr = pv.Render(img)
}

// syncPreview shows the cached preview for the current selection, or returns a
// command to fetch it. The pane is cleared when there is nothing to show.
func (m *Model) syncPreview() tea.Cmd {
	url := m.currentURL()
	if url == "" || m.w == 0 || m.h == 0 {
		m.pvStr = ""
		return nil
	}
	if img, ok := m.imgCache[url]; ok {
		m.renderPreview(img)
		return nil
	}
	m.pvStr = "loading preview…"
	return loadImage(url)
}

// loadImage fetches and decodes a preview image off the UI goroutine.
func loadImage(url string) tea.Cmd {
	return func() tea.Msg {
		img, err := memeio.LoadURL(url)
		return imgLoadedMsg{url: url, img: img, err: err}
	}
}

// pick finalizes the highlighted result as the selection.
func (m Model) pick() (tea.Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.results) {
		return m, nil
	}
	m.selected, m.done = m.results[m.cursor], true
	return m, tea.Quit
}

// Update satisfies tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.scrollToCursor()
		return m, m.syncPreview()
	case imgLoadedMsg:
		if msg.err == nil && msg.img != nil {
			m.imgCache[msg.url] = msg.img
			if msg.url == m.currentURL() {
				m.renderPreview(msg.img)
			}
		} else if msg.url == m.currentURL() {
			m.pvStr = "preview unavailable"
		}
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.MouseMsg:
		return m.handleMouse(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.cancelled = true
		return m, tea.Quit
	case tea.KeyEnter:
		return m.pick()
	case tea.KeyUp:
		m.move(-1)
	case tea.KeyDown:
		m.move(1)
	case tea.KeyBackspace:
		if r := []rune(m.query); len(r) > 0 {
			m.query = string(r[:len(r)-1])
			m.refilter()
		}
	case tea.KeySpace:
		m.query += " "
		m.refilter()
	case tea.KeyRunes:
		m.query += string(msg.Runes)
		m.refilter()
	}
	return m, m.syncPreview()
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.move(-1)
		return m, m.syncPreview()
	case tea.MouseButtonWheelDown:
		m.move(1)
		return m, m.syncPreview()
	}
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	row := msg.Y - listTop
	if row < 0 {
		return m, nil
	}
	idx := m.top + row
	if idx >= len(m.results) {
		return m, nil
	}
	m.cursor = idx
	return m.pick()
}

var (
	searchStyle = lipgloss.NewStyle().Bold(true)
	hintStyle   = lipgloss.NewStyle().Faint(true)
	cursorRow   = lipgloss.NewStyle().Reverse(true)
)

// View satisfies tea.Model.
func (m Model) View() string {
	var sb strings.Builder
	fmt.Fprintln(&sb, searchStyle.Render("search memes: "+m.query+"▏"))
	fmt.Fprintln(&sb, hintStyle.Render(fmt.Sprintf("%d matches · type to filter · ↑↓ move · Enter pick · Esc back", len(m.results))))
	rows := m.visibleRows()
	end := m.top + rows
	if end > len(m.results) {
		end = len(m.results)
	}
	for i := m.top; i < end; i++ {
		line := truncate(m.results[i].Title, listWidth-2)
		if i == m.cursor {
			line = cursorRow.Render("› " + line)
		} else {
			line = "  " + line
		}
		fmt.Fprintln(&sb, line)
	}
	list := sb.String()
	if m.pvStr == "" {
		return list
	}
	if m.gfx {
		return composeGraphics(list, m.pvStr)
	}
	left := lipgloss.NewStyle().Width(listWidth).Render(list)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, m.pvStr)
}

// composeGraphics lays the list beside a Kitty-graphics preview by hand. lipgloss
// cannot measure the graphics escape (the base64 payload has no display width),
// so each line is padded to listWidth from the visible width and the preview
// rows — which carry the escapes — are appended at the end of each line. The
// preview block is offset by listTop so its image aligns with the result rows.
func composeGraphics(list, img string) string {
	left := strings.Split(list, "\n")
	right := append(make([]string, listTop), strings.Split(img, "\n")...)
	n := max(len(left), len(right))
	var sb strings.Builder
	for i := 0; i < n; i++ {
		var l, r string
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		pad := listWidth - lipgloss.Width(l)
		if pad < 0 {
			pad = 0
		}
		sb.WriteString(l + strings.Repeat(" ", pad) + " " + r)
		if i < n-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// truncate shortens s to at most n display runes, adding an ellipsis.
func truncate(s string, n int) string {
	r := []rune(s)
	if n < 1 || len(r) <= n {
		return s
	}
	if n == 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}
