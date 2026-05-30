// Package dashboard implements the project dashboard flow. Dispatches 10+
// message types to sub-models (accordion, carousel, servicepanel, settings)
// and handles service actions, state polling, and file availability changes.
package dashboard

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/config"
	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	svcdocker "github.com/ma-tf/ogle/internal/services/docker"
	"github.com/ma-tf/ogle/internal/services/parser"
	"github.com/ma-tf/ogle/internal/ui/components/accordion"
	"github.com/ma-tf/ogle/internal/ui/components/carousel"
	"github.com/ma-tf/ogle/internal/ui/components/labelsaccordion"
	"github.com/ma-tf/ogle/internal/ui/components/servicepanel"
	"github.com/ma-tf/ogle/internal/ui/components/settings"
	"github.com/ma-tf/ogle/internal/ui/layout"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const accordionInitHeight = 8 // passed to accordion.New; unused by accordion.View()

// Model is the dashboard flow orchestrator.
type Model struct {
	ctx       context.Context
	parser    parser.Parser
	project   *domain.Project
	th        *theme.Theme
	zm        *zone.Manager
	configDir string
	docker    svcdocker.Docker

	accordion       accordion.Model
	labelsAccordion labelsaccordion.Model
	carousel        carousel.Model
	panel           servicepanel.Model
	settings        settings.Model
	showingSettings bool
	cfg             config.Config
	selectedName    string
	runtimeData     map[string]*domain.ServiceRuntimeData
	w, h            int
	frameHeight     int
}

// New returns a Model.
func New(
	ctx context.Context,
	project *domain.Project,
	th *theme.Theme,
	cfg config.Config,
	zm *zone.Manager,
	configDir string,
	w, h int,
	docker svcdocker.Docker,
	p parser.Parser,
) Model {
	selectedName := ""
	if len(project.Services) > 0 {
		selectedName = project.Services[0].Name
	}

	return Model{
		ctx:             ctx,
		parser:          p,
		project:         project,
		th:              th,
		zm:              zm,
		configDir:       configDir,
		docker:          docker,
		accordion:       accordion.New(project, w, accordionInitHeight, th, zm),
		labelsAccordion: labelsaccordion.New(th, w, zm),
		carousel:        carousel.New(project, w, h, th, zm),
		panel:           servicepanel.New(project, th, w, h, cfg.LogBufferCap),
		settings:        settings.New(th, cfg, w, h),
		showingSettings: false,
		cfg:             cfg,
		selectedName:    selectedName,
		runtimeData:     nil,
		w:               w,
		h:               h,
		frameHeight:     layout.FrameHeight,
	}
}

// Init fires sub-model initialisation and sends the keymap to the helpbar.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.carousel.Init(),
		m.labelsAccordion.Init(),
		m.panel.Init(),
		func() tea.Msg {
			return msgs.BindingsMsg{Keymap: appKeymap{}}
		},
	)
}

// Update handles dashboard-level messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var carouselCmd, panCmd, settingsCmd, accCmd, labelsAccordionCmd, selectedCmd, polledCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height

	case msgs.StatePollTick:
		m.panel, panCmd = m.panel.Update(msg)

		return m, tea.Batch(
			m.docker.Ps(m.ctx, m.project.File, m.project.Name),
			panCmd,
		)

	case tea.MouseClickMsg, tea.MouseMotionMsg, tea.MouseReleaseMsg, tea.MouseWheelMsg:
		if m.showingSettings {
			return m, nil
		}

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)

	case msgs.ServiceStop,
		msgs.ServiceStart,
		msgs.ServiceRestart,
		msgs.ServiceRebuild,
		msgs.ServiceActionCompleted:
		return m.handleServiceAction(msg)

	case msgs.FileAvailabilityChanged:
		return m.handleFileAvailabilityChanged(msg.Files)

	case msgs.SettingsApplied:
		return m.handleSettingsApplied(msg)

	case theme.Changed:
		m.th = msg.Theme

	case msgs.SettingsVisibilityChanged:
		m.showingSettings = msg.Visible

		return m, nil

	case msgs.ServiceSelected:
		m, selectedCmd = m.handleServiceSelected(msg)

	case msgs.ServicesPolled:
		m, polledCmd = m.handleServicesPolled(msg)

	case msgs.FrameHeight:
		m.frameHeight = msg.Height
	}

	m.accordion, accCmd = m.accordion.Update(msg)
	m.labelsAccordion, labelsAccordionCmd = m.labelsAccordion.Update(msg)
	m.carousel, carouselCmd = m.carousel.Update(msg)
	m.panel, panCmd = m.panel.Update(msg)

	if m.showingSettings {
		m.settings, settingsCmd = m.settings.Update(msg)
	}

	return m, tea.Batch(
		accCmd,
		labelsAccordionCmd,
		carouselCmd,
		panCmd,
		settingsCmd,
		selectedCmd,
		polledCmd,
	)
}

