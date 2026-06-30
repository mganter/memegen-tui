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
	// parent first, then dir, then image files alphabetical; .txt excluded
	want := []string{"..", "sub/", "a.png", "b.JPG", "d.gif"}
	if len(got) != len(want) {
		t.Fatalf("want %v got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("at %d want %q got %q (all %v)", i, want[i], got[i], got)
		}
	}
}

func TestEnterDirLoadsIt(t *testing.T) {
	d := setup(t)
	s, _ := Load(d)
	s = s.MoveTo(1) // "sub/"
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
	s = s.MoveTo(2) // "a.png"
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
	// entry 0 is "..", entering it goes up to d
	ns, sel, err := s.Enter()
	if err != nil {
		t.Fatal(err)
	}
	if sel != "" || ns.Dir != d {
		t.Fatalf("parent nav failed: dir=%q sel=%q", ns.Dir, sel)
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
