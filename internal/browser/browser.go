// browser.go — pure file-browser state for picking a base image.
// Lists a directory filtered to sub-directories and supported image files,
// tracks cursor position, and resolves Enter into either navigation (dir) or a
// selected image path. No terminal/IO concerns beyond reading the directory, so
// navigation is fully unit-testable; the bubbletea wiring lives in model.go.
package browser

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// imageExts are the base-image formats memegen can load.
var imageExts = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true}

// Template-source identifiers for the synthetic browser entries that open an
// online catalog instead of selecting a local file.
const (
	SourceKnowYourMeme = "knowyourmeme"
	SourceImgflip      = "imgflip"
)

// templateEntries are the synthetic rows shown at the top of every directory,
// each opening a different online meme template catalog.
var templateEntries = []Entry{
	{Name: "★ browse KnowYourMeme templates", Source: SourceKnowYourMeme},
	{Name: "★ browse Imgflip templates", Source: SourceImgflip},
}

// Entry is one row in the browser: a parent link, sub-directory, image file, or
// a synthetic template-source entry (Source set, no Path).
type Entry struct {
	Name   string // display name; dirs end with "/", parent is ".."
	Path   string // absolute path
	IsDir  bool
	Source string // non-empty on a synthetic entry: the online catalog to open
}

// State is the browser at one directory.
type State struct {
	Dir     string
	Entries []Entry
	Cursor  int
}

// Load reads dir and builds the filtered, ordered entry list: the template
// source entries first, then parent (".."), then sub-directories, then image
// files, each group alphabetical.
func Load(dir string) (State, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return State{}, err
	}
	items, err := os.ReadDir(abs)
	if err != nil {
		return State{}, err
	}
	var dirs, files []Entry
	for _, it := range items {
		name := it.Name()
		if strings.HasPrefix(name, ".") {
			continue // skip hidden
		}
		full := filepath.Join(abs, name)
		if it.IsDir() {
			dirs = append(dirs, Entry{Name: name + "/", Path: full, IsDir: true})
		} else if imageExts[strings.ToLower(filepath.Ext(name))] {
			files = append(files, Entry{Name: name, Path: full})
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	entries := append([]Entry{}, templateEntries...)
	if parent := filepath.Dir(abs); parent != abs {
		entries = append(entries, Entry{Name: "..", Path: parent, IsDir: true})
	}
	entries = append(entries, dirs...)
	entries = append(entries, files...)
	return State{Dir: abs, Entries: entries}, nil
}

// MoveTo sets the cursor, clamped to the entry range.
func (s State) MoveTo(i int) State {
	if i < 0 {
		i = 0
	}
	if i >= len(s.Entries) {
		i = len(s.Entries) - 1
	}
	if i < 0 {
		i = 0
	}
	s.Cursor = i
	return s
}

// Enter acts on the cursor entry: navigate into a directory (returns new State,
// empty selection) or select an image file (returns its path).
func (s State) Enter() (State, string, error) {
	if len(s.Entries) == 0 {
		return s, "", nil
	}
	e := s.Entries[s.Cursor]
	if e.IsDir {
		ns, err := Load(e.Path)
		return ns, "", err
	}
	return s, e.Path, nil
}
