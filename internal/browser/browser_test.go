package browser

import (
	"os"
	"path/filepath"
	"testing"
)

func setup(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	mk := func(name string) {
		if err := os.WriteFile(filepath.Join(d, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mk("a.png")
	mk("b.JPG")
	mk("c.txt") // filtered out
	mk("d.gif")
	if err := os.Mkdir(filepath.Join(d, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	return d
}

func names(s State) []string {
	out := make([]string, len(s.Entries))
	for i, e := range s.Entries {
		out[i] = e.Name
	}
	return out
}

func TestLoadFiltersAndOrders(t *testing.T) {
	d := setup(t)
	s, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	got := names(s)
	// two template-source entries first, then parent, dir, image files; .txt excluded
	want := []string{
		templateEntries[0].Name, templateEntries[1].Name,
		"..", "sub/", "a.png", "b.JPG", "d.gif",
	}
	if len(got) != len(want) {
		t.Fatalf("want %v got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("at %d want %q got %q (all %v)", i, want[i], got[i], got)
		}
	}
	if s.Entries[0].Source != SourceKnowYourMeme || s.Entries[1].Source != SourceImgflip {
		t.Fatalf("first two entries should be template sources: %+v %+v", s.Entries[0], s.Entries[1])
	}
}

func TestEnterDirLoadsIt(t *testing.T) {
	d := setup(t)
	s, _ := Load(d)
	s = s.MoveTo(3) // "sub/" (after 2 template entries, "..")
	ns, sel, err := s.Enter()
	if err != nil {
		t.Fatal(err)
	}
	if sel != "" {
		t.Fatalf("dir enter should not select, got %q", sel)
	}
	if ns.Dir != filepath.Join(d, "sub") {
		t.Fatalf("did not enter sub: %q", ns.Dir)
	}
}

func TestEnterImageSelects(t *testing.T) {
	d := setup(t)
	s, _ := Load(d)
	s = s.MoveTo(4) // "a.png" (after 2 template entries, "..", "sub/")
	_, sel, err := s.Enter()
	if err != nil {
		t.Fatal(err)
	}
	if sel != filepath.Join(d, "a.png") {
		t.Fatalf("want a.png path got %q", sel)
	}
}

func TestParentNavigates(t *testing.T) {
	d := setup(t)
	s, _ := Load(filepath.Join(d, "sub"))
	s = s.MoveTo(2) // entries 0,1 are template sources, 2 is ".."
	ns, sel, err := s.Enter()
	if err != nil {
		t.Fatal(err)
	}
	if sel != "" || ns.Dir != d {
		t.Fatalf("parent nav failed: dir=%q sel=%q", ns.Dir, sel)
	}
}

func TestTemplateEntriesPresentAtTop(t *testing.T) {
	d := setup(t)
	s, _ := Load(d)
	if len(s.Entries) < 2 || s.Entries[0].Source == "" || s.Entries[1].Source == "" {
		t.Fatal("expected two template-source entries at the top")
	}
	if s.Entries[0].Path != "" || s.Entries[0].IsDir {
		t.Fatal("template entry should be a non-dir, pathless marker")
	}
}

func TestMoveClamps(t *testing.T) {
	d := setup(t)
	s, _ := Load(d)
	s = s.MoveTo(-5)
	if s.Cursor != 0 {
		t.Fatalf("want 0 got %d", s.Cursor)
	}
	s = s.MoveTo(999)
	if s.Cursor != len(s.Entries)-1 {
		t.Fatalf("want last got %d", s.Cursor)
	}
}
