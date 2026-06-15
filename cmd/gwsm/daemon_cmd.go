package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ambershark-mike/gwsm/internal/daemon"
	"github.com/ambershark-mike/gwsm/internal/dbus"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run gwsm as a daemon that auto-restores window layouts on appearance",
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName, _ := cmd.Flags().GetString("profile")

		client, err := dbus.New()
		if err != nil {
			return err
		}
		defer client.Disconnect()

		if profileName == "" {
			profileName = cfg.DefaultProfile
		}

		var logger *log.Logger
		if cfg.LogFile != "" {
			f, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
			if err != nil {
				return fmt.Errorf("open log file %q: %w", cfg.LogFile, err)
			}
			defer f.Close()
			logger = log.New(f, "", log.LstdFlags)
		} else {
			logger = log.New(os.Stdout, "", log.LstdFlags)
		}

		d := daemon.New(client, store, daemon.Config{
			ProfileName:  profileName,
			PollInterval: time.Duration(cfg.PollIntervalMs) * time.Millisecond,
			RestoreDelay: time.Duration(cfg.RestoreDelayMs) * time.Millisecond,
			Logger:       logger,
		})

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		if err := d.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		return nil
	},
}

func init() {
	daemonCmd.Flags().StringP("profile", "p", "", "profile name (default: from config default_profile)")
	rootCmd.AddCommand(daemonCmd)
}
