package match_test

import (
	"testing"

	"github.com/ambershark-mike/gwsm/internal/dbus"
	"github.com/ambershark-mike/gwsm/internal/match"
)

// helpers ---------------------------------------------------------------

func makeWindow(id uint32, class, instance string) dbus.WindowInfo {
	return dbus.WindowInfo{ID: id, WmClass: class, WmClassInstance: instance}
}

// windowIDs extracts IDs from a slice for easy comparison in assertions.
func windowIDs(windows []dbus.WindowInfo) []uint32 {
	ids := make([]uint32, len(windows))
	for i, w := range windows {
		ids[i] = w.ID
	}
	return ids
}

// Match tests ------------------------------------------------------------

func TestMatch_ExactClass(t *testing.T) {
	windows := []dbus.WindowInfo{
		makeWindow(1, "firefox", "Navigator"),
		makeWindow(2, "code", "code"),
		makeWindow(3, "firefox", "Navigator"),
	}
	key := match.WindowKey{WmClass: "firefox"}
	got := match.Match(key, windows)
	if len(got) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(got))
	}
	for _, w := range got {
		if w.WmClass != "firefox" {
			t.Errorf("unexpected WmClass %q", w.WmClass)
		}
	}
}

func TestMatch_InstanceFilter_NonEmpty(t *testing.T) {
	windows := []dbus.WindowInfo{
		makeWindow(1, "firefox", "Navigator"),
		makeWindow(2, "firefox", "Toolkit"),
		makeWindow(3, "firefox", "Navigator"),
	}
	key := match.WindowKey{WmClass: "firefox", WmClassInstance: "Toolkit"}
	got := match.Match(key, windows)
	if len(got) != 1 || got[0].ID != 2 {
		t.Fatalf("expected window 2, got %v", windowIDs(got))
	}
}

func TestMatch_InstanceFilter_Empty_MatchesAll(t *testing.T) {
	windows := []dbus.WindowInfo{
		makeWindow(1, "firefox", "Navigator"),
		makeWindow(2, "firefox", "Toolkit"),
	}
	key := match.WindowKey{WmClass: "firefox", WmClassInstance: ""}
	got := match.Match(key, windows)
	if len(got) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(got))
	}
}

func TestMatch_NoMatch_ReturnsEmptyNotNil(t *testing.T) {
	windows := []dbus.WindowInfo{
		makeWindow(1, "code", "code"),
	}
	key := match.WindowKey{WmClass: "firefox"}
	got := match.Match(key, windows)
	if got == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(got))
	}
}

func TestMatch_MultipleWindowsSameClass(t *testing.T) {
	const n = 5
	windows := make([]dbus.WindowInfo, n)
	for i := range windows {
		windows[i] = makeWindow(uint32(i+1), "terminal", "terminal")
	}
	key := match.WindowKey{WmClass: "terminal"}
	got := match.Match(key, windows)
	if len(got) != n {
		t.Fatalf("expected %d matches, got %d", n, len(got))
	}
}

func TestMatch_EmptyWindowList(t *testing.T) {
	key := match.WindowKey{WmClass: "firefox"}
	got := match.Match(key, []dbus.WindowInfo{})
	if got == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(got))
	}
}

// ByIndex tests ----------------------------------------------------------

func TestByIndex_ValidIndex(t *testing.T) {
	windows := []dbus.WindowInfo{
		makeWindow(10, "a", "a"),
		makeWindow(20, "b", "b"),
		makeWindow(30, "c", "c"),
	}
	got := match.ByIndex(windows, 1)
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if got.ID != 20 {
		t.Errorf("expected ID 20, got %d", got.ID)
	}
}

func TestByIndex_ZeroIndex(t *testing.T) {
	windows := []dbus.WindowInfo{makeWindow(7, "x", "x")}
	got := match.ByIndex(windows, 0)
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if got.ID != 7 {
		t.Errorf("expected ID 7, got %d", got.ID)
	}
}

func TestByIndex_OutOfBounds(t *testing.T) {
	windows := []dbus.WindowInfo{makeWindow(1, "a", "a")}
	tests := []struct {
		name  string
		index int
	}{
		{"beyond end", 1},
		{"large positive", 999},
		{"negative", -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := match.ByIndex(windows, tc.index); got != nil {
				t.Errorf("expected nil for index %d, got %+v", tc.index, got)
			}
		})
	}
}

func TestByIndex_EmptySlice(t *testing.T) {
	if got := match.ByIndex(nil, 0); got != nil {
		t.Errorf("expected nil for empty slice, got %+v", got)
	}
}

// MatchWithTitles tests --------------------------------------------------

func TestMatchWithTitles_TitleRegexMatch(t *testing.T) {
	windows := []dbus.WindowInfo{
		makeWindow(1, "firefox", "Navigator"),
		makeWindow(2, "firefox", "Navigator"),
	}
	titles := map[uint32]string{
		1: "GitHub - Mozilla Firefox",
		2: "Google - Mozilla Firefox",
	}
	key := match.WindowKey{WmClass: "firefox", TitlePattern: `(?i)github`}
	got := match.MatchWithTitles(key, windows, titles)
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("expected only window 1, got %v", windowIDs(got))
	}
}

func TestMatchWithTitles_TitleRegexNoMatch(t *testing.T) {
	windows := []dbus.WindowInfo{makeWindow(1, "firefox", "Navigator")}
	titles := map[uint32]string{1: "GitHub - Mozilla Firefox"}
	key := match.WindowKey{WmClass: "firefox", TitlePattern: `^GitLab`}
	got := match.MatchWithTitles(key, windows, titles)
	if got == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(got))
	}
}

func TestMatchWithTitles_EmptyPattern_MatchesAll(t *testing.T) {
	windows := []dbus.WindowInfo{
		makeWindow(1, "firefox", "Navigator"),
		makeWindow(2, "firefox", "Navigator"),
	}
	titles := map[uint32]string{1: "Page A", 2: "Page B"}
	key := match.WindowKey{WmClass: "firefox", TitlePattern: ""}
	got := match.MatchWithTitles(key, windows, titles)
	if len(got) != 2 {
		t.Fatalf("expected 2 matches with empty pattern, got %d", len(got))
	}
}

func TestMatchWithTitles_InvalidRegexp_FallsBackToNoTitleFilter(t *testing.T) {
	windows := []dbus.WindowInfo{
		makeWindow(1, "firefox", "Navigator"),
		makeWindow(2, "firefox", "Navigator"),
	}
	titles := map[uint32]string{1: "Page A", 2: "Page B"}
	// "[invalid" is not a valid regexp.
	key := match.WindowKey{WmClass: "firefox", TitlePattern: "[invalid"}
	got := match.MatchWithTitles(key, windows, titles)
	// Should fall back to Match behaviour: both firefox windows returned.
	if len(got) != 2 {
		t.Fatalf("expected 2 matches after invalid-regexp fallback, got %d", len(got))
	}
}

func TestMatchWithTitles_MissingTitleTreatedAsEmpty(t *testing.T) {
	windows := []dbus.WindowInfo{
		makeWindow(1, "firefox", "Navigator"), // title absent from map
		makeWindow(2, "firefox", "Navigator"),
	}
	titles := map[uint32]string{2: "Something"}
	// Pattern that matches empty string.
	key := match.WindowKey{WmClass: "firefox", TitlePattern: `^$`}
	got := match.MatchWithTitles(key, windows, titles)
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("expected window 1 (empty title), got %v", windowIDs(got))
	}
}
