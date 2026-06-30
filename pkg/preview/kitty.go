// kitty.go — high-quality image preview via the Kitty graphics protocol.
// When the terminal supports it (kitty, Ghostty), the preview is drawn as real
// pixels instead of Unicode half-blocks. The image is placed with a direct
// placement (a=T) scaled to a cell grid, the cursor is left in place (C=1), and
// the region is reserved with a grid of spaces so the TUI renderer lays it out
// correctly. (Unicode-placeholder placement, the other Kitty method, renders
// distorted in Ghostty's alternate screen, which the TUI uses.)
package preview

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"

	xdraw "golang.org/x/image/draw"
)

// imageID is the graphics id reused for the single preview image; re-placing
// with the same id replaces it, so previews do not accumulate in the terminal.
const imageID = 1

// maxTransmitWidth caps the transmitted image's width so the base64 payload
// stays small; the terminal scales it into the cell grid regardless.
const maxTransmitWidth = 640

// GraphicsSupported reports whether the terminal can render the Kitty graphics
// protocol. It is conservative (kitty and Ghostty only) and can be force-
// disabled with MEMEGEN_NO_GRAPHICS.
func GraphicsSupported() bool {
	if os.Getenv("MEMEGEN_NO_GRAPHICS") != "" {
		return false
	}
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}
	term := os.Getenv("TERM")
	if strings.Contains(term, "kitty") || strings.Contains(term, "ghostty") {
		return true
	}
	return os.Getenv("TERM_PROGRAM") == "ghostty"
}

// KittyGrid returns the cols x rows cell grid that displays an imgW x imgH image
// without aspect distortion, given the terminal's cell pixel size (cellW x
// cellH), fitted within maxCols x maxRows. Unlike half-block Fit, this accounts
// for the real cell aspect: the terminal scales the image into the cell box, so
// cols*cellW : rows*cellH must match the image aspect.
func KittyGrid(imgW, imgH, maxCols, maxRows, cellW, cellH int) (int, int) {
	if imgW <= 0 || imgH <= 0 || cellW <= 0 || cellH <= 0 {
		return 1, 1
	}
	// Desired columns per row to preserve aspect.
	colsPerRow := float64(imgW*cellH) / float64(imgH*cellW)
	// Try full width, then clamp by height.
	cols := float64(maxCols)
	rows := cols / colsPerRow
	if rows > float64(maxRows) {
		rows = float64(maxRows)
		cols = rows * colsPerRow
	}
	c, r := int(cols+0.5), int(rows+0.5)
	if c < 1 {
		c = 1
	}
	if r < 1 {
		r = 1
	}
	return c, r
}

// KittyImageDirect returns a TUI-ready block: a direct image placement (cursor
// not moved) followed by a cols x rows grid of spaces that reserves the region
// in the renderer's layout. The image is drawn over those cells and fills the
// block.
func KittyImageDirect(img image.Image, cols, rows int) string {
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	var sb strings.Builder
	// Delete the previous placement of this id first: successive previews differ
	// in size, so a new (smaller) image would not fully cover the old one and
	// stale pixels would tear through.
	fmt.Fprintf(&sb, "\x1b_Ga=d,d=i,i=%d,q=2\x1b\\", imageID)
	sb.WriteString(placement(scaleForTransmit(img), cols, rows))
	for r := 0; r < rows; r++ {
		sb.WriteString(strings.Repeat(" ", cols))
		if r < rows-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// placement builds the chunked Kitty direct-placement escape: transmit and
// display img scaled to cols x rows cells, leaving the cursor in place (C=1).
// q=2 suppresses the terminal's acknowledgement so it is not read back as input.
func placement(img image.Image, cols, rows int) string {
	var buf bytes.Buffer
	png.Encode(&buf, img)
	data := base64.StdEncoding.EncodeToString(buf.Bytes())

	const chunkSize = 4096
	var sb strings.Builder
	first := true
	for len(data) > 0 {
		n := chunkSize
		if n > len(data) {
			n = len(data)
		}
		chunk := data[:n]
		data = data[n:]
		more := 0
		if len(data) > 0 {
			more = 1
		}
		if first {
			fmt.Fprintf(&sb, "\x1b_Ga=T,q=2,f=100,C=1,i=%d,c=%d,r=%d,m=%d;%s\x1b\\",
				imageID, cols, rows, more, chunk)
			first = false
		} else {
			fmt.Fprintf(&sb, "\x1b_Gm=%d;%s\x1b\\", more, chunk)
		}
	}
	return sb.String()
}

// scaleForTransmit downscales img to at most maxTransmitWidth, preserving
// aspect, to bound the transmitted payload. Smaller images pass through.
func scaleForTransmit(img image.Image) image.Image {
	b := img.Bounds()
	if b.Dx() <= maxTransmitWidth {
		return img
	}
	w := maxTransmitWidth
	h := b.Dy() * w / b.Dx()
	if h < 1 {
		h = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Over, nil)
	return dst
}
