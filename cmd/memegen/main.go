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
	"github.com/mganter/memegen-tui/internal/menu"
	"github.com/mganter/memegen-tui/internal/templates"
	"github.com/mganter/memegen-tui/internal/ui"
	"github.com/mganter/memegen-tui/pkg/canvas"
	"github.com/mganter/memegen-tui/pkg/imgflip"
	"github.com/mganter/memegen-tui/pkg/kym"
	"github.com/mganter/memegen-tui/pkg/memecat"
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
	fromTemplate := false
	if inPath == "" {
		// No image given: show the launch menu to pick a source — the local file
		// browser or an online meme template catalog.
		source, err := pickSource()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		var picked string
		switch source {
		case "": // cancelled
			return
		case menu.SourceLocal:
			picked, err = pickLocalFile()
		default:
			picked, err = pickTemplate(source)
			fromTemplate = true
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		if picked == "" {
			return // cancelled within the chosen browser
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
		base := inPath
		if fromTemplate {
			// inPath is a cache file; write the meme into the current dir.
			base = filepath.Base(inPath)
		}
		ext := filepath.Ext(base)
		outPath = strings.TrimSuffix(base, ext) + "-meme.png"
	}

	model := ui.New(canvas.New(img), outPath)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// pickSource runs the launch menu and returns the chosen source, or "" if the
// user cancelled.
func pickSource() (string, error) {
	res, err := tea.NewProgram(menu.New(), tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	if err != nil {
		return "", err
	}
	final := res.(menu.Model)
	if final.Cancelled() || !final.Done() {
		return "", nil
	}
	return final.Selected(), nil
}

// pickLocalFile runs the file browser from the current directory and returns the
// chosen image path, or "" if the user cancelled.
func pickLocalFile() (string, error) {
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

// loadCatalog fetches the catalog for the given online template source.
func loadCatalog(source string) (memecat.Catalog, error) {
	switch source {
	case menu.SourceImgflip:
		return imgflip.Load()
	default:
		return kym.Load()
	}
}

// pickTemplate fetches the catalog for source, runs the template browser, and
// downloads the chosen template's image to the cache, returning its local path
// (or "" if the user cancelled).
func pickTemplate(source string) (string, error) {
	fmt.Fprintln(os.Stderr, "fetching meme catalog…")
	cat, err := loadCatalog(source)
	if err != nil {
		return "", fmt.Errorf("load catalog: %w", err)
	}
	tm := templates.New(cat)
	res, err := tea.NewProgram(tm, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	if err != nil {
		return "", err
	}
	final := res.(templates.Model)
	if final.Cancelled() || !final.Done() {
		return "", nil
	}
	sel := final.Selected()
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	fmt.Fprintln(os.Stderr, "downloading", sel.Title, "…")
	return memeio.DownloadImage(sel.ImageURL, filepath.Join(base, "memegen", "templates"))
}
