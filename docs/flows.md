# Flows

Documents the state machines, screen states, and transition logic for the TUI.

---

## Overview

### With `-f <path>` (Explicit File)

```text
-f given
├── path is a directory          → hard exit: "path is a directory, expected a file"
├── file does not exist          → hard exit: "file not found: <path>"
├── file fails Parse()           → hard exit: "invalid compose file: <error>"
└── valid                        → dashboard
```

Hard exits happen in `cmd/root.go` before the TUI is initialised.

### Without `-f` (File Discovery)

```text
no -f
└── ScanAll(CWD) + Parse() each candidate
    ├── 0 valid files            → Watching screen (cold start)
    ├── 1 valid file             → dashboard
    └── 2+ valid files           → Project Selector → dashboard
```

Validity requires both conditions: file exists on disk **and** parses as valid compose YAML.

### Runtime: file disappears (Disconnected)

```text
dashboard → watched file deleted or moved
└── Watching screen ("disconnected — waiting for <filename>")
    └── watches for the SAME filename to reappear
        └── file reappears + valid   → dashboard
```

### Watching screen: file appears (cold start)

```text
fsnotify event (create/write in CWD)
└── re-run ScanAll() + Parse()
    ├── 0 valid  → stay on Watching screen
    ├── 1 valid  → dashboard
    └── 2+ valid → Project Selector → dashboard
```

---

## Watcher Lifetime

The watcher is created at app startup and runs for the entire process lifetime — including while the dashboard is
active. `app.Init()` starts `watcher.Next()` and the app re-subscribes after every `FileAvailabilityChanged` by
returning another `watcher.Next()` Cmd from `Update`. The active sub-model (startup flow or dashboard) receives the
message via `app.Update`'s dispatch logic.

---

## Watcher Edge Case Behaviour

The watcher (`internal/services/watcher`) handles several edge cases in its event loop:

| Behaviour | Implementation |
|---|---|
| `Chmod` events filtered | `Chmod` events are ignored and do not trigger a scan |
| Unknown filenames filtered | Events for files not in the known filename set are ignored |
| Errors on errors channel | Logged at `slog.Warn` level; event processing continues unaffected |
| `Close()` idempotency | Multiple `Close()` calls return `nil` after the first; `Next()` returns `nil` once closed |
| Channel closure on events channel | The goroutine exits and the underlying fsnotify watcher is closed |
| Extra file events with absent file | When an extra (non-known) file is monitored and the event fires but the file does not exist on disk, `Files` in the emitted message is empty |

See `internal/services/watcher/service_test.go` for test coverage of each case.

---

## Root Orchestrator (`internal/app/app.go`)

The app manages three phases plus a cross-phase About overlay:

```text
PhaseStartup    — startup flow is the active sub-model
PhaseDashboard  — dashboard flow is active (post-ProjectLoaded)
PhaseWatching   — watching flow is active (disconnected, waiting for file to reappear)
```

The About overlay is a cross-phase UI layer that can be opened from any phase via F1 or
brand-click; it is rendered on top of the current view using the compositor. While the overlay
is open, phase-specific input is blocked. Close keys: F1, esc, q (q does not quit when about
is open).

### Init (two Cmds in parallel)

```text
app.Init()
├── watcher.Next()                → begins perpetual watcher subscription
└── startup.Init() (or direct)    → kicks off scan (or immediate parse for -f case)
```

If `-f` was given (already validated in `cmd/root.go`), the initial scan is skipped and the startup flow receives the
path directly.

### Message dispatch

```text
app.Update(msg)
├── msgs.ProjectLoaded           → transition startup → dashboard, emit msgs.FrameHeight, msgs.TopbarContext
├── msgs.FileAvailabilityChanged → re-subscribe watcher, dispatch to startup or dashboard
├── msgs.FileRemoved             → transition to PhaseWatching, emit msgs.TopbarContext, msgs.FrameHeight, msgs.BindingsMsg(watching keymap)
├── msgs.DisplayError            → forward to statusbar (auto-clear after 3s), emit msgs.FrameHeight
├── msgs.DisplayStatus           → forward to statusbar (auto-clear after 3s), emit msgs.FrameHeight
├── msgs.ClearStatusMsg          → forward to statusbar, emit msgs.FrameHeight
├── tea.WindowSizeMsg            → forward to active sub-model
├── theme.Changed                → update pointer, forward to active sub-model
├── msgs.SettingsApplied         → load theme, update config, save config, emit theme.Changed (not forwarded)
├── msgs.BindingsMsg             → store keymap for computeFrameHeight
├── tea.KeyPressMsg              → handleKeyPress (help toggle → emit msgs.FrameHeight, about overlay F1/esc/q, quit, `ctrl+p` → 30s CPU profile dump)
├── tea.MouseClickMsg            → handleMouseClick (brand zone → about overlay)
├── msgs.AboutVisibilityChanged  → track showingAbout flag
└── other msgs                   → forward to active sub-model (emits msgs.FrameHeight when chrome height changes)
```

