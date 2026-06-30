// model.go — bubbletea wiring for the file browser.
// Renders the filtered directory listing, moves the cursor via keys/mouse, and
// resolves a click or Enter into directory navigation or a final image
// selection. All navigation logic lives in browser.go; this file only maps
// terminal events onto it and tracks the done/cancelled result for main.
package browser

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const listTop = 1 // rows above the entry list (title)

// Model is the bubbletea model for picking a base image.
type Model struct {
	state     State
	w, h      int
	top       int // first visible entry index (scroll)
	selected  string
	done      bool
	cancelled bool
	err       string
}

// New builds a browser rooted at dir.
func New(dir string) (Model, error) {
	s, err := Load(dir)
	if err != nil {
		return Model{}, err
	}
	return Model{state: s}, nil
}

// Selected returns the chosen image path (valid once Done).
func (m Model) Selected() string { return m.selected }

// Done reports an image was picked.
func (m Model) Done() bool { return m.done }

// Cancelled reports the user quit without picking.
func (m Model) Cancelled() bool { return m.cancelled }

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

func (m Model) visibleRows() int {
	r := m.h - listTop
	if r < 1 {
		r = 1
	}
	return r
}

// scrollToCursor keeps the cursor inside the visible window.
func (m *Model) scrollToCursor() {
	rows := m.visibleRows()
	if m.state.Cursor < m.top {
		m.top = m.state.Cursor
	}
	if m.state.Cursor >= m.top+rows {
		m.top = m.state.Cursor - rows + 1
	}
	if m.top < 0 {
		m.top = 0
	}
}

// activate acts on the current cursor entry (Enter / click).
func (m Model) activate() (Model, tea.Cmd) {
	ns, sel, err := m.state.Enter()
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	if sel != "" {
		m.selected, m.done = sel, true
		return m, tea.Quit
	}
	m.state = ns
	m.top = 0
	return m, nil
}

// Update satisfies tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.scrollToCursor()
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
		m.state = m.state.MoveTo(m.state.Cursor - 1)
		m.scrollToCursor()
	case tea.KeyDown:
		m.state = m.state.MoveTo(m.state.Cursor + 1)
		m.scrollToCursor()
	case tea.KeyEnter:
		return m.activate()
	case tea.KeyRunes:
		if len(msg.Runes) == 1 {
			switch msg.Runes[0] {
			case 'q':
				m.cancelled = true
				return m, tea.Quit
			case 'k':
				m.state = m.state.MoveTo(m.state.Cursor - 1)
				m.scrollToCursor()
			case 'j':
				m.state = m.state.MoveTo(m.state.Cursor + 1)
				m.scrollToCursor()
			}
		}
	}
	return m, nil
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if m.top > 0 {
			m.top--
		}
		return m, nil
	case tea.MouseButtonWheelDown:
		if m.top < len(m.state.Entries)-1 {
			m.top++
		}
		return m, nil
	}
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	row := msg.Y - listTop
	if row < 0 {
		return m, nil
	}
	idx := m.top + row
	if idx >= len(m.state.Entries) {
		return m, nil
	}
	m.state = m.state.MoveTo(idx)
	return m.activate()
}

var (
	browserTitle = lipgloss.NewStyle().Bold(true)
	cursorRow    = lipgloss.NewStyle().Reverse(true)
	dirStyle     = lipgloss.NewStyle().Bold(true)
)

// View satisfies tea.Model.
func (m Model) View() string {
	var sb strings.Builder
	fmt.Fprintln(&sb, browserTitle.Render("pick an image — "+m.state.Dir))
	rows := m.visibleRows()
	end := m.top + rows
	if end > len(m.state.Entries) {
		end = len(m.state.Entries)
	}
	for i := m.top; i < end; i++ {
		e := m.state.Entries[i]
		line := e.Name
		if e.IsDir {
			line = dirStyle.Render(e.Name)
		}
		if i == m.state.Cursor {
			line = cursorRow.Render("› " + e.Name)
		} else {
			line = "  " + line
		}
		fmt.Fprintln(&sb, line)
	}
	if m.err != "" {
		fmt.Fprintln(&sb, "error: "+m.err)
	}
	return sb.String()
}
