// Package match provides window-list filtering helpers for gwsm.
//
// It operates on slices of [dbus.WindowInfo] returned by the window-calls
// extension and never performs additional D-Bus calls itself.
package match

import (
	"regexp"

	"github.com/ambershark-mike/gwsm/internal/dbus"
)

// WindowKey describes the criteria used to select windows from a list.
//
// WmClass is always required.  WmClassInstance, when non-empty, further
// constrains the match to windows whose wm_class_instance equals the given
// value.  TitlePattern is an optional regular expression applied against the
// window title; it is only evaluated by [MatchWithTitles].
type WindowKey struct {
	WmClass         string
	WmClassInstance string // empty = match any instance
	TitlePattern    string // optional regexp; empty = match any title
}

// Match returns every window in windows whose WmClass equals key.WmClass.
// When key.WmClassInstance is non-empty the result is further restricted to
// windows whose WmClassInstance also matches.
//
// The returned slice is always non-nil; callers can rely on len == 0 to signal
// "no match" without a nil check.
func Match(key WindowKey, windows []dbus.WindowInfo) []dbus.WindowInfo {
	result := make([]dbus.WindowInfo, 0)
	for _, w := range windows {
		if w.WmClass != key.WmClass {
			continue
		}
		if key.WmClassInstance != "" && w.WmClassInstance != key.WmClassInstance {
			continue
		}
		result = append(result, w)
	}
	return result
}

// MatchWithTitles behaves like [Match] but additionally filters by
// key.TitlePattern when it is non-empty.  The pattern is compiled as a
// regular expression and matched against the title found in titles (keyed by
// window ID).  Windows whose ID is absent from titles are treated as having an
// empty title string.
//
// If key.TitlePattern is not a valid regular expression the title filter is
// silently skipped and the function falls back to the same behaviour as
// [Match].
func MatchWithTitles(key WindowKey, windows []dbus.WindowInfo, titles map[uint32]string) []dbus.WindowInfo {
	candidates := Match(key, windows)

	if key.TitlePattern == "" {
		return candidates
	}

	re, err := regexp.Compile(key.TitlePattern)
	if err != nil {
		// Invalid regexp: skip title filtering.
		return candidates
	}

	result := make([]dbus.WindowInfo, 0)
	for _, w := range candidates {
		title := titles[w.ID]
		if re.MatchString(title) {
			result = append(result, w)
		}
	}
	return result
}

// ByIndex returns a pointer to the window at position index within windows, or
// nil when index is out of bounds.
func ByIndex(windows []dbus.WindowInfo, index int) *dbus.WindowInfo {
	if index < 0 || index >= len(windows) {
		return nil
	}
	return &windows[index]
}