---

## Startup Flow (`internal/ui/flows/startup`)

A simple model (82 lines, no State pattern). Key behaviour:

- On `msgs.FileSelected`: parse the selected file, emit `msgs.ProjectLoaded`
- On `tea.WindowSizeMsg`: forward to fileSelect sub-model
- All other messages: forward to fileSelect sub-model

The startup flow does not own scan/parse logic — those happen via `scanner.ScanAll()` and `parser.Parse()` in the
watching/fileselect components before a `FileSelected` msg reaches this flow.

---

## Watching View (`internal/ui/components/watching`)

Rendered by the app's `PhaseWatching` phase. Also used when the dashboard transitions to the Disconnected state (file
disappeared at runtime).

```text
stateIdle        — compose file unavailable, waiting for it to reappear
stateParseError  — file exists but failed to parse, showing error inline
```

### Cold start vs. Disconnected

The watching view accepts a mode that controls the message shown:

| Mode           | Heading                                   |
|----------------|-------------------------------------------|
| `cold`         | Watching for a compose file…              |
| `disconnected` | Disconnected — waiting for `<filename>`   |

In disconnected mode, `FileAvailabilityChanged` is only acted on if the specific filename that was being monitored is
present in `Files`.

---

## Fileselect View (`internal/ui/components/fileselect`)

Rendered by the startup flow in the `Selecting` state (Project Selector).

```text
fileselectBrowsing  — list of valid files, cursor navigating
fileselectError     — Parse failed for the confirmed selection
                      (file was valid at list time, broken by the time Parse ran)
                      inline notice beneath the list; list remains active
```

On a new `FileAvailabilityChanged` the list is refreshed. If the previously errored file is no longer present, the error
notice is cleared.

---

## Dashboard (`internal/ui/flows/dashboard`)

Entered after `app` receives `ProjectLoaded{Project}`.

The dashboard is a flat model (no sub-states). It:

- Dispatches `StatePollTick` to the service panel and emits a `docker.Ps()` Cmd
- Routes `ServiceStop/Start/Restart/Rebuild/ActionCompleted` to `handleServiceAction`; when `Err` is set on
  `ServiceActionCompleted`, the stderr content is wrapped into an error and `*exec.ExitError` is preserved with exit code
- Handles `FileAvailabilityChanged` — if the project file is still present, re-parses and updates; if absent, sends a
 msg that triggers `app` to transition to `PhaseWatching`
- Forwards all messages to its sub-models (accordion, labelsaccordion, carousel, panel, settings)
- On `ServiceSelected` or `ServicesPolled` when runtime data is available (non-nil), triggers `docker.Inspect` to fetch
 container labels for `labelsaccordion`
- Handles `c` key — emits `ClearLogBuffer{ServiceName}` to clear the selected service's log buffer
- Toggles settings overlay via `SettingsVisibilityChanged`

---

## Message Types (`internal/msgs`)

