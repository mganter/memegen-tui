// model.go — the bubbletea TUI for the meme editor.
// Owns terminal state (size, selection, edit/drag mode), translates mouse and
// key events into canvas operations, and re-burns the meme into a half-block
// preview on every change. Kept thin: all real logic lives in canvas/render/
// preview/memeio so this file is mostly event wiring.
package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mganter/memegen-tui/pkg/canvas"
	"github.com/mganter/memegen-tui/pkg/memeio"
	"github.com/mganter/memegen-tui/pkg/preview"
	"github.com/mganter/memegen-tui/pkg/render"
)

const footerRows = 6

// Model is the bubbletea model for the editor.
type Model struct {
	meme    *canvas.Meme
	outPath string

	w, h  int
	pv    preview.Preview
	pvStr string

	sel      int // selected box index, -1 = none
	editing  bool
	dragging bool
	grabX    int // image-px offset from box origin to grab point
	grabY    int

	status string
}

// New builds a Model over meme, exporting to outPath on save.
func New(meme *canvas.Meme, outPath string) Model {
	return Model{meme: meme, outPath: outPath, sel: -1, status: "click to add text"}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

func (m Model) previewRows() int {
	r := m.h - footerRows
	if r < 1 {
		r = 1
	}
	return r
}

// refresh re-burns the meme and rebuilds the preview string.
func (m *Model) refresh() {
	if m.w == 0 || m.h == 0 {
		return
	}
	img, err := render.Burn(m.meme)
	if err != nil {
		m.status = "render error: " + err.Error()
		img = m.meme.Base
	}
	m.pv = preview.Fit(img.Bounds(), m.w, m.previewRows())
	m.pvStr = m.pv.Render(img)
}

// mouseToImage converts a terminal cell to an image pixel, reporting whether the
// cell lies inside the preview area.
func (m Model) mouseToImage(x, y int) (ix, iy int, ok bool) {
	col, row := x, y // preview origin is (0,0)
	if col < 0 || col >= m.pv.Cols || row < 0 || row >= m.pv.Rows {
		return 0, 0, false
	}
	ix, iy = m.pv.CellToImage(col, row)
	return ix, iy, true
}

// Update satisfies tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.refresh()
		return m, nil
	case tea.MouseMsg:
		return m.handleMouse(msg)
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			return m, nil
		}
		ix, iy, ok := m.mouseToImage(msg.X, msg.Y)
		if !ok {
			return m, nil
		}
		if hit := m.meme.HitTest(ix, iy); hit >= 0 {
			m.sel = hit
		} else {
			m.sel = m.newBoxAt(ix, iy)
		}
		b := m.meme.Boxes[m.sel]
		m.grabX, m.grabY = ix-b.X, iy-b.Y
		m.dragging = true
		m.editing = false
		m.status = "drag to move · Enter to edit"
		return m, nil
	case tea.MouseActionMotion:
		if !m.dragging || m.sel < 0 {
			return m, nil
		}
		ix, iy, ok := m.mouseToImage(msg.X, msg.Y)
		if !ok {
			return m, nil
		}
		b := m.meme.Boxes[m.sel]
		m.meme.MoveBox(m.sel, (ix-m.grabX)-b.X, (iy-m.grabY)-b.Y)
		m.refresh()
		return m, nil
	case tea.MouseActionRelease:
		m.dragging = false
		return m, nil
	}
	return m, nil
}

// newBoxAt creates a default caption centered on (ix,iy) and returns its index.
func (m *Model) newBoxAt(ix, iy int) int {
	bd := m.meme.Bounds()
	w := bd.Dx() / 2
	h := bd.Dy() / 5
	return m.meme.AddBox(canvas.TextBox{
		X: ix - w/2, Y: iy - h/2, W: w, H: h,
		Text: "TEXT", FontPt: float64(h) * 0.6,
	})
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.editing {
		return m.handleEditKey(msg)
	}
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEnter:
		if m.sel >= 0 {
			m.editing = true
			m.status = "editing · Esc when done"
		}
		return m, nil
	case tea.KeyTab:
		if n := len(m.meme.Boxes); n > 0 {
			m.sel = (m.sel + 1) % n
		}
		return m, nil
	case tea.KeyUp:
		return m.nudge(0, -2)
	case tea.KeyDown:
		return m.nudge(0, 2)
	case tea.KeyLeft:
		return m.nudge(-2, 0)
	case tea.KeyRight:
		return m.nudge(2, 0)
	case tea.KeyRunes:
		return m.handleCommand(msg.Runes)
	}
	return m, nil
}

