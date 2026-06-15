package daemon_test

import (
	"context"
	"io"
	"log"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ambershark-mike/gwsm/internal/daemon"
	"github.com/ambershark-mike/gwsm/internal/dbus"
	"github.com/ambershark-mike/gwsm/internal/state"
)

// ---------------------------------------------------------------------------
// safeMock: thread-safe wrapper around *dbus.MockClient
//
// MockClient is not goroutine-safe: the daemon's polling goroutine and the
// test goroutine both read/write MockClient.Windows and MockClient.Calls
// concurrently.  safeMock serialises all access with a single mutex while
// delegating every method to the underlying *dbus.MockClient.
// ---------------------------------------------------------------------------

type safeMock struct {
	mu sync.Mutex
	mc *dbus.MockClient
}

// addWindow appends w to the mock's window list in a thread-safe way.
func (s *safeMock) addWindow(w dbus.WindowInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mc.Windows = append(s.mc.Windows, w)
}

// calls returns a snapshot of recorded calls.
func (s *safeMock) calls() []dbus.MockCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]dbus.MockCall, len(s.mc.Calls))
	copy(out, s.mc.Calls)
	return out
}

// dbus.Windows interface — every method locks s.mu so that Calls appends and
// Windows slice reads are serialised with addWindow / calls.

func (s *safeMock) List() ([]dbus.WindowInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mc.List()
}

func (s *safeMock) Details(winID uint32) (dbus.WindowInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mc.Details(winID)
}

func (s *safeMock) GetTitle(winID uint32) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mc.GetTitle(winID)
}

func (s *safeMock) Move(winID uint32, x, y int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mc.Move(winID, x, y)
}

func (s *safeMock) Resize(winID uint32, w, h int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mc.Resize(winID, w, h)
}

func (s *safeMock) MoveResize(winID uint32, x, y, w, h int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mc.MoveResize(winID, x, y, w, h)
}

func (s *safeMock) MoveToWorkspace(winID uint32, workspace int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mc.MoveToWorkspace(winID, workspace)
}

func (s *safeMock) Maximize(winID uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mc.Maximize(winID)
}

func (s *safeMock) Unmaximize(winID uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mc.Unmaximize(winID)
}

func (s *safeMock) Minimize(winID uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mc.Minimize(winID)
}

func (s *safeMock) Unminimize(winID uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mc.Unminimize(winID)
}

func (s *safeMock) Activate(winID uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mc.Activate(winID)
}

func (s *safeMock) Close(winID uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mc.Close(winID)
}

func (s *safeMock) MakeFullscreen(winID uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mc.MakeFullscreen(winID)
}

// Compile-time proof that *safeMock satisfies the Windows interface.
var _ dbus.Windows = (*safeMock)(nil)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newStore(t *testing.T) *state.Store {
	t.Helper()
	st, err := state.New(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	return st
}

func saveProfile(t *testing.T, st *state.Store, p state.Profile) {
	t.Helper()
	if err := st.Save(p); err != nil {
		t.Fatalf("st.Save: %v", err)
	}
}

// silentLogger discards all log output so tests stay quiet.
func silentLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}

// methodNames extracts the method names from a call slice.
func methodNames(calls []dbus.MockCall) []string {
	names := make([]string, len(calls))
	for i, c := range calls {
		names[i] = c.Method
	}
	return names
}