| Message                          | Emitted by                      | Consumed by                                 |
|----------------------------------|---------------------------------|---------------------------------------------|
| `FileAvailabilityChanged{Files}` | `watcher`                       | `app` (dispatches to startup/dashboard)     |
| `FileRemoved{File}`              | `dashboard`                     | `app` (triggers PhaseWatching)              |
| `FileSelected{Path}`             | fileselect                      | startup                                     |
| `ProjectLoaded{Project}`         | startup / watching              | `app` (triggers PhaseDashboard)               |
| `DaemonConnected{}`              | `svcdocker.Connect`             | `topbar`, `servicepanel`, `servicehost`     |
| `DaemonUnavailable{Err}`         | `svcdocker.Connect`             | `topbar` (starts retry countdown)           |
| `DaemonTick{}`                   | `topbar.daemonTickCmd()` (1s `tea.Tick`)  | `topbar`                                    |
| `DaemonGraceExpired{}`           | `topbar.Init()` (10s grace one-shot)     | `topbar`                                    |
| `DaemonPoll{}`                   | `topbar.pollDaemonCmd()` (2s `tea.Tick`) | `topbar` (triggers `docker.Connect`)        |
| `TopbarContext{Phase,File,Service}` | `app` (phase transition)        | `topbar`                                    |
| `StatePollTick`                  | `servicepanel` (timer)          | `dashboard` (triggers `docker.Ps`)          |
| `ServicesPolled{Runtimes,Err}`   | `docker.Ps`                     | `dashboard`, `carousel`, `accordion`        |
| `ServiceStop`                    | `carousel/card` (user action)   | `dashboard` → `handleServiceAction`         |
| `ServiceStart`                   | `carousel/card` (user action)   | `dashboard` → `handleServiceAction`         |
| `ServiceRestart`                 | `carousel/card` (user action)   | `dashboard` → `handleServiceAction`         |
| `ServiceRebuild`                 | `carousel/card` (user action)   | `dashboard` → `handleServiceAction`         |
| `ServiceActionCompleted{ServiceName,Action,Err}` | `svcdocker`                     | `dashboard` (error: stderr wrapped into error, `ExitError` preserved with exit code), `carousel/card` |
| `LogLinesAvailable{ServiceName}`  | `LogStreamer`                   | `logpane` (via `servicehost`)               |
| `LogStreamError{Err,ServiceName}`| `LogStreamer`                   | `servicehost` (closes streamer, schedules retry with exponential backoff 2s→30s cap via `LogStreamRetryTick{}`) |
| `LogStreamContainerNotFound{ServiceName}` | `LogStreamer`                   | `servicehost` (closes streamer, resets retry count — no retry scheduled) |
| `LogStreamRetryTick{}`           | `servicehost` (timer)           | `servicehost` (restarts streamer after error) |
| `ServiceSelected{ServiceName}`   | `carousel` (hover/focus)        | `dashboard`, `accordion`, `servicehost`     |
| `SettingsApplied{Theme,LBCap}`   | `settings`                      | `app` (loads theme, saves config, emits `theme.Changed`)                |
| `SettingsVisibilityChanged`      | `settings`                      | `dashboard`                                 |
| `AboutVisibilityChanged{Visible}`| `app`                           | `app` (tracks showingAbout flag)            |
| `ContainerLabelsPolled{Labels,Err}` | `docker.Inspect`                | `dashboard`, `labelsaccordion`              |
| `ToggleLogWrap`                  | `dashboard` (keybinding)        | `logpane`                                   |
| `LogWrapStatus{On,Overflow,ServiceName}` | `logpane` (via `servicehost`) | `topbar` (filters by active service)        |
| `ReportWrapStatus{ServiceName}` | `dashboard` (on `ServiceSelected`) | `servicehost` → `logpane` (triggers current `LogWrapStatus` emission for badge sync) |
| `ClearLogBuffer{ServiceName}`    | `dashboard` (keybinding `c`)    | `logpane` (clears lines, drains chan, resets viewport), `servicehost` (routes by name) |
| `BindingsMsg{Keymap}`            | various flows                   | `app` (stores keymap for `computeFrameHeight`), `helpbar`           |
| `DisplayError{Err}`              | any component                   | `statusbar` (auto-clear after 3s)           |
| `DisplayStatus{Msg}`             | any component                   | `statusbar` (auto-clear after 3s)           |
| `ClearStatusMsg{}`               | `statusbar` (timer)             | `statusbar`                                 |
| `theme.Changed`                  | external (theme switcher)       | all components with theme pointer           |
| `FrameHeight{Height}`            | `app` (on status bar activation, help toggle, project load, file removal) | `logpane`, `dashboard` |

---

## Topbar Daemon Lifecycle (`internal/ui/components/topbar`)

The topbar manages a daemon connectivity state machine with three states and a grace period / retry loop:

