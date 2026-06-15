package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		names, err := store.List()
		if err != nil {
			return err
		}
		if len(names) == 0 {
			fmt.Println("No profiles saved.")
			return nil
		}
		for _, name := range names {
			fmt.Println(name)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