func (m Model) handleKeyPress(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if m.showingSettings {
		var settingsCmd tea.Cmd

		m.settings, settingsCmd = m.settings.Update(msg)

		return m, settingsCmd
	}

	switch {
	case key.Matches(msg, keyQuit):
		return m, tea.Quit

	case key.Matches(msg, keySettings):
		m.showingSettings = true

		return m, nil

	case key.Matches(msg, keyToggleWrap):
		return m, func() tea.Msg { return msgs.ToggleLogWrap{} }

	case key.Matches(msg, keyScrollUp), key.Matches(msg, keyScrollDown),
		key.Matches(msg, keyScrollLeft), key.Matches(msg, keyScrollRight):
		m.panel, _ = m.panel.Update(msg)

		return m, nil

	case key.Matches(msg, keyRestart):
		if m.selectedName == "" {
			return m, nil
		}

		return m, func() tea.Msg {
			return msgs.ServiceRestart{ServiceName: m.selectedName}
		}

	case key.Matches(msg, keyRebuild):
		if m.selectedName == "" {
			return m, nil
		}

		return m, func() tea.Msg {
			return msgs.ServiceRebuild{ServiceName: m.selectedName}
		}

	case key.Matches(msg, keyClearLog):
		if m.selectedName == "" {
			return m, nil
		}

		return m, func() tea.Msg {
			return msgs.ClearLogBuffer{ServiceName: m.selectedName}
		}
	}

	var cmd tea.Cmd

	m.carousel, cmd = m.carousel.Update(msg)

	return m, cmd
}

func (m Model) handleServiceAction(msg tea.Msg) (Model, tea.Cmd) {
	var carouselCmd, svcCmd, statusCmd tea.Cmd

	switch msg := msg.(type) {
	case msgs.ServiceStop:
		m.carousel, carouselCmd = m.carousel.Update(msg)
		statusCmd = func() tea.Msg {
			return msgs.DisplayStatus{Msg: msg.ServiceName + " stopping"}
		}
		svcCmd = m.docker.Stop(m.ctx, m.project.File, m.project.Name, msg.ServiceName)

	case msgs.ServiceStart:
		m.carousel, carouselCmd = m.carousel.Update(msg)
		statusCmd = func() tea.Msg {
			return msgs.DisplayStatus{Msg: msg.ServiceName + " starting"}
		}
		svcCmd = m.docker.Start(m.ctx, m.project.File, m.project.Name, msg.ServiceName)

	case msgs.ServiceRestart:
		m.carousel, carouselCmd = m.carousel.Update(msg)
		statusCmd = func() tea.Msg {
			return msgs.DisplayStatus{Msg: msg.ServiceName + " restarting"}
		}
		svcCmd = m.docker.Restart(m.ctx, m.project.File, m.project.Name, msg.ServiceName)

	case msgs.ServiceRebuild:
		m.carousel, carouselCmd = m.carousel.Update(msg)
		statusCmd = func() tea.Msg {
			return msgs.DisplayStatus{Msg: msg.ServiceName + " rebuilding"}
		}
		svcCmd = m.docker.Rebuild(m.ctx, m.project.File, m.project.Name, msg.ServiceName)

	case msgs.ServiceActionCompleted:
		m.carousel, carouselCmd = m.carousel.Update(msg)
		if msg.Err != nil {
			return m, tea.Batch(
				carouselCmd,
				func() tea.Msg { return msgs.DisplayError{Err: msg.Err.Error()} },
			)
		}

		return m, carouselCmd
	}

	return m, tea.Batch(carouselCmd, statusCmd, svcCmd)
}

