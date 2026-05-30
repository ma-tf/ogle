// Package app implements the root flow orchestrator. It owns the watcher
// lifecycle (creation, subscription, retry, reconnect) and drives the top-level
// flow transitions: startup → dashboard on msgs.ProjectLoaded.
package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"charm.land/bubbles/v2/help"
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

const (
	chromeTopbar      = 1 // topbar is always 1 line
	chromeStatusbar   = 1 // statusbar is 1 line
	chromeCompactHelp = 1 // compact helpbar is 1 line

	// motionThrottleInterval is the minimum interval between processed mouse-motion
	// events. Events arriving faster are dropped to reduce render pressure.
	motionThrottleInterval = 33 * time.Millisecond

	// profileDuration is the CPU profile collection window when ctrl+p is pressed.
	profileDuration = 30 * time.Second
)

type viewCache struct {
	gen        uint64
	lastGen    uint64
	cachedView tea.View
}

// flushMotionMsg is sent by a debounce tick to flush the latest buffered
// mouse-motion position after the throttle window expires.
type flushMotionMsg struct{}

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

	statusActive    bool
	helpExpanded    bool
	keymap          help.KeyMap
	lastFrameHeight int
	lastMotionTime  time.Time
	pendingMotion   *tea.MouseMotionMsg
	cache           *viewCache
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
		ctx:             ctx,
		cfg:             cfg,
		configPath:      configPath,
		projectFile:     pf,
		theme:           th,
		zm:              zm,
		docker:          dockerSvc,
		parser:          parseSvc,
		watcher:         wtr,
		topbar:          topbar.New(ctx, connection.New(), th, dockerSvc, zm),
		helpbar:         helpbar.New(th),
		statusbar:       statusbar.New(th),
		startup:         startup.New(width, height, zm, th, parseSvc),
		dashboard:       dash,
		watching:        watching.New(projectFile, width, height, th, parseSvc),
		about:           about.New(th),
		showingAbout:    false,
		phase:           currentPhase,
		width:           width,
		height:          height,
		statusActive:    false,
		helpExpanded:    false,
		keymap:          nil,
		lastFrameHeight: 0,
		lastMotionTime:  time.Time{},
		pendingMotion:   nil,
		cache: &viewCache{
			gen:        1,
			lastGen:    0,
			cachedView: tea.View{}, //nolint:exhaustruct // zero value is sentinel for "not yet cached"
		},
	}, wtr.Close, nil
}

// Init fires the initial snapshot and starts the active phase.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.watcher.Snapshot(), m.topbar.Init(), m.helpbar.Init()}

	switch m.phase {
	case PhaseDashboard:
		cmds = append(cmds, m.dashboard.Init())
		cmds = append(cmds, func() tea.Msg {
			return msgs.TopbarContext{Phase: "dashboard", File: m.projectFile, Service: ""}
		})
	case PhaseStartup:
		cmds = append(cmds, m.startup.Init())
	case PhaseWatching:
	}

	return tea.Batch(cmds...)
}

// Update drives the root state machine. Messages are either handled by app
// directly or dispatched to the active phase model.
//
// Mouse-motion events are debounced to motionThrottleInterval (~30 fps). The
// latest position during the window is buffered and flushed via a tick so the
// final stopping position always renders. Throttled events return early without
// a gen bump, so View() returns its cached output.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m, cmd, handled := m.tryThrottleMotion(msg); handled {
		return m, cmd
	}

	if _, ok := msg.(flushMotionMsg); ok {
		return m.handleFlushMotion(msg)
	}

	m.cache.gen++

	return m.handleMessage(msg)
}

// handleFlushMotion dispatches the latest buffered mouse position without
// rearming the debounce tick — the next user motion will start a fresh window.
func (m Model) handleFlushMotion(_ tea.Msg) (tea.Model, tea.Cmd) {
	if m.pendingMotion == nil {
		return m, nil
	}

	m.lastMotionTime = time.Now()
	m.cache.gen++

	pending := *m.pendingMotion
	m.pendingMotion = nil

	return m.dispatchToComponents(pending)
}

// handleMessage routes a non-throttled message to the appropriate handler or
// dispatches it to sub-components.
func (m Model) handleMessage(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		return m.handleProfilesDumped(msg)

	case msgs.FileAvailabilityChanged:
		return m.handleFileAvailabilityChanged(msg)

	case msgs.FileRemoved:
		return m.handleFileRemoved(msg)

	case msgs.DisplayError,
		msgs.DisplayStatus,
		msgs.ClearStatusMsg:
		var statusbarCmd, frameCmd tea.Cmd

		m.statusbar, statusbarCmd = m.statusbar.Update(msg)
		m.statusActive = m.statusbar.View().Content != ""

		m, frameCmd = m.frameHeightCmd()

		return m, tea.Batch(statusbarCmd, frameCmd)

	case msgs.AboutVisibilityChanged:
		m.showingAbout = msg.Visible

		return m, nil

	case msgs.BindingsMsg:
		m.keymap = msg.Keymap
	}

	return m.dispatchToComponents(msg)
}

