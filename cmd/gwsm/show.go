package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show [profile]",
	Short: "Show windows saved in a profile",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName, _ := cmd.Flags().GetString("profile")
		if len(args) > 0 {
			profileName = args[0]
		}

		profile, err := store.Load(profileName)
		if err != nil {
			return err
		}

		fmt.Printf("Profile: %s (%d window(s))\n\n", profile.Name, len(profile.Windows))

		const hdrFmt = "%-4s  %-30s  %-20s  %6s  %6s  %6s  %6s  %3s  %3s  %4s  %s\n"
		const rowFmt = "%-4d  %-30s  %-20s  %6d  %6d  %6d  %6d  %3d  %3d  %4d  %s\n"
		fmt.Printf(hdrFmt, "IDX", "WM_CLASS", "INSTANCE", "X", "Y", "W", "H", "WS", "MON", "MAX", "TITLE_PATTERN")
		fmt.Println(strings.Repeat("-", 121))

		for _, w := range profile.Windows {
			fmt.Printf(rowFmt,
				w.Index, w.WmClass, w.WmClassInst,
				w.X, w.Y, w.Width, w.Height,
				w.Workspace, w.Monitor, w.Maximized,
				w.TitlePattern,
			)
		}
		return nil
	},
}

func init() {
	showCmd.Flags().StringP("profile", "p", "default", "profile name")
	rootCmd.AddCommand(showCmd)
}