func (m Model) handleFileAvailabilityChanged(files []string) (Model, tea.Cmd) {
	if !slices.Contains(files, m.project.File) {
		return m, func() tea.Msg {
			return msgs.FileRemoved{File: m.project.File}
		}
	}

	p, err := m.parser.Parse(m.project.File)
	if err != nil {
		return m, func() tea.Msg {
			return msgs.DisplayError{
				Err: fmt.Sprintf("re-parse failed: %v", err),
			}
		}
	}

	newDash := New(m.ctx, p, m.th, m.cfg, m.zm, m.configDir, m.w, m.h, m.docker, m.parser)

	return newDash, newDash.Init()
}

// handleServiceSelected updates the selected service name and returns
// TopbarContext, ReportWrapStatus, and container label inspection commands.
func (m Model) handleServiceSelected(msg msgs.ServiceSelected) (Model, tea.Cmd) {
	m.selectedName = msg.ServiceName

	return m, tea.Batch(
		func() tea.Msg {
			return msgs.TopbarContext{
				Phase:   "dashboard",
				File:    filepath.Base(m.project.File),
				Service: msg.ServiceName,
			}
		},
		func() tea.Msg {
			return msgs.ReportWrapStatus(msg)
		},
		m.inspectSelected(),
	)
}

// handleSettingsApplied updates theme and config from settings changes.
func (m Model) handleSettingsApplied(msg msgs.SettingsApplied) (Model, tea.Cmd) {
	if th, err := theme.Load(msg.Theme, m.configDir); err == nil {
		m.th = th
	}

	m.cfg.Theme = msg.Theme
	m.cfg.LogBufferCap = msg.LogBufferCap

	return m, nil
}

// handleServicesPolled updates runtime data and triggers label inspection
// when the data is available.
func (m Model) handleServicesPolled(msg msgs.ServicesPolled) (Model, tea.Cmd) {
	if msg.Err == nil {
		m.runtimeData = msg.Runtimes

		return m, m.inspectSelected()
	}

	return m, nil
}

// inspectSelected returns a Cmd that fetches container labels for the
// currently selected service if runtime data and a container ID are available.
func (m Model) inspectSelected() tea.Cmd {
	if m.runtimeData == nil {
		return nil
	}

	if rt, ok := m.runtimeData[m.selectedName]; ok && rt.ContainerID != "" {
		return m.docker.Inspect(m.ctx, rt.ContainerID)
	}

	return nil
}

// View renders the service list and inspector side by side. When settings is
// visible it renders as an overlay on top of the normal dashboard.
func (m Model) View() tea.View {
	listContent := m.carousel.View().Content
	listH := lipgloss.Height(listContent)
	listW := lipgloss.Width(listContent)

	usableH := m.h - m.frameHeight

	accView := m.accordion.View().Content
	accH := lipgloss.Height(accView)

	if listH+accH <= usableH && accH > 0 {
		accView = lipgloss.NewStyle().
			Width(listW).
			Height(accH).
			Background(m.th.CarouselBackground).
			Render(accView)
		listContent = lipgloss.JoinVertical(lipgloss.Top, listContent, accView)
	}

	labelsContent := m.labelsAccordion.View().Content
	if labelsContent != "" {
		listContent = lipgloss.JoinVertical(lipgloss.Top, listContent, labelsContent)
	}

	listH = lipgloss.Height(listContent)
	if listH < usableH {
		filler := lipgloss.NewStyle().
			Width(listW).
			Height(usableH - listH).
			Background(m.th.CarouselBackground).
			Render("")
		listContent = lipgloss.JoinVertical(lipgloss.Top, listContent, filler)
	}

	panContent := m.panel.View().Content

	body := lipgloss.JoinHorizontal(lipgloss.Top, listContent, panContent)

	if m.showingSettings {
		overContent := m.settings.View().Content
		overW := lipgloss.Width(overContent)
		overH := lipgloss.Height(overContent)
		overX := max((m.w-overW)/2, 0) //nolint:mnd // halving to centre overlay
		overY := max((m.h-overH)/2, 0) //nolint:mnd // halving to centre overlay

		return tea.NewView(lipgloss.NewCompositor(
			lipgloss.NewLayer(body).X(0).Y(0).Z(0),
			lipgloss.NewLayer(overContent).X(overX).Y(overY).Z(1),
		).Render())
	}

	return tea.NewView(body)
}
