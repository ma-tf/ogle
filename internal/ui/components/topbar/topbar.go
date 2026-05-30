package topbar

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/docker"
	"github.com/ma-tf/ogle/internal/services/docker/connection"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	// BrandZone is the zone identifier for the clickable "ogle" brand text.
	BrandZone = "topbar-brand"

	gracePeriodDuration = 10 * time.Second
	healthCheckInterval = 2 * time.Second
)

// Phase identifies the active UI phase for context text rendering.
type Phase int

const (
	// PhaseStartup is the initial startup phase.
	PhaseStartup Phase = iota
	// PhaseDashboard is the dashboard phase.
	PhaseDashboard
	// PhaseWatching is the file-watching phase.
	PhaseWatching
)

// Model holds top bar state: the active phase, project file, selected service,
// wrap/overflow status, daemon connection machine, spinner, theme, zone
// manager, and terminal width.
type Model struct {
	phase           Phase
	projectFile     string
	selectedService string
	wrapOn          bool
	truncated       bool
	conn            *connection.Machine
	docker          docker.Docker
	spn             spinner.Model
	th              *theme.Theme
	zm              *zone.Manager
	width           int
	ctx             context.Context
}

// New returns a Model in PhaseStartup with no project file.
func New(
	ctx context.Context,
	conn *connection.Machine,
	th *theme.Theme,
	d docker.Docker,
	zm *zone.Manager,
) Model {
	return Model{
		phase:           PhaseStartup,
		projectFile:     "",
		selectedService: "",
		wrapOn:          false,
		truncated:       false,
		ctx:             ctx,
		conn:            conn,
		docker:          d,
		spn:             spinner.New(spinner.WithSpinner(spinner.MiniDot)),
		th:              th,
		zm:              zm,
		width:           0,
	}
}

// Init fires the initial Docker connect, grace-period tick, and spinner tick.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.docker.Connect(m.ctx),
		tea.Tick(gracePeriodDuration, func(_ time.Time) tea.Msg {
			return msgs.DaemonGraceExpired{}
		}),
		m.spn.Tick,
	)
}

// Update handles daemon connectivity messages, spinner ticks, window
// resize events, and topbar context changes.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case msgs.TopbarContext:
		m = m.handleTopbarContext(msg)

	case msgs.LogWrapStatus:
		if msg.ServiceName == m.selectedService {
			m.wrapOn = msg.On
			m.truncated = msg.Overflow
		}

	case theme.Changed:
		m.th = msg.Theme

	case msgs.DaemonConnected,
		msgs.DaemonUnavailable,
		msgs.DaemonGraceExpired,
		msgs.DaemonTick,
		msgs.DaemonPoll:
		var cmds []tea.Cmd

		m, cmds = m.handleDaemonMsg(msg)

		return m, tea.Batch(cmds...)

	case spinner.TickMsg:
		var cmd tea.Cmd

		m.spn, cmd = m.spn.Update(msg)

		return m, cmd
	}

	return m, nil
}

func (m Model) handleTopbarContext(msg msgs.TopbarContext) Model {
	switch msg.Phase {
	case "startup":
		m.phase = PhaseStartup
	case "dashboard":
		m.phase = PhaseDashboard
	case "watching":
		m.phase = PhaseWatching
	}

	m.projectFile = msg.File
	if msg.Service != m.selectedService {
		m.wrapOn = false
		m.truncated = false
	}

	m.selectedService = msg.Service

	return m
}

