package state_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/ambershark-mike/gwsm/internal/state"
)

// newStore is a test helper that creates a Store in t.TempDir().
func newStore(t *testing.T) *state.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state.json")
	s, err := state.New(path)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	return s
}

// storePath returns the file path the Store was constructed with.
// We duplicate the path construction here rather than exporting an accessor.
func storePath(t *testing.T) string {
	return filepath.Join(t.TempDir(), "state.json")
}

// --------------------------------------------------------------------------
// TestStore_SaveAndLoad
// --------------------------------------------------------------------------

func TestStore_SaveAndLoad(t *testing.T) {
	s := newStore(t)

	want := state.Profile{
		Name: "work",
		Windows: []state.SavedWindow{
			{
				WmClass:     "firefox",
				WmClassInst: "Navigator",
				Index:       0,
				X:           100, Y: 200,
				Width: 1280, Height: 800,
				Workspace: 1,
				Monitor:   0,
				Maximized: 0,
			},
		},
	}

	if err := s.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := s.Load("work")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.Name != want.Name {
		t.Errorf("Name: got %q, want %q", got.Name, want.Name)
	}
	if len(got.Windows) != 1 {
		t.Fatalf("Windows len: got %d, want 1", len(got.Windows))
	}
	w := got.Windows[0]
	sw := want.Windows[0]
	if w.WmClass != sw.WmClass || w.WmClassInst != sw.WmClassInst ||
		w.X != sw.X || w.Y != sw.Y || w.Width != sw.Width || w.Height != sw.Height ||
		w.Workspace != sw.Workspace || w.Monitor != sw.Monitor || w.Maximized != sw.Maximized {
		t.Errorf("window mismatch:\n  got  %+v\n  want %+v", w, sw)
	}
}

// --------------------------------------------------------------------------
// TestStore_List
// --------------------------------------------------------------------------

func TestStore_List(t *testing.T) {
	s := newStore(t)

	names := []string{"zebra", "alpha", "mango"}
	for _, n := range names {
		if err := s.Save(state.Profile{Name: n}); err != nil {
			t.Fatalf("Save %q: %v", n, err)
		}
	}

	got, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	want := []string{"alpha", "mango", "zebra"}
	if !slices.Equal(got, want) {
		t.Errorf("List: got %v, want %v", got, want)
	}
}

// --------------------------------------------------------------------------
// TestStore_Delete
// --------------------------------------------------------------------------

