package ui

import (
	"image"
	"image/color"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mganter/memegen-tui/pkg/canvas"
	"github.com/mganter/memegen-tui/pkg/memeio"
)

// TestEndToEndSave drives the full pipeline through the model: size → click to
// add box → type text → save, then verifies a PNG with white text landed.
func TestEndToEndSave(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 300, 150))
	for y := 0; y < 150; y++ {
		for x := 0; x < 300; x++ {
			base.Set(x, y, color.RGBA{0, 0, 0, 255})
		}
	}
	out := filepath.Join(t.TempDir(), "meme.png")
	var m tea.Model = New(canvas.New(base), out)

	steps := []tea.Msg{
		tea.WindowSizeMsg{Width: 80, Height: 40},
		tea.MouseMsg{X: 20, Y: 10, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft},
		tea.MouseMsg{X: 20, Y: 10, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft},
		tea.KeyMsg{Type: tea.KeyEnter}, // edit
		tea.KeyMsg{Type: tea.KeyBackspace}, tea.KeyMsg{Type: tea.KeyBackspace},
		tea.KeyMsg{Type: tea.KeyBackspace}, tea.KeyMsg{Type: tea.KeyBackspace}, // clear "TEXT"
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("YO")},
		tea.KeyMsg{Type: tea.KeyEsc},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}, // save
	}
	for _, s := range steps {
		m, _ = m.Update(s)
	}

	img, err := memeio.Load(out)
	if err != nil {
		t.Fatalf("expected saved meme: %v", err)
	}
	lit := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			if r > 0x8000 && g > 0x8000 && bl > 0x8000 {
				lit++
			}
		}
	}
	if lit == 0 {
		t.Fatal("saved meme has no text pixels")
	}
}
