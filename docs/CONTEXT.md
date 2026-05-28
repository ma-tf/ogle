# ogle

A terminal UI for observing and operating Docker Compose projects — no setup required.

## Language

### Compose file concepts

**Compose File**:
The YAML file on disk that defines a Docker Compose project (e.g., `compose.yaml`, `docker-compose.yml`).
_Avoid_: Project file, compose config

**Project**:
The parsed, named in-memory representation of a Compose File, including its name and list of declared Services.
_Avoid_: Compose project, workspace

**Service**:
A named unit declared in the Compose File that maps to a running (or stopped) container. Spans both the declaration and
its runtime instance.
_Avoid_: Container (reserve "container" for implementation-level discussion, e.g., container ID or Docker API calls)

**Service State**:
The current Docker container state for a Service: `running`, `exited`, `paused`, `restarting`, `dead`, or `not created`
(declared in the Compose File but no container exists yet).
_Avoid_: Status, health (health check result is separate from container state — see **Service Health**)

**Service Health**:
The Docker health check result for a Service: `healthy`, `unhealthy`, `starting`, or `no healthcheck`. Distinct from
Service State — a Service can be `running` but `unhealthy`. Planned feature; not yet implemented: no health polling
or display exists in the current codebase.
_Avoid_: Health status, container health

**State Age**:
The time elapsed since a Service entered its current Service State. Displayed for all states — e.g. `"up 2h"` when
running, `"exited 3m ago"` when not. Resets on every state transition.
_Avoid_: Uptime (running-only connotation), downtime (non-running-only connotation)

**Orphan**:
A running container that has no corresponding Service in the current Project. Typically the result of removing a Service
from the Compose File while its container is still running.
_Avoid_: Orphan container (redundant — all Orphans are containers by definition)

**Orphan Toggle**:
The user action that shows or hides Orphans in the service list on the Dashboard. Planned feature; not yet implemented.
_Avoid_: Show orphans, orphan visibility

### User interaction

**Dashboard**:
The main screen displayed after a Project is loaded. Shows all Services, their states, and the Selected Service's Log
Stream in the Service Inspector.
_Avoid_: Monitor view, main view, service view

**Service Layer**:
A persistent compositor layer that is the unit of observation for a single Service (or Orphan). Owns a Service
Inspector, Log Stream, Log Buffer, and scroll state. Created eagerly when a Project loads; destroyed when a Service is
removed via Live Reload. All Service Layers are peers — none is foreground or background.
_Avoid_: log pane, inspector pane

**Service Inspector**:
The view component within a Service Layer: a compact detail header (service name, image, ports, container hash, Service
State, State Age) above the Log Stream area. The detail header renders Compose File fields immediately;
Docker fields show `—` until the daemon is connected. (Future: will also show Service Health when health polling is
implemented.)
_Avoid_: Logs pane, detail pane, right pane

**Selected Service**:
The Service whose Service Layer is currently on top of the compositor stack.
_Avoid_: Active service, focused service

**Log Stream**:
The live, tailing output of a Selected Service's logs, streamed in real time from Docker.
_Avoid_: Logs (too generic), log tail

**Log Buffer**:
The bounded in-memory store of log lines for the Selected Service. Capped at a configurable maximum; when exceeded, the
oldest lines are discarded to maintain performance.
_Avoid_: Log history, log cache

**State Polling**:
The background process that periodically queries Docker for the current Service State of each Service. Runs on a
configurable interval.
_Avoid_: Polling (unqualified), refresh, state sync

**Service Filter**:
The interactive mode that narrows the Service list to entries whose name matches a user-supplied substring. Activated
with `/` on the Dashboard. Planned feature; not yet implemented.
_Avoid_: Log filter (distinct feature), search

**Label Toggle**:
The user action that globally shows or hides the `ogle.*` label section in the Service Inspector. Hidden by default. The
section is fixed-size, scrollable, and focusable as a sub-focus within the Service Inspector. Planned feature; not yet
implemented.
_Avoid_: Metadata toggle, label visibility