// tryThrottleMotion checks whether msg is a MouseMotionMsg that falls within
// the throttle window. If so it buffers the latest position and schedules a
// flush tick so the final mouse position is never lost. For a non-throttled
// motion event it dispatches immediately and arms the next flush.
func (m Model) tryThrottleMotion(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	if _, ok := msg.(tea.MouseMotionMsg); !ok {
		return m, nil, false
	}

	if !m.lastMotionTime.IsZero() && time.Since(m.lastMotionTime) < motionThrottleInterval {
		mm, _ := msg.(tea.MouseMotionMsg)
		m.pendingMotion = &mm

		return m, nil, true
	}

	m.lastMotionTime = time.Now()
	m.pendingMotion = nil
	m.cache.gen++

	model, cmd := m.dispatchToComponents(msg)

	return model, tea.Batch(cmd, tea.Tick(motionThrottleInterval, func(_ time.Time) tea.Msg {
		return flushMotionMsg{}
	})), true
}

func (m Model) handleProfilesDumped(msg profiling.ProfilesDumped) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		return m, func() tea.Msg {
			return msgs.DisplayError{
				Err: fmt.Sprintf("profiling dump failed: %v", msg.Err),
			}
		}
	}

	parts := "profiling dump written:"

	if msg.CPUProfilePath != "" {
		parts += " cpu=" + msg.CPUProfilePath
	}

	if msg.GoroutinePath != "" {
		parts += " goroutine=" + msg.GoroutinePath
	}

	if msg.HeapPath != "" {
		parts += " heap=" + msg.HeapPath
	}

	return m, func() tea.Msg {
		return msgs.DisplayStatus{Msg: parts}
	}
}

func (m Model) handleFileRemoved(msg msgs.FileRemoved) (tea.Model, tea.Cmd) {
	var frameCmd tea.Cmd

	m.watching = watching.New(msg.File, m.width, m.height, m.theme, m.parser)
	m.phase = PhaseWatching
	m.statusActive = false

	m, frameCmd = m.frameHeightCmd()

	return m, tea.Batch(
		func() tea.Msg { return msgs.TopbarContext{Phase: "watching", File: "", Service: ""} },
		func() tea.Msg { return msgs.BindingsMsg{Keymap: watchingKeymap{}} },
		frameCmd,
	)
}

func (m Model) dispatchToComponents(msg tea.Msg) (tea.Model, tea.Cmd) {
	var topbarCmd, helpbarCmd, statusbarCmd, aboutCmd, frameCmd tea.Cmd

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

	m, frameCmd = m.frameHeightCmd()

	return m, tea.Batch(cmd, topbarCmd, helpbarCmd, statusbarCmd, aboutCmd, frameCmd)
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
		return m, tea.Batch(
			func() tea.Msg {
				return msgs.DisplayStatus{Msg: "Profiling: mouse around for 30s…"}
			},
			profiling.DumpAllCmd(profileDuration),
		)
	case key.Matches(msg, keyHelpToggle):
		var helpbarCmd, frameCmd tea.Cmd

		m.helpbar, helpbarCmd = m.helpbar.Update(helpbar.ToggleMsg{})
		m.helpExpanded = !m.helpExpanded

		m, frameCmd = m.frameHeightCmd()

		return m, tea.Batch(helpbarCmd, frameCmd)
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
	m.statusActive = false

	var frameCmd tea.Cmd

	m, frameCmd = m.frameHeightCmd()

	return m, tea.Batch(
		m.dashboard.Init(),
		func() tea.Msg {
			return msgs.TopbarContext{
				Phase:   "dashboard",
				File:    filepath.Base(msg.Project.File),
				Service: "",
			}
		},
		frameCmd,
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

// computeFrameHeight returns the current number of terminal lines consumed by
// the app chrome (topbar + bottom bar). Called whenever the help bar or status
// bar state changes.
func (m Model) computeFrameHeight() int {
	if m.statusActive {
		return chromeTopbar + chromeStatusbar
	}

	if m.helpExpanded && m.keymap != nil {
		fullHelp := m.keymap.FullHelp()
		maxCol := 0

		for _, col := range fullHelp {
			if len(col) > maxCol {
				maxCol = len(col)
			}
		}

		return chromeTopbar + maxCol
	}

	return chromeTopbar + chromeCompactHelp
}

// frameHeightCmd returns a command that delivers the current frame height
// to phase components so they can adjust their usable body area. It only
// emits a message when the value actually changes.
func (m Model) frameHeightCmd() (Model, tea.Cmd) {
	h := m.computeFrameHeight()
	if h == m.lastFrameHeight {
		return m, nil
	}

	m.lastFrameHeight = h

	return m, func() tea.Msg {
		return msgs.FrameHeight{Height: h}
	}
}

// View composes the top bar, active phase body, status bar, and help bar into a unified frame.
func (m Model) View() tea.View {
	if m.cache.lastGen > 0 && m.cache.gen == m.cache.lastGen {
		return m.cache.cachedView
	}

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

	m.cache.cachedView = v
	m.cache.lastGen = m.cache.gen

	return v
}
