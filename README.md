# GWSM — Gnome Window State Manager (for Wayland)

GNOME does not remember window positions, sizes, or workspaces across application launches. GWSM fills that gap.

It saves a snapshot of your window layout into a named **profile** and then — either on demand or automatically via a background daemon — restores each window to its saved geometry when it appears.

**NOTE**: This only works for Gnome on Wayland.  It is specifically designed to fill the hole that devilspie filled on X11.

---

## How It Works

GWSM communicates with GNOME Shell through the [window-calls](https://github.com/ickyicky/window-calls) extension via D-Bus. The extension exposes methods to list, move, resize, and manage windows from outside GNOME Shell — something Wayland's security model otherwise prevents.

```
┌─────────────┐  gdbus / D-Bus  ┌──────────────────────┐
│    gwsm     │ ──────────────► │  window-calls ext.   │
│ (daemon/CLI)│ ◄────────────── │  (GNOME Shell)       │
└─────────────┘  JSON results   └──────────────────────┘
```

Window identity is matched by `wm_class` + `wm_class_instance` (stable across launches), never by the volatile numeric window ID.

---

## Prerequisites

| Requirement | Notes |
|---|---|
| GNOME Shell 44+ | Under Wayland |
| [window-calls extension](https://extensions.gnome.org/extension/4724/window-calls/) | Must be installed and **enabled** |
| Go 1.21+ | Only needed to build from source |

---

## Installation

### Build from source

```bash
git clone https://github.com/ambershark-mike/gwsm
cd gwsm
make install        # installs to ~/.local/bin/gwsm
```

### Run as a systemd user service (recommended)

```bash
make enable-service
```

This installs `~/.config/systemd/user/gwsm.service` and starts the daemon on every GNOME login.

### Packages (requires Docker)

All package builds are fully self-contained — the binary is statically compiled
(`CGO_ENABLED=0`) so there are no shared library dependencies. The only runtime
requirements are GNOME Shell and the `window-calls` extension.

#### .deb (Debian / Ubuntu)

```bash
make deb                  # → dist/gwsm_1.0.0_amd64.deb
make deb VERSION=1.2.0   # custom version

sudo apt install ./dist/gwsm_1.0.0_amd64.deb
systemctl --user enable --now gwsm
```

Installs to `/usr/bin/gwsm` and `/usr/lib/systemd/user/gwsm.service`.

#### .tar.gz (any Linux distro)

```bash
make tar                  # → dist/gwsm_1.0.0_linux_amd64.tar.gz
make tar VERSION=1.2.0   # custom version

tar -xzf dist/gwsm_1.0.0_linux_amd64.tar.gz
cd gwsm-1.0.0-linux-amd64/
sudo ./install.sh         # installs to /usr/local/bin
```

Or install to your home directory without sudo:
```bash
INSTALL_DIR=~/.local/bin ./install.sh
```

The `install.sh` script patches the correct binary path into the service file
automatically and places it in `/usr/lib/systemd/user/` (root) or
`~/.config/systemd/user/` (non-root).

#### Build both at once

```bash
make dist   # builds .deb and .tar.gz in one go
```

---

## Usage

### Save your current layout

```bash
# Save all open windows to the "default" profile
gwsm save

# Save to a named profile
gwsm save work

# Save only windows of a specific app
gwsm save work --class ghostty
```

### Save specific windows by ID

Use `gwsm windows` to find the ID of the window you want, then pass it with `--id`.
When `--id` is used, windows are **merged into the existing profile by default** —
no other entries are disturbed.

```bash
gwsm windows
# ID            WM_CLASS                       INSTANCE               ...
# 4081547293    com.mitchellh.ghostty          com.mitchellh.ghostty  ...
# 4081547146    zen                            zen                    ...

# Add one window to the profile (merges, does not replace)
gwsm save work --id 4081547293

# Add several windows at once
gwsm save work --id 4081547293 --id 4081547146

# Update (replace) an existing entry for that class with current geometry
gwsm save work --id 4081547293 --replace
```

### Restore a layout

```bash
# Restore the default profile to currently open windows
gwsm restore

# Restore a named profile
gwsm restore --profile work
```

### Manage profiles and windows

```bash
# List all saved profiles
gwsm list

# Show windows saved in a profile
gwsm show work

# Delete an entire profile
gwsm delete work

# Remove all saved entries for a specific app from a profile
gwsm delete work --class com.mitchellh.ghostty

# Remove only a specific instance (use IDX from 'gwsm show')
gwsm delete work --class com.mitchellh.ghostty --index 1
```

### Inspect live windows

```bash
# List all currently open windows (useful for scripting / debugging)
gwsm windows
```

### Run the daemon manually

```bash
# Auto-restore using the default profile
gwsm daemon

# Auto-restore using a named profile
gwsm daemon --profile work
```

---

## Configuration

Optional config file at `~/.config/gwsm/config.toml`:

```toml
# How often the daemon checks for new windows (milliseconds)
poll_interval_ms = 1000

# How long to wait after a window appears before restoring geometry
# (gives the app time to finish drawing its initial layout)
restore_delay_ms = 500

# Profile the daemon restores on startup
default_profile = "default"

# Override the state file location (default: ~/.local/share/gwsm/state.json)
state_file = ""

# Write daemon logs to a file instead of stdout/journal
log_file = ""
```

All settings are optional. Missing config file = use defaults.

---

## How Window Matching Works

Windows are identified by `wm_class` + `wm_class_instance`. These values are stable properties set by the application itself and do not change across launches.

- **Single-window apps** (e.g. `Caprine`): one saved entry restores every window of that class — the daemon always applies the saved geometry regardless of how many instances are open.
- **Multi-window apps with distinct layouts** (e.g. two terminals at different positions): save the layout while both are open. The first window of that class to appear gets index 0's geometry, the second gets index 1's, and so on.
- **More windows than saved entries**: once all saved slots are used, any further windows of that class reuse the **last saved entry**. So a single saved Ghostty entry will position every Ghostty window you open.

### Overflow behaviour example

| Saved entries | Windows opened | Geometry applied |
|---|---|---|
| 1 (index 0) | 1st, 2nd, 3rd | all get index 0 |
| 2 (index 0, 1) | 1st, 2nd, 3rd | 1st → 0, 2nd → 1, 3rd → 1 (last) |

To find the `wm_class` and `wm_class_instance` values of your windows:

```bash
gwsm windows
```

---

## State File

Window layouts are stored at `~/.local/share/gwsm/state.json`. Writes are atomic (temp-file + rename) to prevent corruption. The file is human-readable JSON and can be hand-edited or version-controlled.

---

## Key Constraint: Maximized Windows

GNOME Shell ignores `MoveResize` calls on a maximized window. GWSM always calls `Unmaximize` before applying geometry, then re-maximizes if the saved state was maximized.

---

## Development

### Running Tests

**All packages (recommended):**
```bash
make test
```
Runs every test with the race detector enabled and prints a per-package coverage summary.

**All packages via `go test` directly:**
```bash
go test ./...
```

**A single package, verbose:**
```bash
go test -v ./internal/daemon/
go test -v ./internal/state/
go test -v ./internal/match/
go test -v ./internal/dbus/
go test -v ./internal/config/
```

**Coverage breakdown:**

| Package | Coverage | Notes |
|---|---|---|
| `internal/match` | 100% | |
| `internal/config` | 80% | |
| `internal/daemon` | 80% | |
| `internal/state` | 71% | |
| `internal/dbus` | 49% | `client.go` requires a live session bus; `mock.go` is fully covered |
| `cmd/gwsm` | 0% | Cobra wiring only; all logic lives in the tested internal packages |

### Other Make Targets

```bash
make build   # compile binary
make lint    # go vet + golangci-lint (if installed)
make clean   # remove build artifacts
```

### Project Layout

```
cmd/gwsm/          CLI entrypoint (cobra commands)
internal/dbus/     D-Bus client wrapper for window-calls
internal/match/    Window identity & matching logic
internal/state/    Profile persistence (JSON)
internal/config/   Config file handling (TOML)
internal/daemon/   Background polling daemon
```

---

## License

Copyright (C) 2026 Mike Ryan

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
[GNU General Public License](LICENSE) for more details.
