// Package state persists gwsm window layout profiles to a JSON file on disk.
//
// Profiles are keyed by name inside a single state file.  All mutations are
// written atomically: the new file is first written to a sibling ".tmp" path
// and then renamed over the final path, so a crash mid-write cannot corrupt
// existing data.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// SavedWindow captures the geometry and placement of one window within a
// saved profile.
type SavedWindow struct {
	WmClass      string `json:"wm_class"`
	WmClassInst  string `json:"wm_class_instance"`
	TitlePattern string `json:"title_pattern,omitempty"`
	Index        int    `json:"index"`
	X            int    `json:"x"`
	Y            int    `json:"y"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Workspace    int    `json:"workspace"`
	Monitor      int    `json:"monitor"`
	// Maximized encodes the axis: 0=none, 1=horizontal, 2=vertical, 3=both.
	Maximized int `json:"maximized"`
}

// Profile groups a set of saved windows under a human-readable name.
type Profile struct {
	Name    string        `json:"name"`
	Windows []SavedWindow `json:"windows"`
}

// stateFile is the on-disk JSON format.
type stateFile struct {
	Profiles map[string]Profile `json:"profiles"`
}

// Store manages reading and writing profiles to a single JSON file.
type Store struct {
	path string
}

// DefaultPath returns the conventional state-file location:
// ~/.local/share/gwsm/state.json
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to relative path when home directory is unavailable.
		return filepath.Join(".local", "share", "gwsm", "state.json")
	}
	return filepath.Join(home, ".local", "share", "gwsm", "state.json")
}

// New creates a Store backed by path.  The directory containing path is
// created if it does not already exist.
func New(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("state: create directory %q: %w", dir, err)
	}
	return &Store{path: path}, nil
}

// Save upserts profile into the store by profile.Name, then writes the full
// state file atomically.
func (s *Store) Save(profile Profile) error {
	sf, err := s.load()
	if err != nil {
		return err
	}
	sf.Profiles[profile.Name] = profile
	return s.write(sf)
}

// Load returns the profile with the given name, or an error when not found.
func (s *Store) Load(name string) (Profile, error) {
	sf, err := s.load()
	if err != nil {
		return Profile{}, err
	}
	p, ok := sf.Profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("state: profile %q not found", name)
	}
	return p, nil
}

// List returns the names of all stored profiles in sorted order.
func (s *Store) List() ([]string, error) {
	sf, err := s.load()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(sf.Profiles))
	for name := range sf.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// RemoveWindows removes window entries from a profile.
//
// wmClass and wmClassInst identify which entries to target. If index >= 0,
// only the entry at that class-relative index is removed. If index < 0, all
// entries for that class are removed.
//
// After removal the remaining entries for the affected class are re-indexed
// sequentially so the daemon's counting stays consistent.
func (s *Store) RemoveWindows(profileName, wmClass, wmClassInst string, index int) error {
	sf, err := s.load()
	if err != nil {
		return err
	}
	profile, ok := sf.Profiles[profileName]
	if !ok {
		return fmt.Errorf("state: profile %q not found", profileName)
	}

	var kept []SavedWindow
	removed := 0
	for _, sw := range profile.Windows {
		// Empty wmClassInst matches any instance of the class.
		classMatch := sw.WmClass == wmClass && (wmClassInst == "" || sw.WmClassInst == wmClassInst)
		if classMatch && (index < 0 || sw.Index == index) {
			removed++
			continue // drop this entry
		}
		kept = append(kept, sw)
	}

	if removed == 0 {
		if index < 0 {
			return fmt.Errorf("state: no entries found for class %q in profile %q", wmClass, profileName)
		}
		return fmt.Errorf("state: no entry found for class %q index %d in profile %q", wmClass, index, profileName)
	}

	// Re-index remaining entries for the affected class so indexes stay sequential.
	nextIdx := 0
	for i := range kept {
		if kept[i].WmClass == wmClass && kept[i].WmClassInst == wmClassInst {
			kept[i].Index = nextIdx
			nextIdx++
		}
	}

	profile.Windows = kept
	sf.Profiles[profileName] = profile
	return s.write(sf)
}

// Delete removes the named profile from the store.  It returns an error when
// the profile does not exist.
func (s *Store) Delete(name string) error {
	sf, err := s.load()
	if err != nil {
		return err
	}
	if _, ok := sf.Profiles[name]; !ok {
		return fmt.Errorf("state: profile %q not found", name)
	}
	delete(sf.Profiles, name)
	return s.write(sf)
}

// load reads and unmarshals the state file.  A missing file is treated as an
// empty state rather than an error.
func (s *Store) load() (stateFile, error) {
	sf := stateFile{Profiles: make(map[string]Profile)}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return sf, nil
		}
		return sf, fmt.Errorf("state: read %q: %w", s.path, err)
	}

	if err := json.Unmarshal(data, &sf); err != nil {
		return sf, fmt.Errorf("state: unmarshal %q: %w", s.path, err)
	}
	// Ensure the map is non-nil even when the JSON contained "profiles": null.
	if sf.Profiles == nil {
		sf.Profiles = make(map[string]Profile)
	}
	return sf, nil
}

// write marshals sf and atomically replaces the state file.
func (s *Store) write(sf stateFile) error {
	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return fmt.Errorf("state: marshal: %w", err)
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("state: write temp file %q: %w", tmp, err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		// Best-effort cleanup; ignore secondary error.
		_ = os.Remove(tmp)
		return fmt.Errorf("state: rename %q -> %q: %w", tmp, s.path, err)
	}
	return nil
}