func (m Model) handleDaemonMsg(msg tea.Msg) (Model, []tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.(type) {
	case msgs.DaemonConnected:
		m.conn.HandleConnected()

		cmds = append(cmds, pollDaemonCmd())

	case msgs.DaemonUnavailable:
		m.conn.HandleUnavailable(time.Now().UTC())

		cmds = append(cmds, daemonTickCmd())

	case msgs.DaemonGraceExpired:
		if m.conn.HandleGracePeriodExpired(time.Now().UTC()) {
			cmds = append(cmds, daemonTickCmd())
		}

	case msgs.DaemonTick:
		if m.conn.ConnectState() == connection.ConnectStateUnavailable {
			if m.conn.IsRetryDue(time.Now().UTC()) {
				cmds = append(cmds, m.docker.Connect(m.ctx))
			} else {
				cmds = append(cmds, daemonTickCmd())
			}
		}

	case msgs.DaemonPoll:
		if m.conn.ConnectState() == connection.ConnectStateConnected {
			cmds = append(cmds, m.docker.Connect(m.ctx))
		}
	}

	return m, cmds
}

func (m Model) contextText() string {
	switch m.phase {
	case PhaseStartup:
		return "scanning for compose files"
	case PhaseDashboard:
		if m.selectedService != "" {
			return m.projectFile + " → " + m.selectedService
		}

		return m.projectFile
	case PhaseWatching:
		return "disconnected"
	default:
		return ""
	}
}

func (m Model) renderDaemonStatus() string {
	switch m.conn.ConnectState() {
	case connection.ConnectStateConnecting:
		return lipgloss.NewStyle().
			Foreground(m.th.TopbarStatusText).
			Background(m.th.TopbarRetryBackground).
			Render(" 🐳 ○ RECONNECTING " + m.spn.View() + " ")

	case connection.ConnectStateConnected:
		return lipgloss.NewStyle().
			Foreground(m.th.TopbarStatusText).
			Background(m.th.StateRunning).
			Render(" 🐳 ● LIVE ")

	case connection.ConnectStateUnavailable:
		secs := int(math.Ceil(m.conn.Remaining().Seconds()))
		countdown := "(now)"

		if secs >= 1 {
			countdown = fmt.Sprintf("(%ds)", secs)
		}

		label := lipgloss.NewStyle().
			Foreground(m.th.TopbarStatusText).
			Background(m.th.TopbarDisconnectedBackground).
			Render(" 🐳 ○ DISCONNECTED")
		counter := lipgloss.NewStyle().
			Foreground(m.th.TopbarStatusText).
			Background(m.th.TopbarDisconnectedBackground).
			Render(" " + countdown + " ")

		return label + counter
	default:
		return ""
	}
}

func (m Model) renderBadge() string {
	if m.wrapOn {
		return lipgloss.NewStyle().
			Foreground(m.th.TopbarStatusText).
			Background(m.th.TopbarWrapBackground).
			Render(" WRAP ")
	}

	if m.truncated {
		return lipgloss.NewStyle().
			Foreground(m.th.TopbarStatusText).
			Background(m.th.TopbarTruncBackground).
			Render(" >> ")
	}

	return ""
}

// View renders the top bar: clickable "ogle" brand + phase context on the left,
// wrap/truncation badges (optional), Docker daemon status on the right, all
// right-aligned via padding.
func (m Model) View() tea.View {
	bg := m.th.TopbarBackground
	brandStyle := lipgloss.NewStyle().
		Foreground(m.th.TopbarBrandText).
		Background(m.th.TopbarBrandBackground)

	brand := m.zm.Mark(BrandZone, brandStyle.Render(" ogle "))

	contextStyle := lipgloss.NewStyle().Foreground(m.th.TopbarContextText).Background(bg)
	spacerStyle := lipgloss.NewStyle().Background(bg)

	left := brand + contextStyle.Render("  "+m.contextText())
	badge := m.renderBadge()
	right := m.renderDaemonStatus()

	leftW := lipgloss.Width(left)
	badgeW := lipgloss.Width(badge)
	rightW := lipgloss.Width(right)
	pad := max(m.width-leftW-badgeW-rightW, 0)

	return tea.NewView(left + spacerStyle.Render(strings.Repeat(" ", pad)) + badge + right)
}

func daemonTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return msgs.DaemonTick{}
	})
}

func pollDaemonCmd() tea.Cmd {
	return tea.Tick(healthCheckInterval, func(_ time.Time) tea.Msg {
		return msgs.DaemonPoll{}
	})
}
