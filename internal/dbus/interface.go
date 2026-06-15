package dbus

// Windows is the complete set of operations exposed by the window-calls GNOME
// Shell extension.  Implementations include the real D-Bus Client and the
// MockClient used in tests.
type Windows interface {
	// List returns all windows known to the extension.
	List() ([]WindowInfo, error)

	// Details returns the full WindowInfo for a single window.
	Details(winID uint32) (WindowInfo, error)

	// GetTitle returns the display title of a window.
	GetTitle(winID uint32) (string, error)

	// Move repositions a window to (x, y).
	Move(winID uint32, x, y int) error

	// Resize changes a window's dimensions to (w, h).
	Resize(winID uint32, w, h int) error

	// MoveResize repositions and resizes a window in a single call.
	MoveResize(winID uint32, x, y, w, h int) error

	// MoveToWorkspace sends a window to the given workspace index.
	MoveToWorkspace(winID uint32, workspace int) error

	// Maximize maximises a window on both axes.
	Maximize(winID uint32) error

	// Unmaximize restores a maximised window.
	Unmaximize(winID uint32) error

	// Minimize minimises a window to the task bar.
	Minimize(winID uint32) error

	// Unminimize restores a minimised window.
	Unminimize(winID uint32) error

	// Activate raises and focuses a window.
	Activate(winID uint32) error

	// Close requests that the window be closed.
	Close(winID uint32) error

	// MakeFullscreen puts a window into fullscreen mode.
	MakeFullscreen(winID uint32) error
}
