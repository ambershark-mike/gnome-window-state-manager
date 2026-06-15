package main

import (
	"fmt"
	"strings"

	"github.com/ambershark-mike/gwsm/internal/dbus"
	"github.com/spf13/cobra"
)

var windowsCmd = &cobra.Command{
	Use:   "windows",
	Short: "List all open normal windows",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := dbus.New()
		if err != nil {
			return err
		}
		defer client.Disconnect()

		windows, err := client.List()
		if err != nil {
			return err
		}

		const hdrFmt = "%-10s  %-30s  %-20s  %6s  %6s  %6s  %6s  %3s  %3s  %5s\n"
		const rowFmt = "%-10d  %-30s  %-20s  %6d  %6d  %6d  %6d  %3d  %3d  %5s\n"
		fmt.Printf(hdrFmt, "ID", "WM_CLASS", "INSTANCE", "X", "Y", "W", "H", "WS", "MON", "FOCUS")
		fmt.Println(strings.Repeat("-", 113))

		// List() has broken x/y/width/height on GNOME 44+ (meta_window.get_x() etc.
		// don't exist); call Details() per window for accurate frame geometry.
		count := 0
		for _, base := range windows {
			if base.FrameType != 0 || base.WindowType != 0 {
				continue
			}
			w, err := client.Details(base.ID)
			if err != nil {
				w = base
			}
			w.Workspace = base.Workspace // Details() lacks workspace index
			focus := ""
			if w.Focus {
				focus = "yes"
			}
			fmt.Printf(rowFmt,
				base.ID, w.WmClass, w.WmClassInstance,
				w.X, w.Y, w.Width, w.Height,
				w.Workspace, w.Monitor, focus,
			)
			count++
		}

		fmt.Printf("\n%d window(s)\n", count)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(windowsCmd)
}
