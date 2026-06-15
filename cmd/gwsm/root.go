package main

import (
	"github.com/ambershark-mike/gwsm/internal/config"
	"github.com/ambershark-mike/gwsm/internal/state"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     config.Config
	store   *state.Store
)

var rootCmd = &cobra.Command{
	Use:   "gwsm",
	Short: "Gnome Window State Manager — save and restore window layouts",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return err
		}

		statePath := cfg.StateFile
		if statePath == "" {
			statePath = state.DefaultPath()
		}

		store, err = state.New(statePath)
		return err
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/gwsm/config.toml)")
}