**Log Filter**:
The interactive mode that narrows the Log Stream to lines matching a user-supplied substring. Planned feature; not yet
implemented.
_Avoid_: Log search, service filter (distinct feature)

**Service Action**:
A user-initiated operation applied to a Service: stop, start, restart, or rebuild. Executed asynchronously without
blocking the UI.
_Avoid_: Command (overloaded in the Bubble Tea runtime context), operation

**Settings**:
An in-session overlay that lets the user adjust configuration values (e.g., theme, log buffer cap) without
leaving the TUI or editing the Config File. Rendered as a full-terminal compositor layer over the Dashboard, which
remains live underneath. Changes take effect immediately and are persisted to the Config File on every field
adjustment via `config.Save`. Theme selection supports 8 built-in names: `default`, `default_light`,
`catppuccino_frappe`, `catppuccino_latte`, `catppuccino_macchiato`, `catppuccino_mocha`, `solarized_dark`,
`solarized_light`. Users can also drop custom theme files at `~/.ogle/themes/<name>.yaml` that override a built-in
base theme. Navigate fields with ↑/↓, adjust values with ←/→.
_Avoid_: Config, preferences — also distinct from instant keybinding toggles (e.g., Orphan Toggle) which take effect
immediately with no overlay

**Help Toggle**:
The user action that switches the help bar between compact mode (up to 5 essential key bindings) and full mode
(organised columns of all available key bindings). Activated with `?` on any screen. The toggle is handled at the
app level before any phase sees the key press.
_Avoid_: Show help, help mode

**About**:
An overlay that displays ogle version information, ASCII art logo, and a GitHub URL. Opened with `F1` or by
clicking the brand text (the word "ogle" in the top bar). Closed with `F1`, `esc`, or `q`. The About overlay is
a cross-phase UI layer — available from any phase (startup, dashboard, or watching). While open, phase-specific
key handling is blocked.
_Avoid_: Version dialog, info popup

### Startup

**File Discovery**:
The startup process of scanning the working directory for Compose Files, validating each candidate, and selecting one to
load as a Project.
_Avoid_: Scanning, auto-detection, file search

**Explicit File**:
The startup mode where the user provides the Compose File path directly (via `-f`), bypassing File Discovery entirely.
Validation failures in this mode are hard exits before the TUI opens.
_Avoid_: Manual file, specified file

**Watching**:
The startup state where File Discovery found no valid Compose Files and ogle is monitoring the working directory for one
to appear.
_Avoid_: Waiting, idle — also distinct from **Disconnected**, which waits for a _specific_ file to return

**Watcher Error**:
A recoverable failure state where ogle cannot monitor the working directory (e.g., permissions problem, directory
missing). The user is shown an error message and must explicitly retry.
_Avoid_: Watcher failure, monitor error

**Project Selector**:
The screen shown during startup when File Discovery finds two or more valid Compose Files. Lets the user choose which
Project to load.
_Avoid_: File picker, file selector

**Live Reload**:
The automatic re-parse of the Compose File and silent update of the Dashboard when the Compose File changes on disk,
without interrupting the session.
_Avoid_: Hot reload, refresh, file sync

**Parse Error**:
The condition where the Compose File exists on disk but cannot be successfully parsed into a Project. At startup, shown
inline on the current screen; at runtime (during Live Reload), shown as a persistent banner over the Dashboard with the
last-known state preserved.
_Avoid_: Invalid file, broken compose, YAML error

**Disconnected**:
The state ogle enters when the monitored Compose File disappears at runtime. ogle waits for the same file to reappear
before resuming the Dashboard.
_Avoid_: Offline, paused, suspended

**Docker Unavailable**:
The condition where the Docker daemon cannot be reached at runtime — distinct from **Disconnected**, which refers to the
Compose File disappearing. When Docker Unavailable, the topbar shows a placeholder with a live retry
countdown; Service States freeze at last-known values; Service Actions are disabled. ogle retries automatically.
_Avoid_: Disconnected (reserved for the Compose File disappearing), daemon unreachable

## Relationships

