# GWSM — Gnome Window State Manager

Go CLI + background daemon that saves and restores window geometry for GNOME Shell under Wayland.

## Build & Test

- **Build:** `make build` or `go build -o gwsm ./cmd/gwsm`
- **Test:** `make test` or `go test -v -race -coverprofile=coverage.out ./...`
- **Lint:** `make lint` or `go vet ./...`
- **Install:** `make install` (copies to `~/.local/bin/gwsm`)

## Project Structure

```
cmd/gwsm/           # CLI entrypoint (cobra commands)
internal/dbus/      # window-calls D-Bus client + mock
internal/state/     # profile persistence (JSON, atomic writes)
internal/match/     # window identity & regex matching
internal/daemon/    # background polling daemon
internal/config/    # TOML config loading
```

## Key Conventions

- External test packages (`package foo_test`) — test the public API
- Table-driven tests with `t.Run(name, fn)` subtests
- Errors wrapped with `fmt.Errorf("context: %w", err)` using package prefix
- D-Bus operations abstracted behind `dbus.Windows` interface for testability
- No vendoring; use Go modules
- Window identity is `wm_class` + `wm_class_instance` (never `id` — it's volatile)
- Daemon polls `List()` on interval (no D-Bus signals available)
- Maximized windows must be Unmaximized before MoveResize
- `Client.Disconnect()` (not `Close()` — avoids collision with `Windows.Close`)

## Architecture

```
CLI (cobra) -> internal/config -> internal/state -> internal/dbus -> internal/match -> internal/daemon
```

Daemon uses dependency injection: `dbus.Windows` interface + `*state.Store` injected into `daemon.New()`.

## Project Rules

- **Test Coverage** ≥ 80% — all code should have unit test coverage where possible
- **README** must be kept up to date with documentation on how the project works
- **Do not assume** — always ask for clarification on any task, do not make assumptions
- **Be mindful of security** — check for vulnerabilities; be careful with low-usage libraries, prompt for confirmation when unsure
