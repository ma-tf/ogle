// Package servicepanel manages a set of per-service hosts and their polling
// lifecycle. It renders all hosts as compositor layers stacked vertically.
package servicepanel

import (
	"net/http"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/docker/logs"
	"github.com/ma-tf/ogle/internal/ui/components/servicehost"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// Option configures a Model.
type Option func(*Model)

// WithStreamerHTTPClient sets the HTTP client used by every log streamer
// created for this panel. If nil (the default), each streamer dials the Docker
// Unix socket directly.
func WithStreamerHTTPClient(client *http.Client) Option {
	return func(m *Model) {
		m.streamerClient = client
	}
}

// Model manages a set of per-service hosts and the state polling lifecycle.
type Model struct {
	hosts          []servicehost.Model
	theme          *theme.Theme
	pollerStarted  bool
	streamerClient *http.Client
}

// New constructs a Model with one host per project service.
func New(project *domain.Project, th *theme.Theme, w, h, logBufferCap int, opts ...Option) Model {
	var m Model
	for _, opt := range opts {
		opt(&m)
	}

	hosts := make([]servicehost.Model, len(project.Services))

	for i, svc := range project.Services {
		streamerOpts := []logs.Option{}
		if m.streamerClient != nil {
			streamerOpts = append(streamerOpts, logs.WithHTTPClient(m.streamerClient))
		}

		streamer := logs.New(svc.Name, streamerOpts...)

		hosts[i] = servicehost.New(th, svc, project.Name, w, h, logBufferCap, streamer)
	}

	return Model{
		hosts:          hosts,
		theme:          th,
		pollerStarted:  false,
		streamerClient: m.streamerClient,
	}
}

// Init starts all hosts' flush ticks.
func (m Model) Init() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.hosts))
	for i := range m.hosts {
		cmds[i] = m.hosts[i].Init()
	}

	return tea.Batch(cmds...)
}

// Update handles poll lifecycle messages and forwards everything else to hosts.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0, len(m.hosts)+1)

	switch msg := msg.(type) {
	case theme.Changed:
		m.theme = msg.Theme

	case msgs.DaemonConnected:
		if !m.pollerStarted {
			m.pollerStarted = true
			cmds = append(cmds, m.pollStateCmd())
		}

	case msgs.StatePollTick:
		cmds = append(cmds, m.pollStateCmd())
	}

	for i := range m.hosts {
		var cmd tea.Cmd

		m.hosts[i], cmd = m.hosts[i].Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders all hosts as compositor layers.
func (m Model) View() tea.View {
	lyrs := make([]*lipgloss.Layer, len(m.hosts))
	for i, h := range m.hosts {
		lyrs[i] = lipgloss.NewLayer(h.View().Content).X(0).Y(0).Z(i)
	}

	return tea.NewView(lipgloss.NewCompositor(lyrs...).Render())
}

func (m Model) pollStateCmd() tea.Cmd {
	return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return msgs.StatePollTick{}
	})
}