- A **Project** declares one or more **Services**.
- **File Discovery** finds the **Compose File** and parses it into a **Project**. If no valid Compose File is found,
ogle enters the **Watching** state.
- When 2+ valid Compose Files are found, the **Project Selector** lets the user choose which to load.
- The **Dashboard** displays all Services and the **Selected Service**'s **Log Stream** inside the **Service
Inspector**. Each Service is backed by a **Service Layer** in the compositor.
- The **Service Inspector** shows the Selected Service's detail header and Log Stream. Compose File fields are always
visible; Docker fields (`Service State`, `State Age`, container hash) require a live Docker
connection. (Future: will also query `Service Health` when health polling is implemented.)
- **State Polling** periodically updates each Service's **Service State** and **State Age**. (Future: will also
  update **Service Health** when health polling is implemented.)
- A user triggers a **Service Action** on a Service from the Dashboard; actions run asynchronously and do not block the
UI. Service Actions are disabled when **Docker Unavailable**.
- When the Compose File changes on disk, **Live Reload** updates the Project without leaving the Dashboard.
- When the Compose File disappears at runtime, the Dashboard enters the **Disconnected** state and waits for that
specific file to reappear.
- When the Docker daemon becomes unreachable, the Dashboard enters **Docker Unavailable**: the **topbar**
shows a retry countdown and Service States freeze at last-known values. ogle retries automatically.
- An **Orphan** appears alongside Services in the Dashboard but is not part of the Project. (Future: the **Orphan Toggle**
will control whether Orphans are shown.)
- (Future: The **Label Toggle** will control whether the `ogle.*` label section is shown in the **Service Inspector**.)

## Example dialogue

> **Dev:** "If the user has two compose files, which Project do they load?"
> **Domain expert:** "The **Project Selector** appears. They pick one and the **Dashboard** opens."
>
> **Dev:** "What happens if they edit the compose file while the Dashboard is open?"
> **Domain expert:** "**Live Reload** — the Dashboard re-parses it silently. If the file disappears entirely, the
Dashboard goes **Disconnected** and waits for that specific file to come back."
>
> **Dev:** "Can you stop a service from the Dashboard?"
> **Domain expert:** "Yes — trigger a **Service Action** (stop). It runs in the background; the UI stays responsive.
**State Polling** picks up the state change in the next poll cycle."
>
> **Dev:** "There's a container running that isn't in the compose file anymore — what is it?"
> **Domain expert:** "An **Orphan**. It shows up in the Dashboard alongside the Services but it's not part of the
**Project**."
>
> **Dev:** "The Docker daemon crashed. What does the user see?"
> **Domain expert:** "The **Service Inspector** shows a placeholder: 'Docker unavailable — retrying in Xs…'. The service
list freezes at the last-known **Service States**. **Service Actions** are gone from the help bar. Once Docker recovers,
everything resumes automatically."
>
> **Dev:** "Is the service 'healthy'?"
> **Domain expert:** "That depends which concept you mean. **Service Health** is the Docker health check result —
> `healthy` or `unhealthy`. **Service State** is whether the container is `running`, `exited`, etc. A service can be
> `running` but `unhealthy`. (Note: Service Health is defined here for domain clarity — not yet implemented.)"
>
> **Dev:** "How long has this service been up?"
> **Domain expert:** "Check the **State Age** — it shows how long the service has been in its current state. If it's
running, you see 'up 2h'. If it's exited, you see 'exited 3m ago'."

## Flagged ambiguities

- **"Zero configuration"** was initially proposed in the aim statement. Resolved: ogle supports optional configuration
via config files and environment variables, so the accurate claim is _"no setup required"_ — it works out of the box but
can be configured.
- **"Container" vs "Service"** — ogle uses **Service** as the user-facing term that spans both the Compose File
declaration and its runtime container. "Container" is reserved for implementation-level precision (e.g., when targeting
a specific container ID for log streaming or Docker API calls).
- **"Disconnected" vs "Docker Unavailable"** — "Disconnected" is reserved exclusively for the Compose File disappearing.
"Docker Unavailable" is the term for the Docker daemon being unreachable. They are distinct states with different
recovery paths and different UI treatments.
