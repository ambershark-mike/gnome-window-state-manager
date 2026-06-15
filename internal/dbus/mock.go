package dbus

import "fmt"

// MockCall records a single invocation on MockClient.
type MockCall struct {
	Method string
	Args   []interface{}
}

// MockClient is a test double for Windows.  Populate Windows before use and
// inject errors via the *Err fields.
type MockClient struct {
	// Windows is the prepopulated set of windows returned by List / Details.
	Windows []WindowInfo

	// Injected error fields — set to a non-nil error to simulate failures.
	ListErr     error
	DetailsErr  error
	GetTitleErr error

	// Calls records every method invocation in order.
	Calls []MockCall
}

func (m *MockClient) record(method string, args ...interface{}) {
	m.Calls = append(m.Calls, MockCall{Method: method, Args: args})
}

// List returns m.Windows or m.ListErr.
func (m *MockClient) List() ([]WindowInfo, error) {
	m.record("List")
	if m.ListErr != nil {
		return nil, m.ListErr
	}
	out := make([]WindowInfo, len(m.Windows))
	copy(out, m.Windows)
	return out, nil
}

// Details finds the window with the matching ID in m.Windows.
// Returns m.DetailsErr when set.
func (m *MockClient) Details(winID uint32) (WindowInfo, error) {
	m.record("Details", winID)
	if m.DetailsErr != nil {
		return WindowInfo{}, m.DetailsErr
	}
	for _, w := range m.Windows {
		if w.ID == winID {
			return w, nil
		}
	}
	return WindowInfo{}, fmt.Errorf("mock: window %d not found", winID)
}

// GetTitle returns the WmClass of the matching window as a stand-in title.
// Returns m.GetTitleErr when set.
func (m *MockClient) GetTitle(winID uint32) (string, error) {
	m.record("GetTitle", winID)
	if m.GetTitleErr != nil {
		return "", m.GetTitleErr
	}
	for _, w := range m.Windows {
		if w.ID == winID {
			return w.WmClass, nil
		}
	}
	return "", fmt.Errorf("mock: window %d not found", winID)
}

// Move records the call.
func (m *MockClient) Move(winID uint32, x, y int) error {
	m.record("Move", winID, x, y)
	return nil
}

// Resize records the call.
func (m *MockClient) Resize(winID uint32, w, h int) error {
	m.record("Resize", winID, w, h)
	return nil
}

// MoveResize records the call.
func (m *MockClient) MoveResize(winID uint32, x, y, w, h int) error {
	m.record("MoveResize", winID, x, y, w, h)
	return nil
}

// MoveToWorkspace records the call.
func (m *MockClient) MoveToWorkspace(winID uint32, workspace int) error {
	m.record("MoveToWorkspace", winID, workspace)
	return nil
}

// Maximize records the call.
func (m *MockClient) Maximize(winID uint32) error {
	m.record("Maximize", winID)
	return nil
}

// Unmaximize records the call.
func (m *MockClient) Unmaximize(winID uint32) error {
	m.record("Unmaximize", winID)
	return nil
}

// Minimize records the call.
func (m *MockClient) Minimize(winID uint32) error {
	m.record("Minimize", winID)
	return nil
}

// Unminimize records the call.
func (m *MockClient) Unminimize(winID uint32) error {
	m.record("Unminimize", winID)
	return nil
}

// Activate records the call.
func (m *MockClient) Activate(winID uint32) error {
	m.record("Activate", winID)
	return nil
}

// Close records the call.
func (m *MockClient) Close(winID uint32) error {
	m.record("Close", winID)
	return nil
}

// MakeFullscreen records the call.
func (m *MockClient) MakeFullscreen(winID uint32) error {
	m.record("MakeFullscreen", winID)
	return nil
}

// Compile-time proof that *MockClient satisfies the Windows interface.
var _ Windows = (*MockClient)(nil)
