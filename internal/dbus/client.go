package dbus

import (
	"encoding/json"
	"fmt"

	"github.com/godbus/dbus/v5"
)

const (
	dbusDestination = "org.gnome.Shell"
	dbusObjectPath  = "/org/gnome/Shell/Extensions/Windows"
	dbusInterface   = "org.gnome.Shell.Extensions.Windows"
)

// Client is a live D-Bus connection to the window-calls GNOME Shell extension.
// Obtain one via New(); release the underlying connection with Close().
type Client struct {
	conn *dbus.Conn
	obj  dbus.BusObject
}

// New connects to the session bus and verifies that the window-calls extension
// is reachable before returning.  Returns a descriptive error when the
// extension is absent or not yet activated.
func New() (*Client, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("dbus: connect to session bus: %w", err)
	}

	c := &Client{
		conn: conn,
		obj:  conn.Object(dbusDestination, dbusObjectPath),
	}

	if err := c.checkExtension(); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return c, nil
}

// Disconnect releases the underlying D-Bus connection.
// Named Disconnect rather than Close to avoid conflicting with the Windows
// interface method Close(winID uint32) error.
func (c *Client) Disconnect() error {
	return c.conn.Close()
}

// checkExtension issues a lightweight List call to confirm the extension is
// present.  A failure is wrapped in a user-friendly message.
func (c *Client) checkExtension() error {
	call := c.obj.Call(dbusInterface+".List", 0)
	if call.Err != nil {
		return fmt.Errorf(
			"dbus: window-calls extension not available (is it installed and enabled?): %w",
			call.Err,
		)
	}
	return nil
}

// voidCall invokes a D-Bus method that returns no useful value.
func (c *Client) voidCall(method string, args ...interface{}) error {
	call := c.obj.Call(dbusInterface+"."+method, 0, args...)
	return call.Err
}

// List returns all windows currently tracked by the extension.
func (c *Client) List() ([]WindowInfo, error) {
	var raw string
	if err := c.obj.Call(dbusInterface+".List", 0).Store(&raw); err != nil {
		return nil, fmt.Errorf("dbus List: %w", err)
	}

	var windows []WindowInfo
	if err := json.Unmarshal([]byte(raw), &windows); err != nil {
		return nil, fmt.Errorf("dbus List: unmarshal JSON: %w", err)
	}
	return windows, nil
}

// Details returns the full WindowInfo for the window identified by winID.
func (c *Client) Details(winID uint32) (WindowInfo, error) {
	var raw string
	if err := c.obj.Call(dbusInterface+".Details", 0, winID).Store(&raw); err != nil {
		return WindowInfo{}, fmt.Errorf("dbus Details: %w", err)
	}

	var win WindowInfo
	if err := json.Unmarshal([]byte(raw), &win); err != nil {
		return WindowInfo{}, fmt.Errorf("dbus Details: unmarshal JSON: %w", err)
	}
	return win, nil
}

// GetTitle returns the title string for the given window ID.
func (c *Client) GetTitle(winID uint32) (string, error) {
	var title string
	if err := c.obj.Call(dbusInterface+".GetTitle", 0, winID).Store(&title); err != nil {
		return "", fmt.Errorf("dbus GetTitle: %w", err)
	}
	return title, nil
}

// Move repositions window winID to (x, y).
func (c *Client) Move(winID uint32, x, y int) error {
	return c.voidCall("Move", winID, int32(x), int32(y))
}

// Resize changes window winID's size to (w, h).
// width and height are uint32 per the extension's D-Bus signature (uuu).
func (c *Client) Resize(winID uint32, w, h int) error {
	return c.voidCall("Resize", winID, uint32(w), uint32(h))
}

// MoveResize moves and resizes window winID in a single call.
// Extension signature is (uiiuu): winid=u, x=i, y=i, width=u, height=u.
func (c *Client) MoveResize(winID uint32, x, y, w, h int) error {
	return c.voidCall("MoveResize", winID, int32(x), int32(y), uint32(w), uint32(h))
}

// MoveToWorkspace sends window winID to the given workspace index.
// Extension signature is (uu): winid=u, workspaceNum=u.
func (c *Client) MoveToWorkspace(winID uint32, workspace int) error {
	return c.voidCall("MoveToWorkspace", winID, uint32(workspace))
}

// Maximize maximises window winID.
func (c *Client) Maximize(winID uint32) error {
	return c.voidCall("Maximize", winID)
}

// Unmaximize restores window winID from maximised state.
func (c *Client) Unmaximize(winID uint32) error {
	return c.voidCall("Unmaximize", winID)
}

// Minimize minimises window winID to the task bar.
func (c *Client) Minimize(winID uint32) error {
	return c.voidCall("Minimize", winID)
}

// Unminimize restores window winID from minimised state.
func (c *Client) Unminimize(winID uint32) error {
	return c.voidCall("Unminimize", winID)
}

// Activate raises and focuses window winID.
func (c *Client) Activate(winID uint32) error {
	return c.voidCall("Activate", winID)
}

// Close requests that window winID be closed.
func (c *Client) Close(winID uint32) error {
	return c.voidCall("Close", winID)
}

// MakeFullscreen puts window winID into fullscreen mode.
func (c *Client) MakeFullscreen(winID uint32) error {
	return c.voidCall("MakeFullscreen", winID)
}

// Compile-time proof that *Client satisfies the Windows interface.
var _ Windows = (*Client)(nil)
