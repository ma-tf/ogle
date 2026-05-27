// Package msgs defines all inter-component tea.Msg types used across the
// Bubble Tea runtime. Types only — no logic, no imports beyond domain.
package msgs

import (
	"charm.land/bubbles/v2/help"

	"github.com/ma-tf/ogle/internal/domain"
)

type (
	// FileAvailabilityChanged is emitted by the watcher whenever compose file
	// presence in the watched directory changes. Files contains the absolute paths
	// of all compose filenames that currently exist on disk. Consumers are
	// responsible for calling parser.Service.Validate on each path before use;
	// the watcher only performs existence checks.
	FileAvailabilityChanged struct {
		Files []string
	}

	// FileSelected is emitted by the fileselect view when the user confirms a file
	// choice from the picker.
	FileSelected struct {
		Path string
	}

	// FileRemoved is emitted by Dashboard when the project file is no longer
	// present in a FileAvailabilityChanged snapshot. App catches it and
	// transitions to phaseWatching.
	FileRemoved struct {
		File string
	}
)

type (
	// ProjectLoaded is emitted by the startup flow after a successful
	// parser.Service.Parse call and signals the app root to transition to the dashboard.
	ProjectLoaded struct {
		Project *domain.Project
	}
)

// ServiceSelected is emitted by the service list component when the cursor
// moves to a new service. ServiceName identifies the selected service.
type ServiceSelected struct {
	ServiceName string
}

type (
	// ServiceStop is emitted when the user triggers stop on a service.
	ServiceStop struct{ ServiceName string }

	// ServiceStart is emitted when the user triggers start on a service.
	ServiceStart struct{ ServiceName string }

	// ServiceRestart is emitted when the user triggers restart on a service.
	ServiceRestart struct{ ServiceName string }

	// ServiceRebuild is emitted when the user triggers rebuild on a service.
	ServiceRebuild struct{ ServiceName string }

	// ServiceActionCompleted is emitted by a docker action cmd when the
	// docker compose subprocess exits, whether successfully or not.
	ServiceActionCompleted struct {
		ServiceName string
		Action      domain.ServiceAction
		Err         error
	}
)

type (
	// DaemonConnected is emitted by the docker service when the Docker daemon ping
	// succeeds. It signals the Dashboard to start State Polling and Log Stream.
	DaemonConnected struct{}

	// DaemonUnavailable is emitted by the docker service when the Docker daemon
	// cannot be reached. The Dashboard shows a retry countdown and freezes Service
	// States at their last-known values.
	DaemonUnavailable struct{ Err error }

	// DaemonTick fires every 1 second during the Docker retry countdown loop.
	DaemonTick struct{}

	// DaemonGraceExpired fires once after the initial connection grace period.
	DaemonGraceExpired struct{}

	// DaemonPoll fires at a regular interval while connected to detect when Docker
	// becomes unavailable.
	DaemonPoll struct{}
)

// TopbarContext is delivered by the app on phase transitions. The topbar
// component updates its displayed context text accordingly.
type TopbarContext struct {
	Phase string
	File  string
}

// BindingsMsg delivers a unified keymap to the helpbar component.
type BindingsMsg struct {
	Keymap help.KeyMap
}

type (
	// LogLinesAvailable signals that new log lines are waiting in the streamer's
	// line channel. The logpane drains the channel on receipt.
	LogLinesAvailable struct{}

	// LogStreamError is emitted when the LogStreamer goroutine hits a read error.
	LogStreamError struct {
		Err         error
		ServiceName string
	}

	// LogStreamContainerNotFound is emitted when the logs endpoint returns 404.
	LogStreamContainerNotFound struct {
		ServiceName string
	}

	// LogStreamRetryTick is emitted after a brief delay when the log stream
	// fails to start (404 or read error). The servicehost handles it by
	// re-calling Start and re-subscribing Next.
	LogStreamRetryTick struct{}
)

// StatePollTick is emitted by servicepanel's poll loop to trigger a compose ps poll.
type StatePollTick struct{}

// ServicesPolled is emitted by the docker service after a "docker compose ps"
// poll completes. Runtimes is nil on error.
type ServicesPolled struct {
	Runtimes map[string]*domain.ServiceRuntimeData
	Err      error
}

type (
	// SettingsApplied is emitted by the settings component on every field adjustment.
	// app.Model handles it to load the theme, persist config, and emit ThemeChanged.
	SettingsApplied struct {
		Theme        string
		LogBufferCap int
	}

	// SettingsVisibilityChanged is emitted by settings when the user closes the
	// overlay. dashboard.Model tracks the visibility flag.
	SettingsVisibilityChanged struct {
		Visible bool
	}

	// AboutVisibilityChanged is emitted by the about overlay when the user opens
	// or closes it. app.Model tracks the visibility flag.
	AboutVisibilityChanged struct {
		Visible bool
	}
)

// ClearLogBuffer clears all buffered log lines for the named service.
type ClearLogBuffer struct {
	ServiceName string
}

// ToggleLogWrap toggles soft wrapping of log lines in all log panes.
type ToggleLogWrap struct{}

type (
	// DisplayError asks the app chrome to show err in the status bar for 3 seconds.
	DisplayError struct {
		Err string
	}

	// DisplayStatus asks the app chrome to show msg in the status bar for 3 seconds.
	DisplayStatus struct {
		Msg string
	}

	// ClearStatusMsg dismisses the status bar message.
	ClearStatusMsg struct{}
)