```text
Connecting (initial)
├── DaemonConnected         → Connected (clear retry, start health polling)
└── DaemonGraceExpired      → Unavailable (set retry deadline = now + 10s, start 1s tick)
    └── DaemonTick (each 1s)
        ├── IsRetryDue      → Connecting (clear retry, fire docker.Connect)
        └── not due         → daemonTickCmd (continue 1s tick)

Connected
├── DaemonUnavailable       → Unavailable (set retry deadline = now + 10s, start 1s tick)
└── DaemonPoll (each 2s)    → docker.Connect (health check)

Unavailable
├── DaemonTick (each 1s)
│   ├── IsRetryDue          → Connecting (fire docker.Connect)
│   └── not due             → daemonTickCmd (continue 1s tick)
├── DaemonConnected         → Connected (clear retry, start health polling)
└── DaemonUnavailable       → update retry deadline (idempotent)
```

Key behaviours:

- Grace period: 10 seconds from `Init()`; if no `DaemonConnected` arrives in that window, transitions to Unavailable
- Retry: every 1 second after entering Unavailable; retry interval is 10 seconds (defined as `connection.RetryInterval`
 const)
- Health polling: every 2 seconds when Connected; fires `DaemonPoll` which triggers `docker.Connect()` as a health check
- The topbar renders the daemon status (Connecting/Connected/Unavailable) in the top-right of the application frame
- The retry countdown is rendered by the topbar, not the Service Inspector
- The topbar renders wrap/truncation badges between the context text and the daemon status: `WRAP` (amber background)
 when soft wrap is on, `>>` (amber background) when the log viewport has overflow content. Only one badge is shown at a
  time; wrap takes precedence over overflow.
- The topbar filters `LogWrapStatus` by `ServiceName`: only messages where `ServiceName` matches `selectedService` are
 applied. On service switch (`TopbarContext` with a different `Service`), the wrap and overflow badges are reset to off.

---

## About Overlay

The About overlay shows version information, ASCII art, and a GitHub URL. It is a
cross-phase UI layer — accessible from any phase (startup, dashboard, or watching).

| Trigger | Action |
|---------|--------|
| `F1`    | Open / Close the About overlay |
| Brand click (mouse) | Open the About overlay |
| `esc` or `q` | Close the About overlay (q does not quit when about is open) |

The overlay is rendered using the compositor on top of the current view. While the
overlay is open, phase-specific key handling is blocked. The overlay is implemented
in `internal/ui/components/about/`.

Custom theme overrides in `UserThemeFile` support 6 about-specific YAML fields:

| YAML field | Maps to | Description |
|------------|---------|-------------|
| `aboutBackgroundColour` | `Theme.AboutBackground` | About overlay background fill |
| `aboutTitleColour` | `Theme.AboutTitleColour` | Title ("ogle") foreground |
| `aboutArtColour` | `Theme.AboutArtColour` | ASCII art foreground |
| `aboutTextColour` | `Theme.AboutTextColour` | Version line foreground |
| `aboutLinkColour` | `Theme.AboutLinkColour` | URL hyperlink foreground |
| `aboutHintColour` | `Theme.AboutHintColour` | Close hint foreground |

`UserThemeFile` additionally supports 4 help bar-specific YAML fields:

| YAML field | Maps to | Description |
|------------|---------|-------------|
| `helpKeyColour` | `Theme.HelpKey` | Key binding label foreground (e.g. "ctrl+c") |
| `helpDescColour` | `Theme.HelpDesc` | Key binding description foreground (e.g. "quit") |
| `helpSepColour` | `Theme.HelpSep` | Separator and ellipsis foreground |
| `helpBackgroundColour` | `Theme.HelpBackground` | Full-width background fill behind the help bar |

## Help Toggle

The help bar supports two modes:

| Mode | Description |
|------|-------------|
| Compact (default) | Truncatable key bindings left-aligned, pinned bindings right-aligned, never truncated |
| Full | Shows organised columns of all available key bindings |

Pinned bindings (e.g., `? toggle help`, `q quit`) are always visible in compact mode,
right-aligned, and never truncated. Non-pinned bindings are shown left-aligned and
truncated with an ellipsis (`…`) when they exceed the available width. Keymaps that
do not implement `PinnedKeyMap` render with an empty pinned section (backwards-compatible).

Press `?` to toggle between compact and full help. The toggle is handled at the app
level before any phase sees the key press. The help bar is implemented in
`internal/ui/components/helpbar/`.

---

## Runtime: file disappears (full trace)

```text
dashboard (PhaseDashboard)
└── FileAvailabilityChanged{Files} where project file ∉ Files
    └── app → PhaseWatching
        └── watching view (disconnected mode)
            └── watches for the SAME filename to reappear
                └── file reappears + valid → Parsing → PhaseDashboard
```
