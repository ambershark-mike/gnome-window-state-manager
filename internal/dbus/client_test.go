package dbus

import (
	"errors"
	"testing"
)

// fixture returns a MockClient pre-loaded with two windows.
func fixture() *MockClient {
	return &MockClient{
		Windows: []WindowInfo{
			{
				ID:                 1,
				WmClass:            "Alacritty",
				WmClassInstance:    "alacritty",
				PID:                1001,
				Width:              800,
				Height:             600,
				X:                  100,
				Y:                  200,
				Workspace:          0,
				Monitor:            0,
				Focus:              true,
				InCurrentWorkspace: true,
				Moveable:           true,
				Resizeable:         true,
			},
			{
				ID:                 2,
				WmClass:            "firefox",
				WmClassInstance:    "Navigator",
				PID:                2002,
				Width:              1280,
				Height:             900,
				X:                  0,
				Y:                  0,
				Workspace:          1,
				Monitor:            0,
				Focus:              false,
				InCurrentWorkspace: false,
				Moveable:           true,
				Resizeable:         true,
			},
		},
	}
}

// hasCall returns true when method appears at least once in Calls.
func hasCall(calls []MockCall, method string) bool {
	for _, c := range calls {
		if c.Method == method {
			return true
		}
	}
	return false
}

// lastCall returns the most recent MockCall for method, panics if absent.
func lastCall(t *testing.T, calls []MockCall, method string) MockCall {
	t.Helper()
	for i := len(calls) - 1; i >= 0; i-- {
		if calls[i].Method == method {
			return calls[i]
		}
	}
	t.Fatalf("no call recorded for method %q", method)
	return MockCall{}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestList_ReturnsAllWindows(t *testing.T) {
	m := fixture()
	got, err := m.List()
	if err != nil {
		t.Fatalf("List() unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("List() len = %d, want 2", len(got))
	}
	if got[0].ID != 1 || got[1].ID != 2 {
		t.Errorf("List() IDs = [%d, %d], want [1, 2]", got[0].ID, got[1].ID)
	}
}

func TestList_RecordsCall(t *testing.T) {
	m := fixture()
	_, _ = m.List()
	if !hasCall(m.Calls, "List") {
		t.Error("List() call not recorded")
	}
}

func TestList_ReturnsCopy(t *testing.T) {
	m := fixture()
	got, _ := m.List()
	// Mutating the returned slice must not affect internal state.
	got[0].WmClass = "mutated"
	orig, _ := m.List()
	if orig[0].WmClass == "mutated" {
		t.Error("List() returned a reference to internal slice; expected a copy")
	}
}

func TestList_ErrorPropagation(t *testing.T) {
	sentinel := errors.New("list error")
	m := &MockClient{ListErr: sentinel}
	_, err := m.List()
	if !errors.Is(err, sentinel) {
		t.Errorf("List() error = %v, want %v", err, sentinel)
	}
	if !hasCall(m.Calls, "List") {
		t.Error("List() call not recorded on error path")
	}
}

// ---------------------------------------------------------------------------
// Details
// ---------------------------------------------------------------------------

func TestDetails_FindsWindowByID(t *testing.T) {
	tests := []struct {
		name    string
		id      uint32
		wantWmc string
	}{
		{"first window", 1, "Alacritty"},
		{"second window", 2, "firefox"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := fixture()
			got, err := m.Details(tt.id)
			if err != nil {
				t.Fatalf("Details(%d) unexpected error: %v", tt.id, err)
			}
			if got.WmClass != tt.wantWmc {
				t.Errorf("Details(%d).WmClass = %q, want %q", tt.id, got.WmClass, tt.wantWmc)
			}
		})
	}
}

func TestDetails_RecordsCallWithID(t *testing.T) {
	m := fixture()
	_, _ = m.Details(1)
	c := lastCall(t, m.Calls, "Details")
	if c.Args[0] != uint32(1) {
		t.Errorf("Details args[0] = %v, want uint32(1)", c.Args[0])
	}
}

func TestDetails_UnknownIDReturnsError(t *testing.T) {
	m := fixture()
	_, err := m.Details(999)
	if err == nil {
		t.Fatal("Details(999) expected error, got nil")
	}
}

func TestDetails_ErrorPropagation(t *testing.T) {
	sentinel := errors.New("details error")
	m := &MockClient{DetailsErr: sentinel}
	_, err := m.Details(1)
	if !errors.Is(err, sentinel) {
		t.Errorf("Details() error = %v, want %v", err, sentinel)
	}
}

// ---------------------------------------------------------------------------
// GetTitle
// ---------------------------------------------------------------------------

func TestGetTitle_ReturnsWmClass(t *testing.T) {
	m := fixture()
	title, err := m.GetTitle(1)
	if err != nil {
		t.Fatalf("GetTitle(1) unexpected error: %v", err)
	}
	if title != "Alacritty" {
		t.Errorf("GetTitle(1) = %q, want %q", title, "Alacritty")
	}
}

func TestGetTitle_RecordsCall(t *testing.T) {
	m := fixture()
	_, _ = m.GetTitle(2)
	c := lastCall(t, m.Calls, "GetTitle")
	if c.Args[0] != uint32(2) {
		t.Errorf("GetTitle args[0] = %v, want uint32(2)", c.Args[0])
	}
}

func TestGetTitle_UnknownIDReturnsError(t *testing.T) {
	m := fixture()
	_, err := m.GetTitle(404)
	if err == nil {
		t.Fatal("GetTitle(404) expected error, got nil")
	}
}

func TestGetTitle_ErrorPropagation(t *testing.T) {
	sentinel := errors.New("title error")
	m := &MockClient{GetTitleErr: sentinel}
	_, err := m.GetTitle(1)
	if !errors.Is(err, sentinel) {
		t.Errorf("GetTitle() error = %v, want %v", err, sentinel)
	}
}

// ---------------------------------------------------------------------------
// Mutating methods — verify each is recorded with correct arguments
// ---------------------------------------------------------------------------

func TestMove_RecordsCall(t *testing.T) {
	m := fixture()
	if err := m.Move(1, 50, 75); err != nil {
		t.Fatalf("Move() error: %v", err)
	}
	c := lastCall(t, m.Calls, "Move")
	want := []interface{}{uint32(1), 50, 75}
	assertArgs(t, c.Args, want)
}

func TestResize_RecordsCall(t *testing.T) {
	m := fixture()
	if err := m.Resize(1, 1024, 768); err != nil {
		t.Fatalf("Resize() error: %v", err)
	}
	c := lastCall(t, m.Calls, "Resize")
	assertArgs(t, c.Args, []interface{}{uint32(1), 1024, 768})
}

func TestMoveResize_RecordsCall(t *testing.T) {
	m := fixture()
	if err := m.MoveResize(2, 10, 20, 640, 480); err != nil {
		t.Fatalf("MoveResize() error: %v", err)
	}
	c := lastCall(t, m.Calls, "MoveResize")
	assertArgs(t, c.Args, []interface{}{uint32(2), 10, 20, 640, 480})
}

func TestMoveToWorkspace_RecordsCall(t *testing.T) {
	m := fixture()
	if err := m.MoveToWorkspace(1, 3); err != nil {
		t.Fatalf("MoveToWorkspace() error: %v", err)
	}
	c := lastCall(t, m.Calls, "MoveToWorkspace")
	assertArgs(t, c.Args, []interface{}{uint32(1), 3})
}

func TestMaximize_RecordsCall(t *testing.T) {
	m := fixture()
	if err := m.Maximize(1); err != nil {
		t.Fatalf("Maximize() error: %v", err)
	}
	c := lastCall(t, m.Calls, "Maximize")
	assertArgs(t, c.Args, []interface{}{uint32(1)})
}

func TestUnmaximize_RecordsCall(t *testing.T) {
	m := fixture()
	if err := m.Unmaximize(1); err != nil {
		t.Fatalf("Unmaximize() error: %v", err)
	}
	c := lastCall(t, m.Calls, "Unmaximize")
	assertArgs(t, c.Args, []interface{}{uint32(1)})
}

func TestMinimize_RecordsCall(t *testing.T) {
	m := fixture()
	if err := m.Minimize(2); err != nil {
		t.Fatalf("Minimize() error: %v", err)
	}
	c := lastCall(t, m.Calls, "Minimize")
	assertArgs(t, c.Args, []interface{}{uint32(2)})
}

func TestUnminimize_RecordsCall(t *testing.T) {
	m := fixture()
	if err := m.Unminimize(2); err != nil {
		t.Fatalf("Unminimize() error: %v", err)
	}
	c := lastCall(t, m.Calls, "Unminimize")
	assertArgs(t, c.Args, []interface{}{uint32(2)})
}

func TestActivate_RecordsCall(t *testing.T) {
	m := fixture()
	if err := m.Activate(1); err != nil {
		t.Fatalf("Activate() error: %v", err)
	}
	c := lastCall(t, m.Calls, "Activate")
	assertArgs(t, c.Args, []interface{}{uint32(1)})
}

func TestClose_RecordsCall(t *testing.T) {
	m := fixture()
	if err := m.Close(2); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
	c := lastCall(t, m.Calls, "Close")
	assertArgs(t, c.Args, []interface{}{uint32(2)})
}

func TestMakeFullscreen_RecordsCall(t *testing.T) {
	m := fixture()
	if err := m.MakeFullscreen(1); err != nil {
		t.Fatalf("MakeFullscreen() error: %v", err)
	}
	c := lastCall(t, m.Calls, "MakeFullscreen")
	assertArgs(t, c.Args, []interface{}{uint32(1)})
}

// ---------------------------------------------------------------------------
// Table-driven: all single-arg void methods
// ---------------------------------------------------------------------------

func TestSingleArgMethods_RecordedCorrectly(t *testing.T) {
	type action func(w Windows, id uint32) error

	tests := []struct {
		name   string
		action action
	}{
		{"Maximize", func(w Windows, id uint32) error { return w.Maximize(id) }},
		{"Unmaximize", func(w Windows, id uint32) error { return w.Unmaximize(id) }},
		{"Minimize", func(w Windows, id uint32) error { return w.Minimize(id) }},
		{"Unminimize", func(w Windows, id uint32) error { return w.Unminimize(id) }},
		{"Activate", func(w Windows, id uint32) error { return w.Activate(id) }},
		{"Close", func(w Windows, id uint32) error { return w.Close(id) }},
		{"MakeFullscreen", func(w Windows, id uint32) error { return w.MakeFullscreen(id) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := fixture()
			const testID = uint32(42)
			if err := tt.action(m, testID); err != nil {
				t.Fatalf("%s() unexpected error: %v", tt.name, err)
			}
			if !hasCall(m.Calls, tt.name) {
				t.Errorf("%s() call not recorded", tt.name)
			}
			c := lastCall(t, m.Calls, tt.name)
			if len(c.Args) != 1 || c.Args[0] != testID {
				t.Errorf("%s() args = %v, want [%v]", tt.name, c.Args, testID)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// MockClient satisfies Windows interface (compile-time; also verified below)
// ---------------------------------------------------------------------------

func TestMockClientImplementsWindows(t *testing.T) {
	var _ Windows = (*MockClient)(nil)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// assertArgs checks that got matches want element-by-element.
func assertArgs(t *testing.T, got, want []interface{}) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("args length = %d, want %d; got %v want %v", len(got), len(want), got, want)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("args[%d] = %v (%T), want %v (%T)", i, got[i], got[i], want[i], want[i])
		}
	}
}
