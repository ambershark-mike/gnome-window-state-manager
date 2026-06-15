package main

import (
	"fmt"
	"time"

	"github.com/ambershark-mike/gwsm/internal/dbus"
	"github.com/ambershark-mike/gwsm/internal/match"
	"github.com/ambershark-mike/gwsm/internal/state"
	"github.com/spf13/cobra"
)

var restoreCmd = &cobra.Command{
	Use:   "restore [profile]",
	Short: "Restore a saved window layout profile",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName, _ := cmd.Flags().GetString("profile")
		if len(args) > 0 {
			profileName = args[0]
		}

		client, err := dbus.New()
		if err != nil {
			return err
		}
		defer client.Disconnect()

		profile, err := store.Load(profileName)
		if err != nil {
			return err
		}

		windows, err := client.List()
		if err != nil {
			return err
		}

		applied := 0
		for _, saved := range profile.Windows {
			key := match.WindowKey{
				WmClass:         saved.WmClass,
				WmClassInstance: saved.WmClassInst,
				TitlePattern:    saved.TitlePattern,
			}
			candidates := match.Match(key, windows)
			win := match.ByIndex(candidates, saved.Index)
			if win == nil {
				fmt.Printf("skip: no match for %s (index %d)\n", saved.WmClass, saved.Index)
				continue
			}
			if err := applyGeometry(client, win.ID, saved); err != nil {
				fmt.Printf("error: %s id=%d: %v\n", saved.WmClass, win.ID, err)
				continue
			}
			fmt.Printf("restored: %s instance=%s id=%d\n", saved.WmClass, saved.WmClassInst, win.ID)
			applied++
		}

		fmt.Printf("\nRestored %d/%d window(s) from profile %q\n", applied, len(profile.Windows), profileName)
		return nil
	},
}

// applyGeometry unmaximizes a window, repositions and resizes it, moves it to
// the correct workspace, then re-maximizes if the saved state requires it.
func applyGeometry(client dbus.Windows, winID uint32, saved state.SavedWindow) error {
	_ = client.Unmaximize(winID) // ignore error — window may not be maximized
	time.Sleep(100 * time.Millisecond)
	if err := client.MoveResize(winID, saved.X, saved.Y, saved.Width, saved.Height); err != nil {
		return err
	}
	if err := client.MoveToWorkspace(winID, saved.Workspace); err != nil {
		return err
	}
	if saved.Maximized > 0 {
		return client.Maximize(winID)
	}
	return nil
}

func init() {
	restoreCmd.Flags().StringP("profile", "p", "default", "profile name")
	rootCmd.AddCommand(restoreCmd)
}
