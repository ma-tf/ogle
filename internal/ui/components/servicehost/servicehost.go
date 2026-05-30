package servicehost

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/docker/logs"
	"github.com/ma-tf/ogle/internal/ui/components/logpane"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	baseRetryDelay = 2 * time.Second
	maxRetryDelay  = 30 * time.Second
)

// Model wraps a per-service log pane and streamer into a compositor-hostable unit.
type Model struct {
	ctx             context.Context
	def             domain.ServiceDef
	logPane         logpane.Model
	streamer        logs.Streamer
	streamerStarted bool
	theme           *theme.Theme
	project         string
	selected        bool
	retryCount      int
}

// New constructs a host for the given service.
func New(
	ctx context.Context,
	th *theme.Theme,
	def domain.ServiceDef,
	project string,
	w, h, logBufferCap int,
	streamer logs.Streamer,
) Model {
	return Model{
		ctx:             ctx,
		def:             def,
		logPane:         logpane.New(th, w, h, logBufferCap, streamer.Lines()),
		streamer:        streamer,
		streamerStarted: false,
		theme:           th,
		project:         project,
		selected:        false,
		retryCount:      0,
	}
}

// Init batches the init cmds of all children.
func (m Model) Init() tea.Cmd {
	return m.logPane.Init()
}

// Update routes messages to children.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case msgs.ReportWrapStatus:
		if msg.ServiceName != m.def.Name {
			return m, nil
		}

	case msgs.ClearLogBuffer:
		if msg.ServiceName != m.def.Name {
			return m, nil
		}

	case msgs.ServiceSelected:
		m.selected = (msg.ServiceName == m.def.Name)

		return m, nil

	case tea.KeyPressMsg, tea.MouseWheelMsg:
		if !m.selected {
			return m, nil
		}

	case msgs.ServicesPolled:
		var cmd tea.Cmd

		m, cmd = m.handleServicesPolled(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case msgs.LogLinesAvailable,
		msgs.LogStreamContainerNotFound,
		msgs.LogStreamError,
		msgs.LogStreamRetryTick:
		var streamCmds []tea.Cmd

		m, streamCmds = m.handleStreamEvent(msg)
		cmds = append(cmds, streamCmds...)

	case theme.Changed:
		m.theme = msg.Theme
	}

	var logCmd tea.Cmd

	m.logPane, logCmd = m.logPane.Update(msg)
	if logCmd != nil {
		logCmd = wrapLogWrapStatus(logCmd, m.def.Name)
		cmds = append(cmds, logCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleServicesPolled(msg msgs.ServicesPolled) (Model, tea.Cmd) {
	if msg.Err != nil {
		return m, nil
	}

	rt, hasRuntime := msg.Runtimes[m.def.Name]
	isRunning := hasRuntime && rt.State == domain.ServiceStateRunning

	if isRunning && !m.streamerStarted {
		return m.startStreamer()
	}

	if !isRunning && m.streamerStarted {
		m.streamer.Close()
		m.streamerStarted = false
		m.retryCount = 0
	}

	return m, nil
}

func (m Model) handleStreamEvent(msg tea.Msg) (Model, []tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.LogLinesAvailable:
		if msg.ServiceName != "" && msg.ServiceName != m.def.Name {
			return m, nil
		}

		m.retryCount = 0

		return m, []tea.Cmd{m.streamer.Next()}

	case msgs.LogStreamContainerNotFound:
		m.streamer.Close()
		m.streamerStarted = false
		m.retryCount = 0

		return m, nil

	case msgs.LogStreamError:
		m.streamer.Close()
		m.streamerStarted = false

		m.retryCount++

		return m, []tea.Cmd{
			tea.Tick(m.retryDelay(), func(_ time.Time) tea.Msg {
				return msgs.LogStreamRetryTick{}
			}),
		}

	case msgs.LogStreamRetryTick:
		if !m.streamerStarted {
			var cmd tea.Cmd

			m, cmd = m.startStreamer()

			return m, []tea.Cmd{cmd}
		}

		return m, nil
	}

	return m, nil
}

// wrapLogWrapStatus intercepts a LogWrapStatus command and injects the given
// service name so the topbar can filter by active service.
func wrapLogWrapStatus(cmd tea.Cmd, name string) tea.Cmd {
	return func() tea.Msg {
		msg := cmd()
		if msg == nil {
			return nil
		}

		lws, ok := msg.(msgs.LogWrapStatus)
		if !ok {
			return msg
		}

		lws.ServiceName = name

		return lws
	}
}

func (m Model) retryDelay() time.Duration {
	d := baseRetryDelay * (1 << m.retryCount)
	if d > maxRetryDelay {
		return maxRetryDelay
	}

	return d
}

// startStreamer begins log streaming for the service and returns a Next cmd.
func (m Model) startStreamer() (Model, tea.Cmd) {
	m.streamerStarted = true
	containerName := logs.ContainerName(m.project, m.def.Name, m.def.ContainerName)
	m.streamer.Start(m.ctx, containerName)

	return m, m.streamer.Next()
}

// View returns the log pane for the selected service, or an empty view.
func (m Model) View() tea.View {
	if !m.selected {
		return tea.NewView("")
	}

	return m.logPane.View()
}
