// Package daemon implements a polling daemon that watches for new windows and
// restores their saved geometry from a named profile in the state store.
package daemon

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/ambershark-mike/gwsm/internal/dbus"
	"github.com/ambershark-mike/gwsm/internal/state"
)

// Config holds the runtime configuration for a Daemon.
type Config struct {
	ProfileName  string
	PollInterval time.Duration
	RestoreDelay time.Duration
	Logger       *log.Logger // nil = use log.Default()
}

// Daemon polls for newly opened windows and restores their saved geometry.
type Daemon struct {
	client       dbus.Windows
	store        *state.Store
	profileName  string
	pollInterval time.Duration
	restoreDelay time.Duration
	logger       *log.Logger

	// seenIDs tracks window IDs that have already been processed so that
	// windows present at startup and already-handled new windows are never
	// restored twice.
	seenIDs map[uint32]bool

	// restoredCounts tracks how many windows of each wm_class+wm_class_instance
	// pair have been restored so far, allowing the Nth new window of a class to
	// be matched against the Nth saved entry for that class.
	restoredCounts map[string]int
}

// New creates a Daemon that uses client for window operations and store for
// loading profiles. If cfg.Logger is nil, log.Default() is used.
func New(client dbus.Windows, store *state.Store, cfg Config) *Daemon {
	l := cfg.Logger
	if l == nil {
		l = log.Default()
	}
	return &Daemon{
		client:         client,
		store:          store,
		profileName:    cfg.ProfileName,
		pollInterval:   cfg.PollInterval,
		restoreDelay:   cfg.RestoreDelay,
		logger:         l,
		seenIDs:        make(map[uint32]bool),
		restoredCounts: make(map[string]int),
	}
}

// Run seeds the daemon with the currently open windows (so they are not
// restored), then polls on each ticker interval until ctx is cancelled.
// It returns ctx.Err() when the context is done.
func (d *Daemon) Run(ctx context.Context) error {
	// Seed seenIDs with windows that are already open so they are not restored.
	initial, err := d.client.List()
	if err != nil {
		d.logger.Printf("daemon: seed: list windows: %v", err)
	} else {
		for _, w := range initial {
			d.seenIDs[w.ID] = true
		}
	}

	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := d.poll(ctx); err != nil {
				d.logger.Printf("daemon: poll: %v", err)
			}
		case <-ctx.Done():
			d.logger.Printf("daemon stopping")
			return ctx.Err()
		}
	}
}

// classKey returns the restoredCounts map key for a (wm_class, wm_class_instance) pair.
func classKey(wmClass, wmClassInst string) string {
	return wmClass + "\x00" + wmClassInst
}

// poll lists all current windows, skips ones already seen, and schedules a
// restore goroutine for every new application window that has a saved entry.
func (d *Daemon) poll(ctx context.Context) error {
	windows, err := d.client.List()
	if err != nil {
		return fmt.Errorf("list windows: %w", err)
	}

	profile, err := d.store.Load(d.profileName)
	if err != nil {
		d.logger.Printf("daemon: profile %q not available, skipping poll: %v", d.profileName, err)
		return nil
	}

	for _, w := range windows {
		if d.seenIDs[w.ID] {
			continue
		}
		d.seenIDs[w.ID] = true

		// Skip system windows; only normal application windows have both
		// FrameType == 0 and WindowType == 0.
		if w.FrameType != 0 || w.WindowType != 0 {
			continue
		}

		d.logger.Printf("daemon: new window id=%d class=%q instance=%q", w.ID, w.WmClass, w.WmClassInstance)

		// Collect all saved entries for this window's class, sorted by Index.
		var savedForClass []state.SavedWindow
		for _, sw := range profile.Windows {
			if sw.WmClass == w.WmClass && sw.WmClassInst == w.WmClassInstance {
				savedForClass = append(savedForClass, sw)
			}
		}
		sort.Slice(savedForClass, func(i, j int) bool {
			return savedForClass[i].Index < savedForClass[j].Index
		})

		// No saved entries for this class at all — nothing to restore.
		if len(savedForClass) == 0 {
			continue
		}

		key := classKey(w.WmClass, w.WmClassInstance)
		count := d.restoredCounts[key]
		d.restoredCounts[key]++

		// If more windows open than saved slots, reuse the last saved entry.
		// This means a single saved ghostty entry restores every ghostty window.
		if count >= len(savedForClass) {
			count = len(savedForClass) - 1
			d.logger.Printf("daemon: window id=%d class=%q: beyond saved slots, reusing last entry (index %d)",
				w.ID, w.WmClass, count)
		}

		saved := savedForClass[count]

		go d.restore(ctx, w.ID, saved)
	}
	return nil
}

// restore waits for the configured restore delay, then applies the saved
// geometry (position, size, workspace, and maximised state) to the window.
// It returns early without making any calls if ctx is cancelled during the
// delay.
func (d *Daemon) restore(ctx context.Context, winID uint32, saved state.SavedWindow) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(d.restoreDelay):
	}

	d.logger.Printf("daemon: restoring window %d (%s): x=%d y=%d w=%d h=%d workspace=%d",
		winID, saved.WmClass, saved.X, saved.Y, saved.Width, saved.Height, saved.Workspace)

	// Unmaximize first; a maximised window ignores MoveResize.
	if err := d.client.Unmaximize(winID); err != nil {
		d.logger.Printf("daemon: window %d: unmaximize: %v", winID, err)
	}

	if err := d.client.MoveResize(winID, saved.X, saved.Y, saved.Width, saved.Height); err != nil {
		d.logger.Printf("daemon: window %d: move-resize: %v", winID, err)
	}

	if err := d.client.MoveToWorkspace(winID, saved.Workspace); err != nil {
		d.logger.Printf("daemon: window %d: move-to-workspace: %v", winID, err)
	}

	if saved.Maximized > 0 {
		if err := d.client.Maximize(winID); err != nil {
			d.logger.Printf("daemon: window %d: maximize: %v", winID, err)
		}
	}

	d.logger.Printf("daemon: restored window %d (%s)", winID, saved.WmClass)
}
