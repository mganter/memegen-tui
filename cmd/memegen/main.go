// main.go — entry point for the memegen CLI.
// Parses the base image path and output flag, loads the image, and launches the
// bubbletea TUI editor with mouse tracking and the alternate screen buffer.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mganter/memegen-tui/internal/browser"
	"github.com/mganter/memegen-tui/internal/ui"
	"github.com/mganter/memegen-tui/pkg/canvas"
	"github.com/mganter/memegen-tui/pkg/memeio"
)

func main() {
	out := flag.String("o", "", "output PNG path (default: <input>-meme.png)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [-o out.png] <image>\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(2)
	}

	inPath := flag.Arg(0)
	if inPath == "" {
		// No image given: open the file browser to pick one.
		picked, err := pickImage()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		if picked == "" {
			return // cancelled
		}
		inPath = picked
	}

	img, err := memeio.Load(inPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	outPath := *out
	if outPath == "" {
		ext := filepath.Ext(inPath)
		outPath = strings.TrimSuffix(inPath, ext) + "-meme.png"
	}

	model := ui.New(canvas.New(img), outPath)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// pickImage runs the file browser from the current directory and returns the
// chosen image path, or "" if the user cancelled.
func pickImage() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	bm, err := browser.New(cwd)
	if err != nil {
		return "", err
	}
	res, err := tea.NewProgram(bm, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	if err != nil {
		return "", err
	}
	final := res.(browser.Model)
	if final.Cancelled() || !final.Done() {
		return "", nil
	}
	return final.Selected(), nil
}
