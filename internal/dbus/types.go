// Package dbus wraps the window-calls GNOME Shell extension D-Bus interface.
//
// Destination:  org.gnome.Shell
// Object path:  /org/gnome/Shell/Extensions/Windows
// Interface:    org.gnome.Shell.Extensions.Windows
//
// The extension serialises all return values as JSON strings rather than using
// native D-Bus types, so every response must be unmarshalled before use.
package dbus

// WindowInfo describes a single managed window as returned by the window-calls
// extension.  JSON field names match the extension's output exactly.
type WindowInfo struct {
	WmClass         string `json:"wm_class"`
	WmClassInstance string `json:"wm_class_instance"`
	PID             int    `json:"pid"`
	ID              uint32 `json:"id"`
	FrameType       int    `json:"frame_type"`
	WindowType      int    `json:"window_type"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	X               int    `json:"x"`
	Y               int    `json:"y"`
	// Maximized encodes the axis: 0=none, 1=horizontal, 2=vertical, 3=both.
	Maximized          int  `json:"maximized"`
	Focus              bool `json:"focus"`
	InCurrentWorkspace bool `json:"in_current_workspace"`
	Moveable           bool `json:"moveable"`
	Resizeable         bool `json:"resizeable"`
	Workspace          int  `json:"workspace"`
	Monitor            int  `json:"monitor"`
}