func TestStore_Delete(t *testing.T) {
	s := newStore(t)

	if err := s.Save(state.Profile{Name: "temp"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := s.Delete("temp"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Load("temp"); err == nil {
		t.Fatal("Load after Delete: expected error, got nil")
	}
}

// --------------------------------------------------------------------------
// TestStore_Delete_NotFound
// --------------------------------------------------------------------------

func TestStore_Delete_NotFound(t *testing.T) {
	s := newStore(t)
	if err := s.Delete("nonexistent"); err == nil {
		t.Fatal("expected error when deleting nonexistent profile")
	}
}

// --------------------------------------------------------------------------
// TestStore_Load_NotFound
// --------------------------------------------------------------------------

func TestStore_Load_NotFound(t *testing.T) {
	s := newStore(t)
	if _, err := s.Load("ghost"); err == nil {
		t.Fatal("expected error when loading nonexistent profile")
	}
}

// --------------------------------------------------------------------------
// TestStore_AtomicWrite
// --------------------------------------------------------------------------

func TestStore_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	s, err := state.New(path)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}

	if err := s.Save(state.Profile{Name: "atomic"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// The final file must exist and be valid JSON.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var check any
	if err := json.Unmarshal(data, &check); err != nil {
		t.Fatalf("state file is not valid JSON: %v", err)
	}

	// The temporary file must not be left behind.
	tmp := path + ".tmp"
	if _, err := os.Stat(tmp); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("temp file %q should not exist after successful write", tmp)
	}
}

// --------------------------------------------------------------------------
// TestStore_MultipleWindows
// --------------------------------------------------------------------------

func TestStore_MultipleWindows(t *testing.T) {
	s := newStore(t)

	windows := []state.SavedWindow{
		{WmClass: "firefox", WmClassInst: "Navigator", Index: 0, X: 0, Y: 0, Width: 1920, Height: 1080, Workspace: 0, Monitor: 0},
		{WmClass: "code", WmClassInst: "code", Index: 0, X: 1920, Y: 0, Width: 1920, Height: 1080, Workspace: 0, Monitor: 1},
		{WmClass: "terminal", WmClassInst: "terminal", TitlePattern: `bash`, Index: 0, X: 0, Y: 1080, Width: 960, Height: 540, Workspace: 0, Monitor: 0, Maximized: 3},
	}
	want := state.Profile{Name: "multi", Windows: windows}

	if err := s.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := s.Load("multi")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(got.Windows) != len(want.Windows) {
		t.Fatalf("Windows len: got %d, want %d", len(got.Windows), len(want.Windows))
	}
	for i := range want.Windows {
		gw, ww := got.Windows[i], want.Windows[i]
		if gw != ww {
			t.Errorf("window[%d]:\n  got  %+v\n  want %+v", i, gw, ww)
		}
	}
}

// --------------------------------------------------------------------------
// TestStore_New_CreatesDirs
// --------------------------------------------------------------------------

func TestStore_New_CreatesDirs(t *testing.T) {
	dir := t.TempDir()
	// Nest several levels deep to confirm MkdirAll is used.
	path := filepath.Join(dir, "a", "b", "c", "state.json")
	s, err := state.New(path)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	if err := s.Save(state.Profile{Name: "x"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("state file not created: %v", err)
	}
}

// --------------------------------------------------------------------------
// TestStore_RemoveWindows
// --------------------------------------------------------------------------

func TestStore_RemoveWindows_AllEntriesForClass(t *testing.T) {
	s := newStore(t)
	s.Save(state.Profile{
		Name: "p",
		Windows: []state.SavedWindow{
			{WmClass: "ghostty", WmClassInst: "ghostty", Index: 0},
			{WmClass: "ghostty", WmClassInst: "ghostty", Index: 1},
			{WmClass: "zen", WmClassInst: "zen", Index: 0},
		},
	})

	if err := s.RemoveWindows("p", "ghostty", "ghostty", -1); err != nil {
		t.Fatalf("RemoveWindows: %v", err)
	}

	p, _ := s.Load("p")
	if len(p.Windows) != 1 {
		t.Fatalf("expected 1 window remaining, got %d", len(p.Windows))
	}
	if p.Windows[0].WmClass != "zen" {
		t.Errorf("expected remaining window to be zen, got %q", p.Windows[0].WmClass)
	}
}

func TestStore_RemoveWindows_SpecificIndex(t *testing.T) {
	s := newStore(t)
	s.Save(state.Profile{
		Name: "p",
		Windows: []state.SavedWindow{
			{WmClass: "ghostty", WmClassInst: "ghostty", Index: 0, X: 0},
			{WmClass: "ghostty", WmClassInst: "ghostty", Index: 1, X: 1000},
		},
	})

	if err := s.RemoveWindows("p", "ghostty", "ghostty", 0); err != nil {
		t.Fatalf("RemoveWindows: %v", err)
	}

	p, _ := s.Load("p")
	if len(p.Windows) != 1 {
		t.Fatalf("expected 1 window remaining, got %d", len(p.Windows))
	}
	// The old index-1 entry should be re-indexed to 0.
	if p.Windows[0].Index != 0 {
		t.Errorf("expected re-indexed to 0, got %d", p.Windows[0].Index)
	}
	if p.Windows[0].X != 1000 {
		t.Errorf("expected X=1000 (old index 1), got %d", p.Windows[0].X)
	}
}

func TestStore_RemoveWindows_NotFound(t *testing.T) {
	s := newStore(t)
	s.Save(state.Profile{Name: "p", Windows: []state.SavedWindow{
		{WmClass: "ghostty", WmClassInst: "ghostty", Index: 0},
	}})

	if err := s.RemoveWindows("p", "zen", "zen", -1); err == nil {
		t.Fatal("expected error for nonexistent class, got nil")
	}
}

func TestStore_RemoveWindows_ProfileNotFound(t *testing.T) {
	s := newStore(t)
	if err := s.RemoveWindows("missing", "ghostty", "ghostty", -1); err == nil {
		t.Fatal("expected error for nonexistent profile, got nil")
	}
}

// --------------------------------------------------------------------------
// TestStore_Load_MissingFile_NoError
// --------------------------------------------------------------------------

func TestStore_Load_MissingFile_NoError(t *testing.T) {
	// New on a non-existent file is fine; Load on missing profile returns
	// a "not found" error, not an I/O error.
	path := filepath.Join(t.TempDir(), "missing.json")
	s, err := state.New(path)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	_, err = s.Load("anything")
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
	// The file itself should still not exist (we didn't create it).
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("state file should not exist yet; Stat returned %v", statErr)
	}
}
