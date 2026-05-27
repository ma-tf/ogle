// Package app implements the root flow orchestrator. It owns the watcher
// lifecycle (creation, subscription, retry, reconnect) and drives the top-level
// flow transitions: startup → dashboard on msgs.ProjectLoaded.
package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/config"
	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/profiling"
	svcdocker "github.com/ma-tf/ogle/internal/services/docker"
	"github.com/ma-tf/ogle/internal/services/docker/connection"
	"github.com/ma-tf/ogle/internal/services/parser"
	"github.com/ma-tf/ogle/internal/services/watcher"
	"github.com/ma-tf/ogle/internal/ui/components/about"
	"github.com/ma-tf/ogle/internal/ui/components/helpbar"
	"github.com/ma-tf/ogle/internal/ui/components/statusbar"
	"github.com/ma-tf/ogle/internal/ui/components/topbar"
	"github.com/ma-tf/ogle/internal/ui/components/watching"
	"github.com/ma-tf/ogle/internal/ui/flows/dashboard"
	"github.com/ma-tf/ogle/internal/ui/flows/startup"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

//nolint:gochecknoglobals // package-level key bindings
var (
	keyQuit       = key.NewBinding(key.WithKeys("ctrl+c"))
	keyProfile    = key.NewBinding(key.WithKeys("ctrl+p"))
	keyAbout      = key.NewBinding(key.WithKeys("f1"))
	keyHelpToggle = key.NewBinding(key.WithKeys("?"))
	keyEsc        = key.NewBinding(key.WithKeys("esc"))
	keyQ          = key.NewBinding(key.WithKeys("q"))
)

const (
	// PhaseStartup is the initial lifecycle phase.
	PhaseStartup = iota
	// PhaseDashboard is the active project dashboard phase.
	PhaseDashboard
	// PhaseWatching is the disconnected watching phase.
	PhaseWatching
)

// Model is the root flow orchestrator.
type Model struct {
	ctx         context.Context
	cfg         config.Config
	configPath  string
	projectFile string
	theme       *theme.Theme
	zm          *zone.Manager
	docker      svcdocker.Docker
	parser      parser.Parser
	watcher     watcher.Watcher

	topbar       topbar.Model
	helpbar      helpbar.Model
	statusbar    statusbar.Model
	startup      startup.Model
	dashboard    dashboard.Model
	watching     watching.Model
	about        about.Model
	showingAbout bool
	phase        int
	width        int
	height       int
}

// New constructs the app Model. Watcher creation is synchronous; if it
// fails the entire program exits with an error.
//
// dockerSvc, parseSvc, and wtr are injected for testability. The caller is
// responsible for constructing the watcher (which requires a scanner.Scanner).
// wtr.Close is returned as the cleanup function.
func New(
	ctx context.Context,
	cfg config.Config,
	configPath string,
	projectFile string,
	th *theme.Theme,
	dockerSvc svcdocker.Docker,
	parseSvc parser.Parser,
	wtr watcher.Watcher,
) (Model, func() error, error) {
	width, height, errSize := term.GetSize(os.Stdout.Fd())
	if errSize != nil {
		width, height = 0, 0
	}

	var (
		project *domain.Project
		dash    dashboard.Model
	)

	currentPhase := PhaseStartup
	zm := zone.New()
	pf := ""

	if projectFile != "" {
		var errParse error
		if project, errParse = parseSvc.Parse(projectFile); errParse != nil {
			_ = wtr.Close()

			return Model{}, nil, fmt.Errorf("parse project file: %w", errParse)
		}

		currentPhase = PhaseDashboard
		pf = filepath.Base(projectFile)

		dash = dashboard.New(
			ctx,
			project,
			th,
			cfg,
			zm,
			filepath.Dir(configPath),
			width,
			height,
			dockerSvc,
			parseSvc,
		)
	}

	return Model{
		ctx:          ctx,
		cfg:          cfg,
		configPath:   configPath,
		projectFile:  pf,
		theme:        th,
		zm:           zm,
		docker:       dockerSvc,
		parser:       parseSvc,
		watcher:      wtr,
		topbar:       topbar.New(ctx, connection.New(), th, dockerSvc, zm),
		helpbar:      helpbar.New(th),
		statusbar:    statusbar.New(th),
		startup:      startup.New(width, height, zm, th, parseSvc),
		dashboard:    dash,
		watching:     watching.New(projectFile, width, height, th, parseSvc),
		about:        about.New(th),
		showingAbout: false,
		phase:        currentPhase,
		width:        width,
		height:       height,
	}, wtr.Close, nil
}

