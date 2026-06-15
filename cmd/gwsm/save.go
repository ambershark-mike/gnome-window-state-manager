package main

import (
	"fmt"
	"strconv"

	"github.com/ambershark-mike/gwsm/internal/dbus"
	"github.com/ambershark-mike/gwsm/internal/state"
	"github.com/spf13/cobra"
)

var saveCmd = &cobra.Command{
	Use:   "save [profile]",
	Short: "Save the current window layout to a profile",
	Long: `Save the current window layout to a named profile.

Without --id, all visible normal windows are saved and the profile is replaced.

With --id, only the specified window(s) are saved and they are merged into the
existing profile by default. Use --replace to update an entry that already
exists for that class instead of appending a new one.

Examples:
  gwsm save                             # snapshot all windows → "default"
  gwsm save work                        # snapshot all windows → "work"
  gwsm save work --class ghostty        # snapshot only Ghostty windows
  gwsm save work --id 4081547293        # add one window to "work"
  gwsm save work --id 123 --id 456      # add two windows to "work"
  gwsm save work --id 4081547293 --replace  # update existing ghostty entry`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName, _ := cmd.Flags().GetString("profile")
		if len(args) > 0 {
			profileName = args[0]
		}
		filterClass, _ := cmd.Flags().GetString("class")
		idStrs, _ := cmd.Flags().GetStringSlice("id")
		replace, _ := cmd.Flags().GetBool("replace")

		// Parse --id values into a set for O(1) lookup.
		filterIDs := make(map[uint32]bool, len(idStrs))
		for _, s := range idStrs {
			v, err := strconv.ParseUint(s, 10, 32)
			if err != nil {
				return fmt.Errorf("invalid window ID %q: %w", s, err)
			}
			filterIDs[uint32(v)] = true
		}

		client, err := dbus.New()
		if err != nil {
			return err
		}
		defer client.Disconnect()

		windows, err := client.List()
		if err != nil {
			return err
		}

		// Filter: normal windows only, then optionally by --class and/or --id.
		var filtered []dbus.WindowInfo
		for _, w := range windows {
			if w.FrameType != 0 || w.WindowType != 0 {
				continue
			}
			if filterClass != "" && w.WmClass != filterClass {
				continue
			}
			if len(filterIDs) > 0 && !filterIDs[w.ID] {
				continue
			}
			filtered = append(filtered, w)
		}

		if len(filtered) == 0 {
			return fmt.Errorf("no matching windows found")
		}

		// Warn about any requested IDs that weren't found.
		if len(filterIDs) > 0 {
			found := make(map[uint32]bool, len(filtered))
			for _, w := range filtered {
				found[w.ID] = true
			}
			for id := range filterIDs {
				if !found[id] {
					fmt.Printf("warning: window ID %d not found (it may have closed)\n", id)
				}
			}
		}

		// Decide whether to merge into the existing profile or replace it.
		// When --id is used the default is merge; without --id it's replace.
		mergeMode := len(filterIDs) > 0

		// Load existing profile for merge/replace operations.
		var existing []state.SavedWindow
		classCounts := make(map[string]int)
		if mergeMode {
			if p, err := store.Load(profileName); err == nil {
				existing = p.Windows
				if !replace {
					// Append mode: seed counts so new entries get the next index.
					for _, sw := range existing {
						key := sw.WmClass + "\x00" + sw.WmClassInst
						if sw.Index+1 > classCounts[key] {
							classCounts[key] = sw.Index + 1
						}
					}
				}
				// replace mode: classCounts stays at 0 so new entries start at
				// index 0 and overwrite matching existing entries by position.
			}
			// profile-not-found is fine — we just start fresh.
		}

		// Fetch accurate geometry via Details() — List() returns broken
		// x/y/width/height on GNOME 44+. Workspace comes from List().
		newWindows := make([]state.SavedWindow, 0, len(filtered))
		for _, base := range filtered {
			details, err := client.Details(base.ID)
			if err != nil {
				fmt.Printf("warning: could not get details for window %d (%s): %v\n", base.ID, base.WmClass, err)
				details = base
			}
			details.Workspace = base.Workspace

			key := details.WmClass + "\x00" + details.WmClassInstance
			idx := classCounts[key]
			classCounts[key]++
			newWindows = append(newWindows, state.SavedWindow{
				WmClass:     details.WmClass,
				WmClassInst: details.WmClassInstance,
				Index:       idx,
				X:           details.X,
				Y:           details.Y,
				Width:       details.Width,
				Height:      details.Height,
				Workspace:   details.Workspace,
				Monitor:     details.Monitor,
				Maximized:   details.Maximized,
			})
		}

		var finalWindows []state.SavedWindow
		if !mergeMode {
			// Full snapshot — replace the entire profile.
			finalWindows = newWindows
		} else if replace {
			// Replace mode — overwrite existing entries by class+index, append the rest.
			type entryKey struct {
				wmClass, wmClassInst string
				index                int
			}
			posMap := make(map[entryKey]int, len(existing))
			for i, sw := range existing {
				posMap[entryKey{sw.WmClass, sw.WmClassInst, sw.Index}] = i
			}
			merged := make([]state.SavedWindow, len(existing))
			copy(merged, existing)
			for _, nw := range newWindows {
				ek := entryKey{nw.WmClass, nw.WmClassInst, nw.Index}
				if pos, found := posMap[ek]; found {
					merged[pos] = nw // update in-place
				} else {
					merged = append(merged, nw) // new entry
				}
			}
			finalWindows = merged
		} else {
			// Default merge — append new windows to existing list.
			finalWindows = append(existing, newWindows...)
		}

		profile := state.Profile{Name: profileName, Windows: finalWindows}
		if err := store.Save(profile); err != nil {
			return err
		}

		switch {
		case !mergeMode:
			fmt.Printf("Saved %d window(s) to profile %q\n", len(finalWindows), profileName)
		case replace:
			fmt.Printf("Replaced %d window entry(s) in profile %q (%d total)\n",
				len(newWindows), profileName, len(finalWindows))
		default:
			fmt.Printf("Added %d window(s) to profile %q (%d total)\n",
				len(newWindows), profileName, len(finalWindows))
		}
		return nil
	},
}

func init() {
	saveCmd.Flags().StringP("profile", "p", "default", "profile name")
	saveCmd.Flags().StringP("class", "c", "", "filter by WM_CLASS (empty = save all normal windows)")
	saveCmd.Flags().StringSlice("id", nil, "save only the window(s) with these IDs (from 'gwsm windows'); repeatable")
	saveCmd.Flags().Bool("replace", false, "update existing profile entries for the same class instead of appending (only meaningful with --id)")
	rootCmd.AddCommand(saveCmd)
}
