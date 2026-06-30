// preview.go — image → terminal half-block renderer + coordinate mapping.
// Fits an image into a cell grid (each ▀ cell = 2 stacked pixels, top=fg,
// bottom=bg via ANSI truecolor) and converts between terminal cells and image
// pixels. The cell↔pixel map is what lets the UI translate a mouse click into a
// caption position, so it is the most important piece to keep correct.

// Package preview renders an image as a grid of Unicode upper-half-block cells
// using ANSI truecolor, and maps between terminal cells and image pixels so the
// UI can place captions by mouse.
package preview

import (
	"fmt"
	"image"
	"math"
	"strings"
)

// Preview describes a fitted rendering of an image bounds into a cell grid.
// Each cell stacks two vertical image pixels (top = foreground, bottom =
// background) via the ▀ glyph.
type Preview struct {
	Cols, Rows     int
	srcW, srcH     int
	scaleX, scaleY float64 // image px per cell column / row
}

// Fit computes a Preview that scales the image bounds to fit within maxCols x
// maxRows, preserving aspect (2 image rows per cell row), never upscaling.
func Fit(b image.Rectangle, maxCols, maxRows int) Preview {
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return Preview{Cols: 0, Rows: 0, srcW: w, srcH: h}
	}
	// Native size in cells: cols=w, rows=ceil(h/2).
	scale := math.Min(float64(maxCols)/float64(w), float64(maxRows)/(float64(h)/2))
	if scale > 1 {
		scale = 1
	}
	cols := int(math.Round(float64(w) * scale))
	rows := int(math.Round(float64(h) / 2 * scale))
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	return Preview{
		Cols: cols, Rows: rows, srcW: w, srcH: h,
		scaleX: float64(w) / float64(cols),
		scaleY: float64(h) / float64(rows),
	}
}

// Grid builds a Preview over an explicit cols x rows cell grid (no aspect math).
// Used for graphics rendering where the grid is chosen separately from cell
// pixel dimensions; the cell↔pixel mapping then matches the displayed image.
func Grid(b image.Rectangle, cols, rows int) Preview {
	w, h := b.Dx(), b.Dy()
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	return Preview{
		Cols: cols, Rows: rows, srcW: w, srcH: h,
		scaleX: float64(w) / float64(cols),
		scaleY: float64(h) / float64(rows),
	}
}

// CellToImage maps a terminal cell to the center image pixel it covers.
func (p Preview) CellToImage(col, row int) (x, y int) {
	x = int((float64(col) + 0.5) * p.scaleX)
	y = int((float64(row) + 0.5) * p.scaleY)
	return clamp(x, 0, p.srcW-1), clamp(y, 0, p.srcH-1)
}

// ImageToCell maps an image pixel to its terminal cell.
func (p Preview) ImageToCell(x, y int) (col, row int) {
	col = int(float64(x) / p.scaleX)
	row = int(float64(y) / p.scaleY)
	return clamp(col, 0, p.Cols-1), clamp(row, 0, p.Rows-1)
}

// Render produces the ANSI half-block string for img.
func (p Preview) Render(img image.Image) string {
	var sb strings.Builder
	b := img.Bounds()
	for row := 0; row < p.Rows; row++ {
		for col := 0; col < p.Cols; col++ {
			// Top pixel (foreground) and bottom pixel (background).
			tx, ty := p.sample(col, row, 0)
			bx, by := p.sample(col, row, 1)
			tr, tg, tb := rgb(img.At(b.Min.X+tx, b.Min.Y+ty))
			br, bg, bb := rgb(img.At(b.Min.X+bx, b.Min.Y+by))
			fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀", tr, tg, tb, br, bg, bb)
		}
		sb.WriteString("\x1b[0m")
		if row < p.Rows-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// sample returns the image pixel for the given cell and vertical half (0=top,
// 1=bottom).
func (p Preview) sample(col, row, half int) (x, y int) {
	x = int((float64(col) + 0.5) * p.scaleX)
	yf := (float64(row) + (float64(half)+0.5)/2) * p.scaleY
	return clamp(x, 0, p.srcW-1), clamp(int(yf), 0, p.srcH-1)
}

func rgb(c interface{ RGBA() (r, g, b, a uint32) }) (uint8, uint8, uint8) {
	r, g, b, _ := c.RGBA()
	return uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
