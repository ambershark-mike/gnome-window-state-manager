package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds all runtime configuration for gwsm.
type Config struct {
	PollIntervalMs int    `toml:"poll_interval_ms"`
	RestoreDelayMs int    `toml:"restore_delay_ms"`
	StateFile      string `toml:"state_file"`
	DefaultProfile string `toml:"default_profile"`
	LogFile        string `toml:"log_file"`
}

// Defaults returns a Config populated with sensible out-of-the-box values.
func Defaults() Config {
	return Config{
		PollIntervalMs: 1000,
		RestoreDelayMs: 500,
		DefaultProfile: "default",
	}
}

// DefaultPath returns the canonical config file path: ~/.config/gwsm/config.toml.
// It falls back to $HOME/.config if os.UserConfigDir() fails.
func DefaultPath() string {
	base, err := os.UserConfigDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "gwsm", "config.toml")
}

// Load reads the TOML config file at path and merges it on top of Defaults().
// If path is empty, DefaultPath() is used.
// A missing config file is not an error — Defaults() is returned unchanged.
func Load(path string) (Config, error) {
	if path == "" {
		path = DefaultPath()
	}

	cfg := Defaults()

	_, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Defaults(), nil
		}
		return Config{}, err
	}

	return cfg, nil
}
