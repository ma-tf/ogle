# ADR-0004: app/app.go is the root Bubble Tea model

**Status:** Accepted

The original intent was implemented as of this refactor. Package path is now `internal/app/app.go`.

## Context

Early iterations had the Bubble Tea entry point in `internal/tui.go` as a file-level function. As the application grew,
the root model needed to manage sub-models (startup flow and dashboard), dispatch messages between them, and own the
watcher lifecycle.

## Decision

The root Bubble Tea model lives in `internal/app/app.go` as a proper package. `cmd/root.go` calls `app.Start()` and
remains thin — it validates flags and delegates immediately.

## Consequences

- The `app` package has a clear, testable interface (`app.Start()`).
- `cmd/root.go` contains no Bubble Tea or TUI logic, only CLI flag parsing and pre-TUI validation (Explicit File mode
hard exits happen here).
- The root model owns the three top-level phases (`PhaseStartup`, `PhaseDashboard`, `PhaseWatching`) and dispatches `FileAvailabilityChanged`
to the correct active sub-model.
- `internal/tui.go` is removed.
