// model.go — the bubbletea TUI for the meme editor.
// Owns terminal state (size, selection, edit/drag mode), translates mouse and
// key events into canvas operations, and re-burns the meme into a half-block
// preview on every change. Kept thin: all real logic lives in canvas/render/
// preview/memeio so this file is mostly event wiring.
package ui

import (
	"fmt"
	"image"
	"image/color"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mganter/memegen-tui/pkg/canvas"
	"github.com/mganter/memegen-tui/pkg/memeio"
	"github.com/mganter/memegen-tui/pkg/preview"
	"github.com/mganter/memegen-tui/pkg/render"
)

const footerRows = 6

// dragFrame caps how often the preview re-renders during a mouse drag (~30fps).
const dragFrame = 33 * time.Millisecond

// dragTransmitPx is the (smaller) transmitted image width while dragging, so the
// graphics payload stays light; the full-resolution image is restored on drop.
const dragTransmitPx = 480

// selectColor outlines the selected text box in the preview (bright cyan).
var selectColor = color.RGBA{0, 255, 255, 255}

// Model is the bubbletea model for the editor.
type Model struct {
	meme    *canvas.Meme
	outPath string

	w, h  int
	pv    preview.Preview
	pvStr string
	gfx   bool // terminal supports Kitty graphics (real pixels)

	sel      int // selected box index, -1 = none
	editing  bool
	dragging bool
	grabX    int // image-px offset from box origin to grab point
	grabY    int
	lastDraw time.Time // last preview render time, for drag throttling

	status string
}

// New builds a Model over meme, exporting to outPath on save.
func New(meme *canvas.Meme, outPath string) Model {
	return Model{meme: meme, outPath: outPath, sel: -1, status: "click to add text", gfx: preview.GraphicsSupported()}
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
	// Overlay the selected box's boundary on the preview only; the saved/copied
	// meme uses Burn directly and never carries this marker.
	if m.sel >= 0 && m.sel < len(m.meme.Boxes) {
		b := m.meme.Boxes[m.sel]
		thick := img.Bounds().Dx() / 300
		if thick < 2 {
			thick = 2
		}
		img = render.Outline(img, image.Rect(b.X, b.Y, b.X+b.W, b.Y+b.H), selectColor, thick)
	}
	// The Kitty placeholder grid and the half-block grid both drive pv, so the
	// cell↔pixel mapping used for mouse placement matches whatever is displayed.
	// Kitty sizes the grid from the real cell pixel aspect (cells are not a fixed
	// 2:1) so the image is not distorted; half-blocks sample pixels directly.
	if m.gfx {
		b := img.Bounds()
		cw, ch := preview.CellPixels()
		cols, rows := preview.KittyGrid(b.Dx(), b.Dy(), m.w, m.previewRows(), cw, ch)
		m.pv = preview.Grid(b, cols, rows)
		// While dragging, transmit a smaller image so each frame is cheap; the
		// release re-renders at full resolution.
		tx := img
		if m.dragging {
			tx = memeio.ScaleToFit(img, dragTransmitPx, dragTransmitPx)
		}
		m.pvStr = preview.KittyImageDirect(tx, cols, rows)
	} else {
		m.pv = preview.Fit(img.Bounds(), m.w, m.previewRows())
		m.pvStr = m.pv.Render(img)
	}
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
		m.refresh() // show the new box / selection outline immediately
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
		// Throttle: motion events fire far faster than a graphics re-transmit can
		// keep up, so cap drag re-renders. The release does a final full render.
		if time.Since(m.lastDraw) >= dragFrame {
			m.refresh()
			m.lastDraw = time.Now()
		}
		return m, nil
	case tea.MouseActionRelease:
		if m.dragging {
			m.dragging = false
			m.refresh() // final, full-resolution render at the dropped position
		}
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
			m.refresh() // move the selection outline to the new box
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
	case tea.KeyShiftUp:
		return m.nudge(0, -m.meme.Bounds().Dy()) // jump to top (MoveBox clamps)
	case tea.KeyShiftDown:
		return m.nudge(0, m.meme.Bounds().Dy()) // jump to bottom
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
	case '.', '>':
		return m.resizeWidth(1)
	case ',', '<':
		return m.resizeWidth(-1)
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

// resizeWidth widens (dir>0) or narrows (dir<0) the selected box by a step
// proportional to the image width.
func (m Model) resizeWidth(dir int) (tea.Model, tea.Cmd) {
	if m.sel >= 0 {
		step := m.meme.Bounds().Dx() / 20
		if step < 10 {
			step = 10
		}
		m.meme.ResizeBox(m.sel, dir*step)
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
	help := "mouse: click=add/select drag=move · Enter edit · Tab next · +/- size · ,/. width · arrows nudge · shift+↑/↓ top/bottom · d del · s save · c copy · q quit"
	return fmt.Sprintf("%s\n%s\n%s\n%s",
		titleStyle.Render("memegen — "+m.outPath),
		m.pvStr,
		titleStyle.Render(sel),
		statusStyle.Render(m.status+"\n"+help))
}
