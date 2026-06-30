// menu.go — the top-level launch menu shown when no image is passed.
// Lets the user pick an image source: the local file browser or one of the
// online meme template catalogs. It only reports the chosen source; main wires
// each choice to the matching browser. Mirrors the browser/templates list
// idioms (key + mouse navigation) so the three screens feel the same.
package menu

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Source identifies a chosen image source.
const (
	SourceLocal        = "local"
	SourceKnowYourMeme = "knowyourmeme"
	SourceImgflip      = "imgflip"
)

const listTop = 1 // rows above the entry list (title)

type item struct {
	label  string
	source string
}

var items = []item{
	{label: "📁 browse local files", source: SourceLocal},
	{label: "★ browse KnowYourMeme templates", source: SourceKnowYourMeme},
	{label: "★ browse Imgflip templates", source: SourceImgflip},
}

// Model is the bubbletea model for the launch menu.
type Model struct {
	cursor    int
	w, h      int
	selected  string
	done      bool
	cancelled bool
}

// New builds the launch menu.
func New() Model { return Model{} }

// Selected returns the chosen source (valid once Done).
func (m Model) Selected() string { return m.selected }

// Done reports a source was chosen.
func (m Model) Done() bool { return m.done }

// Cancelled reports the user quit without choosing.
func (m Model) Cancelled() bool { return m.cancelled }

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

func (m *Model) move(d int) {
	m.cursor += d
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(items) {
		m.cursor = len(items) - 1
	}
}

func (m Model) pick() (tea.Model, tea.Cmd) {
	m.selected, m.done = items[m.cursor].source, true
	return m, tea.Quit
}

// Update satisfies tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
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
	case tea.KeyUp:
		m.move(-1)
	case tea.KeyDown:
		m.move(1)
	case tea.KeyEnter:
		return m.pick()
	case tea.KeyRunes:
		if len(msg.Runes) == 1 {
			switch msg.Runes[0] {
			case 'q':
				m.cancelled = true
				return m, tea.Quit
			case 'k':
				m.move(-1)
			case 'j':
				m.move(1)
			}
		}
	}
	return m, nil
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	row := msg.Y - listTop
	if row < 0 || row >= len(items) {
		return m, nil
	}
	m.cursor = row
	return m.pick()
}

var (
	menuTitle = lipgloss.NewStyle().Bold(true)
	cursorRow = lipgloss.NewStyle().Reverse(true)
)

// View satisfies tea.Model.
func (m Model) View() string {
	var sb strings.Builder
	fmt.Fprintln(&sb, menuTitle.Render("memegen — pick an image source"))
	for i, it := range items {
		if i == m.cursor {
			fmt.Fprintln(&sb, cursorRow.Render("› "+it.label))
		} else {
			fmt.Fprintln(&sb, "  "+it.label)
		}
	}
	return sb.String()
}