// Init fires the initial snapshot and starts the active phase.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.watcher.Snapshot(), m.topbar.Init(), m.helpbar.Init()}

	switch m.phase {
	case PhaseDashboard:
		cmds = append(cmds, m.dashboard.Init())
		cmds = append(cmds, func() tea.Msg {
			return msgs.TopbarContext{Phase: "dashboard", File: m.projectFile}
		})
	case PhaseStartup:
		cmds = append(cmds, m.startup.Init())
	case PhaseWatching:
	}

	return tea.Batch(cmds...)
}

// Update drives the root state machine. Messages are either handled by app
// directly or dispatched to the active phase model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var topbarCmd, helpbarCmd, statusbarCmd, aboutCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		kpModel, kpCmd := m.handleKeyPress(msg)
		if kpCmd != nil {
			return kpModel, kpCmd
		}

		m = kpModel

	case tea.MouseClickMsg:
		if mcModel, mcCmd := m.handleMouseClick(msg); mcCmd != nil {
			return mcModel, mcCmd
		}

	case msgs.ProjectLoaded:
		return m.handleProjectLoaded(msg)

	case msgs.SettingsApplied:
		return m.handleSettingsApplied(msg)

	case theme.Changed:
		m.theme = msg.Theme

	case profiling.ProfilesDumped:
		if msg.Err != nil {
			return m, func() tea.Msg {
				return msgs.DisplayError{
					Err: fmt.Sprintf("profiling dump failed: %v", msg.Err),
				}
			}
		}

		return m, func() tea.Msg {
			return msgs.DisplayStatus{
				Msg: fmt.Sprintf("profiling dump written: goroutine=%s heap=%s",
					msg.GoroutinePath, msg.HeapPath),
			}
		}

	case msgs.FileAvailabilityChanged:
		return m.handleFileAvailabilityChanged(msg)

	case msgs.FileRemoved:
		m.watching = watching.New(msg.File, m.width, m.height, m.theme, m.parser)
		m.phase = PhaseWatching

		return m, tea.Batch(
			func() tea.Msg { return msgs.TopbarContext{Phase: "watching", File: ""} },
			func() tea.Msg { return msgs.BindingsMsg{Keymap: watchingKeymap{}} },
		)

	case msgs.DisplayError,
		msgs.DisplayStatus,
		msgs.ClearStatusMsg:
		m.statusbar, statusbarCmd = m.statusbar.Update(msg)

		return m, statusbarCmd

	case msgs.AboutVisibilityChanged:
		m.showingAbout = msg.Visible

		return m, nil
	}

	m.topbar, topbarCmd = m.topbar.Update(msg)
	m.helpbar, helpbarCmd = m.helpbar.Update(msg)
	m.statusbar, statusbarCmd = m.statusbar.Update(msg)

	if m.showingAbout {
		m.about, aboutCmd = m.about.Update(msg)
	}

	var cmd tea.Cmd

	switch m.phase {
	case PhaseStartup:
		m.startup, cmd = m.startup.Update(msg)
	case PhaseDashboard:
		m.dashboard, cmd = m.dashboard.Update(msg)
	case PhaseWatching:
		m.watching, cmd = m.watching.Update(msg)
	}

	return m, tea.Batch(cmd, topbarCmd, helpbarCmd, statusbarCmd, aboutCmd)
}

func (m Model) handleSettingsApplied(msg msgs.SettingsApplied) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	th, err := theme.Load(msg.Theme, filepath.Dir(m.configPath))
	if err != nil {
		cmds = append(cmds, func() tea.Msg {
			return msgs.DisplayError{
				Err: fmt.Sprintf("theme: %v", err),
			}
		})
	} else {
		m.theme = th
	}

	m.cfg.Theme = msg.Theme
	m.cfg.LogBufferCap = msg.LogBufferCap

	if saveErr := config.Save(m.configPath, m.cfg); saveErr != nil {
		cmds = append(cmds, func() tea.Msg {
			return msgs.DisplayError{
				Err: fmt.Sprintf("config save: %v", saveErr),
			}
		})
	}

	cmds = append(cmds, func() tea.Msg { return theme.Changed{Theme: m.theme} })

	return m, tea.Batch(cmds...)
}