// waitForCall polls mock.calls() until a call with the given method name appears
// or timeout elapses.  It returns true on success.
//
// Using active polling instead of a fixed sleep avoids the race condition where
// mock.calls() is snapshotted before the restore goroutine finishes all of its
// client calls (Unmaximize → MoveResize → MoveToWorkspace).
func waitForCall(mock *safeMock, method string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, c := range mock.calls() {
			if c.Method == method {
				return true
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

// waitForNthCall polls until at least n calls with the given method name appear
// or timeout elapses.  It returns true on success.
func waitForNthCall(mock *safeMock, method string, n int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		count := 0
		for _, c := range mock.calls() {
			if c.Method == method {
				count++
			}
		}
		if count >= n {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

// assertHasCall fails if no call with the given method name is found.
func assertHasCall(t *testing.T, calls []dbus.MockCall, method string) {
	t.Helper()
	for _, c := range calls {
		if c.Method == method {
			return
		}
	}
	t.Errorf("expected call %q not found in %v", method, methodNames(calls))
}

// assertNoCall fails if a call with the given method name is found.
func assertNoCall(t *testing.T, calls []dbus.MockCall, method string) {
	t.Helper()
	for _, c := range calls {
		if c.Method == method {
			t.Errorf("unexpected call %q found; all calls: %v", method, methodNames(calls))
			return
		}
	}
}

// assertCallOrder fails unless the first occurrence of first appears strictly
// before the first occurrence of second.
func assertCallOrder(t *testing.T, calls []dbus.MockCall, first, second string) {
	t.Helper()
	firstIdx, secondIdx := -1, -1
	for i, c := range calls {
		if c.Method == first && firstIdx == -1 {
			firstIdx = i
		}
		if c.Method == second && secondIdx == -1 {
			secondIdx = i
		}
	}
	if firstIdx == -1 {
		t.Errorf("call %q not found", first)
		return
	}
	if secondIdx == -1 {
		t.Errorf("call %q not found", second)
		return
	}
	if firstIdx >= secondIdx {
		t.Errorf("expected %q (index %d) before %q (index %d)", first, firstIdx, second, secondIdx)
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestDaemon_NewWindowRestored verifies that a window appearing after seeding
// is fully restored: Unmaximize → MoveResize → MoveToWorkspace.
func TestDaemon_NewWindowRestored(t *testing.T) {
	st := newStore(t)
	saveProfile(t, st, state.Profile{
		Name: "test",
		Windows: []state.SavedWindow{
			{WmClass: "Foo", WmClassInst: "foo", Index: 0, X: 100, Y: 200, Width: 800, Height: 600, Workspace: 1},
		},
	})

	mock := &safeMock{mc: &dbus.MockClient{}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d := daemon.New(mock, st, daemon.Config{
		ProfileName:  "test",
		PollInterval: 10 * time.Millisecond,
		RestoreDelay: 0,
		Logger:       silentLogger(),
	})
	go d.Run(ctx) //nolint:errcheck

	// Wait for the seed call to complete before adding the window, so it is
	// not captured into seenIDs.
	time.Sleep(25 * time.Millisecond)

	mock.addWindow(dbus.WindowInfo{
		ID: 1, WmClass: "Foo", WmClassInstance: "foo",
		FrameType: 0, WindowType: 0,
	})

	// Poll until MoveToWorkspace appears (the last call in the restore
	// sequence), then snapshot.  This avoids reading the calls before the
	// restore goroutine has finished.
	if !waitForCall(mock, "MoveToWorkspace", 300*time.Millisecond) {
		t.Fatalf("MoveToWorkspace not called within timeout; calls so far: %v",
			methodNames(mock.calls()))
	}

	calls := mock.calls()
	assertHasCall(t, calls, "Unmaximize")
	assertHasCall(t, calls, "MoveResize")
	assertHasCall(t, calls, "MoveToWorkspace")
}

// TestDaemon_ExistingWindowNotRestored verifies that windows open before Run
// is called are seeded into seenIDs and never restored.
func TestDaemon_ExistingWindowNotRestored(t *testing.T) {
	st := newStore(t)
	saveProfile(t, st, state.Profile{
		Name: "test",
		Windows: []state.SavedWindow{
			{WmClass: "Foo", WmClassInst: "foo", Index: 0, X: 100, Y: 200, Width: 800, Height: 600},
		},
	})

	mock := &safeMock{mc: &dbus.MockClient{
		Windows: []dbus.WindowInfo{
			{ID: 1, WmClass: "Foo", WmClassInstance: "foo", FrameType: 0, WindowType: 0},
		},
	}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d := daemon.New(mock, st, daemon.Config{
		ProfileName:  "test",
		PollInterval: 10 * time.Millisecond,
		RestoreDelay: 0,
		Logger:       silentLogger(),
	})
	go d.Run(ctx) //nolint:errcheck

	// Run several poll cycles.
	time.Sleep(80 * time.Millisecond)
	cancel()

	assertNoCall(t, mock.calls(), "MoveResize")
}

// TestDaemon_ProfileNotFound verifies that a missing profile is non-fatal:
// the daemon keeps running and returns only the context error on shutdown.
func TestDaemon_ProfileNotFound(t *testing.T) {
	st := newStore(t) // no profiles saved

	mock := &safeMock{mc: &dbus.MockClient{}}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	d := daemon.New(mock, st, daemon.Config{
		ProfileName:  "missing",
		PollInterval: 10 * time.Millisecond,
		RestoreDelay: 0,
		Logger:       silentLogger(),
	})

	err := d.Run(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Run: want context.DeadlineExceeded, got %v", err)
	}
}

// TestDaemon_MaximizedWindowRestored verifies that when a SavedWindow has
// Maximized > 0, the Maximize call is issued after MoveResize.
func TestDaemon_MaximizedWindowRestored(t *testing.T) {
	st := newStore(t)
	saveProfile(t, st, state.Profile{
		Name: "test",
		Windows: []state.SavedWindow{
			{WmClass: "Foo", WmClassInst: "foo", Index: 0, X: 0, Y: 0, Width: 1920, Height: 1080, Maximized: 3},
		},
	})

	mock := &safeMock{mc: &dbus.MockClient{}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d := daemon.New(mock, st, daemon.Config{
		ProfileName:  "test",
		PollInterval: 10 * time.Millisecond,
		RestoreDelay: 0,
		Logger:       silentLogger(),
	})
	go d.Run(ctx) //nolint:errcheck

	time.Sleep(25 * time.Millisecond)

	mock.addWindow(dbus.WindowInfo{
		ID: 1, WmClass: "Foo", WmClassInstance: "foo",
		FrameType: 0, WindowType: 0,
	})

	// Wait for Maximize, the last call in this restore sequence.
	if !waitForCall(mock, "Maximize", 300*time.Millisecond) {
		t.Fatalf("Maximize not called within timeout; calls so far: %v",
			methodNames(mock.calls()))
	}

	calls := mock.calls()
	assertHasCall(t, calls, "Maximize")
	assertCallOrder(t, calls, "MoveResize", "Maximize")
}

// TestDaemon_UnknownClassDoesNotPanic verifies that a window whose wm_class has
// no saved entries is silently skipped and does not cause a panic.
func TestDaemon_UnknownClassDoesNotPanic(t *testing.T) {
	st := newStore(t)
	// Profile only has ghostty — zen has no entry.
	saveProfile(t, st, state.Profile{
		Name: "test",
		Windows: []state.SavedWindow{
			{WmClass: "ghostty", WmClassInst: "ghostty", Index: 0, X: 0, Y: 0, Width: 800, Height: 600},
		},
	})

	mock := &safeMock{mc: &dbus.MockClient{}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d := daemon.New(mock, st, daemon.Config{
		ProfileName:  "test",
		PollInterval: 10 * time.Millisecond,
		RestoreDelay: 0,
		Logger:       silentLogger(),
	})
	go d.Run(ctx) //nolint:errcheck

	time.Sleep(25 * time.Millisecond)

	// Add a window whose class is NOT in the saved profile.
	mock.addWindow(dbus.WindowInfo{ID: 1, WmClass: "zen", WmClassInstance: "zen"})

	// Run several poll cycles — daemon must not panic.
	time.Sleep(60 * time.Millisecond)
	cancel()

	// zen should never have been restored.
	for _, c := range mock.calls() {
		if c.Method == "MoveResize" {
			t.Errorf("unexpected MoveResize for unknown class; calls: %v", methodNames(mock.calls()))
			break
		}
	}
}

// TestDaemon_ExtraWindowsReuseLastSlot verifies that when more windows of a
// class appear than there are saved entries, the extras are restored using the
// last saved entry rather than being ignored.
func TestDaemon_ExtraWindowsReuseLastSlot(t *testing.T) {
	st := newStore(t)
	saveProfile(t, st, state.Profile{
		Name: "test",
		Windows: []state.SavedWindow{
			// Only one ghostty entry saved.
			{WmClass: "ghostty", WmClassInst: "ghostty", Index: 0, X: 50, Y: 100, Width: 1200, Height: 800},
		},
	})

	mock := &safeMock{mc: &dbus.MockClient{}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d := daemon.New(mock, st, daemon.Config{
		ProfileName:  "test",
		PollInterval: 10 * time.Millisecond,
		RestoreDelay: 0,
		Logger:       silentLogger(),
	})
	go d.Run(ctx) //nolint:errcheck

	time.Sleep(25 * time.Millisecond)

	// First ghostty window.
	mock.addWindow(dbus.WindowInfo{ID: 1, WmClass: "ghostty", WmClassInstance: "ghostty"})
	time.Sleep(30 * time.Millisecond)

	// Second ghostty window — no saved slot at index 1, should reuse index 0.
	mock.addWindow(dbus.WindowInfo{ID: 2, WmClass: "ghostty", WmClassInstance: "ghostty"})

	// Expect two MoveResize calls.
	if !waitForNthCall(mock, "MoveResize", 2, 300*time.Millisecond) {
		t.Fatalf("expected 2 MoveResize calls; calls so far: %v", methodNames(mock.calls()))
	}

	// Both calls should use the same saved geometry (x=50).
	var mrCalls []dbus.MockCall
	for _, c := range mock.calls() {
		if c.Method == "MoveResize" {
			mrCalls = append(mrCalls, c)
		}
	}
	for i, c := range mrCalls {
		x, ok := c.Args[1].(int)
		if !ok {
			t.Fatalf("MoveResize call %d: x arg is not int (%T)", i, c.Args[1])
		}
		if x != 50 {
			t.Errorf("MoveResize call %d: want x=50, got x=%d", i, x)
		}
	}
}

// TestDaemon_MultipleWindowsSameClass verifies that the Nth window of a
// wm_class is matched against the Nth saved entry (ordered by Index), giving
// each window its own geometry.
func TestDaemon_MultipleWindowsSameClass(t *testing.T) {
	st := newStore(t)
	saveProfile(t, st, state.Profile{
		Name: "test",
		Windows: []state.SavedWindow{
			{WmClass: "Term", WmClassInst: "term", Index: 0, X: 0, Y: 0, Width: 800, Height: 600},
			{WmClass: "Term", WmClassInst: "term", Index: 1, X: 1000, Y: 0, Width: 800, Height: 600},
		},
	})

	mock := &safeMock{mc: &dbus.MockClient{}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d := daemon.New(mock, st, daemon.Config{
		ProfileName:  "test",
		PollInterval: 10 * time.Millisecond,
		RestoreDelay: 0,
		Logger:       silentLogger(),
	})
	go d.Run(ctx) //nolint:errcheck

	// Wait for seeding, then add the two Term windows in separate poll cycles.
	time.Sleep(25 * time.Millisecond)

	mock.addWindow(dbus.WindowInfo{ID: 1, WmClass: "Term", WmClassInstance: "term"})

	// Give the daemon time to discover window 1 before window 2 appears, so
	// that index assignment is deterministic.
	time.Sleep(30 * time.Millisecond)

	mock.addWindow(dbus.WindowInfo{ID: 2, WmClass: "Term", WmClassInstance: "term"})

	// Wait until two MoveResize calls have been recorded.
	if !waitForNthCall(mock, "MoveResize", 2, 300*time.Millisecond) {
		t.Fatalf("expected 2 MoveResize calls within timeout; calls so far: %v",
			methodNames(mock.calls()))
	}

	calls := mock.calls()

	// Collect MoveResize calls in order.
	var mrCalls []dbus.MockCall
	for _, c := range calls {
		if c.Method == "MoveResize" {
			mrCalls = append(mrCalls, c)
		}
	}

	if len(mrCalls) != 2 {
		t.Fatalf("expected 2 MoveResize calls, got %d; all calls: %v", len(mrCalls), methodNames(calls))
	}

	// Args layout for MoveResize: [winID uint32, x int, y int, w int, h int]
	x0, ok0 := mrCalls[0].Args[1].(int)
	x1, ok1 := mrCalls[1].Args[1].(int)
	if !ok0 || !ok1 {
		t.Fatalf("MoveResize x argument is not int: %T, %T", mrCalls[0].Args[1], mrCalls[1].Args[1])
	}

	if x0 != 0 {
		t.Errorf("first Term window MoveResize x: want 0, got %d", x0)
	}
	if x1 != 1000 {
		t.Errorf("second Term window MoveResize x: want 1000, got %d", x1)
	}
}
