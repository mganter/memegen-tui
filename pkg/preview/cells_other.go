//go:build !unix

// cells_other.go — cell pixel size fallback for non-unix platforms.
// The TIOCGWINSZ ioctl is unix-only; elsewhere we assume the conventional 1x2
// cell aspect, the same assumption the half-block renderer makes.
package preview

// CellPixels returns the conventional 1x2 cell size fallback.
func CellPixels() (w, h int) { return 1, 2 }
