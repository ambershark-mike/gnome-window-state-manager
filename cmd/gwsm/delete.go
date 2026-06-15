package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [profile]",
	Short: "Delete a profile or remove specific window entries from a profile",
	Long: `Delete an entire profile, or remove individual window entries from a profile.

Without --class, the entire named profile is deleted.
With --class, only window entries matching that WM_CLASS are removed.
With --class and --index, only the entry at that specific index is removed.

The index corresponds to the IDX column shown by 'gwsm show'.
After removal, remaining entries for the affected class are re-indexed
sequentially so the daemon's restore counting stays consistent.

Examples:
  gwsm delete work                            # delete entire "work" profile
  gwsm delete work --class ghostty            # remove all ghostty entries
  gwsm delete work --class ghostty --index 1  # remove only ghostty index 1`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName, _ := cmd.Flags().GetString("profile")
		if len(args) > 0 {
			profileName = args[0]
		}
		if profileName == "" {
			return fmt.Errorf("profile name required: use `gwsm delete <name>` or `gwsm delete -p <name>`")
		}

		wmClass, _ := cmd.Flags().GetString("class")
		index, _ := cmd.Flags().GetInt("index")

		if wmClass == "" {
			// No --class: delete the entire profile.
			if err := store.Delete(profileName); err != nil {
				return err
			}
			fmt.Printf("Deleted profile %q\n", profileName)
			return nil
		}

		// --class provided: remove window entry(s) from the profile.
		// Pass empty string for wmClassInst so all instances of the class match.
		if err := store.RemoveWindows(profileName, wmClass, "", index); err != nil {
			return err
		}

		if index < 0 {
			fmt.Printf("Removed all %q entries from profile %q\n", wmClass, profileName)
		} else {
			fmt.Printf("Removed %q index %d from profile %q\n", wmClass, index, profileName)
		}
		return nil
	},
}

func init() {
	deleteCmd.Flags().StringP("profile", "p", "", "profile name")
	deleteCmd.Flags().StringP("class", "c", "", "WM_CLASS of the window entry to remove (from 'gwsm show')")
	deleteCmd.Flags().Int("index", -1, "class-relative index of the entry to remove (-1 = all entries for the class)")
	rootCmd.AddCommand(deleteCmd)
}