func (m Model) handleKeyPress(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if m.showingAbout {
		switch {
		case key.Matches(msg, keyAbout), key.Matches(msg, keyEsc), key.Matches(msg, keyQ):
			m.showingAbout = false

			return m, func() tea.Msg { return msgs.AboutVisibilityChanged{Visible: false} }
		}

		return m, nil
	}

	switch {
	case key.Matches(msg, keyQuit):
		return m, tea.Quit
	case key.Matches(msg, keyProfile):
		return m, profiling.DumpCmd()
	case key.Matches(msg, keyHelpToggle):
		var helpbarCmd tea.Cmd

		m.helpbar, helpbarCmd = m.helpbar.Update(helpbar.ToggleMsg{})

		return m, helpbarCmd
	case key.Matches(msg, keyAbout):
		m.showingAbout = true

		return m, func() tea.Msg { return msgs.AboutVisibilityChanged{Visible: true} }
	}

	return m, nil
}

func (m Model) handleProjectLoaded(msg msgs.ProjectLoaded) (Model, tea.Cmd) {
	m.dashboard = dashboard.New(
		m.ctx,
		msg.Project,
		m.theme,
		m.cfg,
		m.zm,
		filepath.Dir(m.configPath),
		m.width,
		m.height,
		m.docker,
		m.parser,
	)
	m.phase = PhaseDashboard

	return m, tea.Batch(
		m.dashboard.Init(),
		func() tea.Msg {
			return msgs.TopbarContext{Phase: "dashboard", File: filepath.Base(msg.Project.File)}
		},
	)
}

func (m Model) handleMouseClick(msg tea.MouseClickMsg) (Model, tea.Cmd) {
	if m.showingAbout {
		return m, nil
	}

	if brand := m.zm.Get(topbar.BrandZone); brand != nil && brand.InBounds(msg) {
		m.showingAbout = true

		return m, func() tea.Msg { return msgs.AboutVisibilityChanged{Visible: true} }
	}

	return m, nil
}

func (m Model) handleFileAvailabilityChanged(
	msg msgs.FileAvailabilityChanged,
) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.phase {
	case PhaseStartup:
		m.startup, cmd = m.startup.Update(msg)
	case PhaseDashboard:
		m.dashboard, cmd = m.dashboard.Update(msg)
	case PhaseWatching:
		m.watching, cmd = m.watching.Update(msg)
	}

	return m, tea.Batch(cmd, m.watcher.Next())
}

// View composes the top bar, active phase body, status bar, and help bar into a unified frame.
func (m Model) View() tea.View {
	var body tea.View

	switch m.phase {
	case PhaseStartup:
		body = m.startup.View()
	case PhaseDashboard:
		body = m.dashboard.View()
	case PhaseWatching:
		body = m.watching.View()
	}

	helpView := m.helpbar.View()
	statusView := m.statusbar.View()

	var barHeight int
	if statusView.Content != "" {
		barHeight = 1
	} else {
		barHeight = max(lipgloss.Height(helpView.Content), 1)
	}

	bodyH := max(0, m.height-1-barHeight)

	parts := []string{
		m.topbar.View().Content,
		lipgloss.NewStyle().
			Width(m.width).
			Height(bodyH).
			Background(m.theme.BodyBackground).
			Render(body.Content),
	}

	if statusView.Content == "" {
		parts = append(parts, helpView.Content)
	} else {
		parts = append(parts, statusView.Content)
	}

	frame := lipgloss.JoinVertical(lipgloss.Top, parts...)

	if m.showingAbout {
		overContent := m.about.View().Content
		overW := lipgloss.Width(overContent)
		overH := lipgloss.Height(overContent)
		overX := max((m.width-overW)/2, 0)  //nolint:mnd // halving to centre overlay
		overY := max((m.height-overH)/2, 0) //nolint:mnd // halving to centre overlay

		frame = lipgloss.NewCompositor(
			lipgloss.NewLayer(frame).X(0).Y(0).Z(0),
			lipgloss.NewLayer(overContent).X(overX).Y(overY).Z(1),
		).Render()
	}

	v := tea.NewView(frame)
	v.Content = m.zm.Scan(v.Content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeAllMotion

	return v
}
