package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ambershark-mike/gwsm/internal/config"
)

func TestDefaults(t *testing.T) {
	d := config.Defaults()

	if d.PollIntervalMs != 1000 {
		t.Errorf("PollIntervalMs: got %d, want 1000", d.PollIntervalMs)
	}
	if d.RestoreDelayMs != 500 {
		t.Errorf("RestoreDelayMs: got %d, want 500", d.RestoreDelayMs)
	}
	if d.DefaultProfile != "default" {
		t.Errorf("DefaultProfile: got %q, want \"default\"", d.DefaultProfile)
	}
	if d.StateFile != "" {
		t.Errorf("StateFile: got %q, want empty string", d.StateFile)
	}
	if d.LogFile != "" {
		t.Errorf("LogFile: got %q, want empty string", d.LogFile)
	}
}

func TestDefaultPath(t *testing.T) {
	p := config.DefaultPath()
	want := filepath.Join("gwsm", "config.toml")
	if !strings.HasSuffix(p, want) {
		t.Errorf("DefaultPath() = %q, want suffix %q", p, want)
	}
}

func TestLoad_FileNotExist(t *testing.T) {
	cfg, err := config.Load("/nonexistent/path/that/cannot/exist/config.toml")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if cfg != config.Defaults() {
		t.Errorf("expected Defaults() when file is missing, got %+v", cfg)
	}
}

func TestLoad_ValidToml(t *testing.T) {
	content := `
poll_interval_ms = 250
state_file       = "/tmp/gwsm.state"
`
	f := writeTempFile(t, content)

	cfg, err := config.Load(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Overridden values.
	if cfg.PollIntervalMs != 250 {
		t.Errorf("PollIntervalMs: got %d, want 250", cfg.PollIntervalMs)
	}
	if cfg.StateFile != "/tmp/gwsm.state" {
		t.Errorf("StateFile: got %q, want \"/tmp/gwsm.state\"", cfg.StateFile)
	}

	// Non-overridden fields must retain their defaults.
	def := config.Defaults()
	if cfg.RestoreDelayMs != def.RestoreDelayMs {
		t.Errorf("RestoreDelayMs: got %d, want %d", cfg.RestoreDelayMs, def.RestoreDelayMs)
	}
	if cfg.DefaultProfile != def.DefaultProfile {
		t.Errorf("DefaultProfile: got %q, want %q", cfg.DefaultProfile, def.DefaultProfile)
	}
	if cfg.LogFile != "" {
		t.Errorf("LogFile: got %q, want empty string", cfg.LogFile)
	}
}

func TestLoad_InvalidToml(t *testing.T) {
	content := `this is not [ valid toml ===`
	f := writeTempFile(t, content)

	_, err := config.Load(f)
	if err == nil {
		t.Fatal("expected an error for invalid TOML, got nil")
	}
}

// writeTempFile creates a temporary file containing content and returns its path.
// It registers a cleanup function to remove the file at test teardown.
func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "gwsm-config-*.toml")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("closing temp file: %v", err)
	}
	return f.Name()
}
