//go:build unix

// cells_unix.go — terminal cell pixel size via the TIOCGWINSZ ioctl.
// The window size struct reports the window's pixel dimensions alongside its
// row/column count, so cell width/height in pixels is xpixel/cols and
// ypixel/rows. This is what lets the Kitty renderer choose a cell grid that
// preserves image aspect (cells are not a fixed 2:1). Falls back to 1x2 when the
// terminal does not report pixel sizes.
package preview

import (
	"os"

	"golang.org/x/sys/unix"
)

// CellPixels returns the terminal cell size in pixels (width, height). When the
// terminal does not report pixel dimensions it returns the conventional 1x2
// fallback, which matches the half-block assumption.
func CellPixels() (w, h int) {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err == nil && ws.Col > 0 && ws.Row > 0 && ws.Xpixel > 0 && ws.Ypixel > 0 {
		return int(ws.Xpixel) / int(ws.Col), int(ws.Ypixel) / int(ws.Row)
	}
	return 1, 2
}