func (m Model) handleCommand(runes []rune) (tea.Model, tea.Cmd) {
	if len(runes) != 1 {
		return m, nil
	}
	switch runes[0] {
	case 'q':
		return m, tea.Quit
	case 'n':
		bd := m.meme.Bounds()
		m.sel = m.newBoxAt(bd.Dx()/2, bd.Dy()/2)
		m.refresh()
	case 'd':
		if m.sel >= 0 {
			m.meme.RemoveBox(m.sel)
			m.sel = -1
			m.editing = false
			m.refresh()
		}
	case '+', '=':
		return m.scaleFont(2)
	case '-', '_':
		return m.scaleFont(-2)
	case 's':
		return m.save()
	case 'c':
		return m.copy()
	}
	return m, nil
}

func (m Model) handleEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.sel < 0 {
		m.editing = false
		return m, nil
	}
	b := &m.meme.Boxes[m.sel]
	switch msg.Type {
	case tea.KeyEsc, tea.KeyEnter:
		m.editing = false
		m.status = "click to add text · s save · c copy · q quit"
	case tea.KeyBackspace:
		if r := []rune(b.Text); len(r) > 0 {
			b.Text = string(r[:len(r)-1])
			m.refresh()
		}
	case tea.KeySpace:
		b.Text += " "
		m.refresh()
	case tea.KeyRunes:
		b.Text += string(msg.Runes)
		m.refresh()
	}
	return m, nil
}

func (m Model) nudge(dx, dy int) (tea.Model, tea.Cmd) {
	if m.sel >= 0 {
		m.meme.MoveBox(m.sel, dx, dy)
		m.refresh()
	}
	return m, nil
}

func (m Model) scaleFont(d float64) (tea.Model, tea.Cmd) {
	if m.sel >= 0 {
		b := &m.meme.Boxes[m.sel]
		b.FontPt += d
		if b.FontPt < 6 {
			b.FontPt = 6
		}
		m.refresh()
	}
	return m, nil
}

func (m Model) save() (tea.Model, tea.Cmd) {
	img, err := render.Burn(m.meme)
	if err != nil {
		m.status = "render error: " + err.Error()
		return m, nil
	}
	if err := memeio.SavePNG(m.outPath, img); err != nil {
		m.status = "save failed: " + err.Error()
		return m, nil
	}
	m.status = "saved → " + m.outPath
	return m, nil
}

func (m Model) copy() (tea.Model, tea.Cmd) {
	img, err := render.Burn(m.meme)
	if err != nil {
		m.status = "render error: " + err.Error()
		return m, nil
	}
	if err := memeio.CopyImage(img); err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.status = "copied to clipboard"
	return m, nil
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true)
	statusStyle = lipgloss.NewStyle().Faint(true)
)

// View satisfies tea.Model.
func (m Model) View() string {
	if m.pvStr == "" {
		return "loading…"
	}
	sel := "none"
	if m.sel >= 0 && m.sel < len(m.meme.Boxes) {
		b := m.meme.Boxes[m.sel]
		mode := "selected"
		if m.editing {
			mode = "EDITING"
		}
		sel = fmt.Sprintf("#%d %q  pos(%d,%d) %dx%d  %0.fpt  [%s]",
			m.sel, b.Text, b.X, b.Y, b.W, b.H, b.FontPt, mode)
	}
	help := "mouse: click=add/select drag=move · Enter edit · Tab next · +/- size · arrows nudge · d del · s save · c copy · q quit"
	return fmt.Sprintf("%s\n%s\n%s\n%s",
		titleStyle.Render("memegen — "+m.outPath),
		m.pvStr,
		titleStyle.Render(sel),
		statusStyle.Render(m.status+"\n"+help))
}
