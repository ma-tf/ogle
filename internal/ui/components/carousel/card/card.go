// Package card implements a single service card as a tea.Model.
package card

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// FocusMsg tells a card it is now focused.
type FocusMsg struct{ ServiceName string }

// BlurMsg tells a card it is no longer focused.
type BlurMsg struct{ ServiceName string }

// HoverMsg tells a card the mouse is hovering over it.
type HoverMsg struct{ ServiceName string }

// UnhoverMsg tells a card the mouse is no longer hovering over it.
type UnhoverMsg struct{ ServiceName string }

// ScrollTick advances the scrolling text window for a truncated service name.
type ScrollTick struct {
	gen int
}

const (
	cols               = 3
	listRatio          = 30
	listMinTermWidth   = 80
	pctDivisor         = 100
	maxCardH           = 12
	terminalCellAspect = 2
	// BorderW is the total width in cells of both left and right (or top and
	// bottom) border edges.
	BorderW            = 2
	scrollStepInterval = 300
	scrollIdleInterval = 2500
)

// Model is a tea.Model representing a single service card.
type Model struct {
	def            domain.ServiceDef
	w, h           int
	focused        bool
	hovered        bool
	th             *theme.Theme
	runtime        *domain.ServiceRuntimeData
	inFlight       bool
	scrollOffset   int
	scrollDir      int
	nextScrollTime time.Time
	focusGen       int
}

// New returns a Model for the given service definition and terminal dimensions.
func New(def domain.ServiceDef, w, h int, th *theme.Theme) Model {
	return Model{
		def:            def,
		w:              w,
		h:              h,
		focused:        false,
		hovered:        false,
		th:             th,
		runtime:        nil,
		inFlight:       false,
		scrollDir:      1,
		scrollOffset:   0,
		nextScrollTime: time.Time{},
		focusGen:       0,
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update satisfies tea.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)

	case FocusMsg:
		return m.handleFocusMsg(msg)

	case BlurMsg:
		if m.def.Name != msg.ServiceName {
			break
		}

		m.focused = false
		m.scrollOffset = 0
		m.scrollDir = 1

	case HoverMsg:
		if m.def.Name != msg.ServiceName {
			break
		}

		m.hovered = true

	case UnhoverMsg:
		if m.def.Name != msg.ServiceName {
			break
		}

		m.hovered = false

	case ScrollTick:
		return m.handleScrollTick(msg)

	case msgs.ServicesPolled:
		m = m.handleServicesPolledMsg(msg)

	case msgs.ServiceStop, msgs.ServiceStart, msgs.ServiceRestart, msgs.ServiceRebuild:
		m = m.setInFlightIfMatch(serviceNameFromMsg(msg))

	case msgs.ServiceActionCompleted:
		m = m.handleActionCompleted(msg)

	case theme.Changed:
		m.th = msg.Theme
	}

	return m, nil
}

func (m Model) handleWindowSizeMsg(msg tea.WindowSizeMsg) (Model, tea.Cmd) {
	m.w = msg.Width
	m.h = msg.Height
	m.scrollOffset = 0
	m.scrollDir = 1

	if m.focused && m.needsScroll() {
		m.focusGen++
		m.nextScrollTime = time.Now().Add(scrollIdleInterval * time.Millisecond)

		return m, tickScroll(m.nextScrollTime, m.focusGen)
	}

	return m, nil
}

func (m Model) handleFocusMsg(msg FocusMsg) (Model, tea.Cmd) {
	if m.def.Name != msg.ServiceName {
		return m, nil
	}

	m.focused = true
	m.scrollOffset = 0
	m.scrollDir = 1

	if m.needsScroll() {
		m.focusGen++
		m.nextScrollTime = time.Now().Add(scrollIdleInterval * time.Millisecond)

		return m, tickScroll(m.nextScrollTime, m.focusGen)
	}

	return m, nil
}

func (m Model) handleServicesPolledMsg(msg msgs.ServicesPolled) Model {
	if msg.Err == nil && m.def.Name != "" {
		m.runtime = msg.Runtimes[m.def.Name]
	}

	return m
}

func (m Model) handleScrollTick(msg ScrollTick) (Model, tea.Cmd) {
	if !m.focused || !m.needsScroll() || msg.gen != m.focusGen {
		return m, nil
	}

	if time.Now().Before(m.nextScrollTime) {
		return m, tickScroll(m.nextScrollTime, m.focusGen)
	}

	maxOff := m.maxScrollOffset()

	m.scrollOffset += m.scrollDir

	switch {
	case m.scrollOffset >= maxOff:
		m.scrollOffset = maxOff
		m.scrollDir = -1
		m.nextScrollTime = time.Now().Add(scrollIdleInterval * time.Millisecond)

	case m.scrollOffset <= 0:
		m.scrollOffset = 0
		m.scrollDir = 1
		m.nextScrollTime = time.Now().Add(scrollIdleInterval * time.Millisecond)

	default:
		m.nextScrollTime = time.Now().Add(scrollStepInterval * time.Millisecond)
	}

	return m, tickScroll(m.nextScrollTime, m.focusGen)
}

func (m Model) setInFlightIfMatch(serviceName string) Model {
	if m.def.Name == serviceName {
		m.inFlight = true
	}

	return m
}

func (m Model) handleActionCompleted(msg msgs.ServiceActionCompleted) Model {
	if m.def.Name != msg.ServiceName {
		return m
	}

	m.inFlight = false

	if msg.Err != nil {
		return m
	}

	opt := domain.ServiceStateRunning
	if msg.Action == domain.ServiceActionStop {
		opt = domain.ServiceStateExited
	}

	if m.runtime == nil {
		m.runtime = &domain.ServiceRuntimeData{
			State:       opt,
			ContainerID: "",
			Status:      "",
			CreatedAt:   time.Time{},
		}
	} else {
		rt := *m.runtime
		rt.State = opt
		m.runtime = &rt
	}

	return m
}

func serviceNameFromMsg(msg tea.Msg) string {
	switch m := msg.(type) {
	case msgs.ServiceStop:
		return m.ServiceName
	case msgs.ServiceStart:
		return m.ServiceName
	case msgs.ServiceRestart:
		return m.ServiceName
	case msgs.ServiceRebuild:
		return m.ServiceName
	default:
		return ""
	}
}

func tickScroll(t time.Time, gen int) tea.Cmd {
	return tea.Tick(max(0, time.Until(t)), func(_ time.Time) tea.Msg {
		return ScrollTick{gen: gen}
	})
}

// View satisfies tea.Model.
func (m Model) View() tea.View {
	cardW, cardH := m.cardWidth(), m.cardHeight()

	if cardW <= 0 || cardH <= 0 {
		return tea.NewView("")
	}

	innerW := cardW - BorderW
	innerH := cardH - BorderW
	name := m.def.Name

	var shown string

	switch {
	case len(name) <= innerW:
		shown = name

	case m.focused:
		shown = name[m.scrollOffset : m.scrollOffset+innerW]

	default:
		shown = name[:innerW-1] + "…"
	}

	bg := m.th.CarouselBackground
	switch {
	case m.focused:
		bg = m.th.SelectedBackground
	case m.hovered:
		bg = m.th.CardHoverBackground
	}

	content := lipgloss.NewStyle().
		Width(innerW).
		Align(lipgloss.Center).
		Foreground(m.th.Text).
		Background(bg).
		Render(shown)
	padded := lipgloss.PlaceVertical(innerH, lipgloss.Center, content)

	borderFg := m.th.StateMuted
	switch {
	case m.inFlight:
		borderFg = m.th.StateTransient
	case m.runtime != nil:
		borderFg = m.th.ColourForState(m.runtime.State)
	}

	return tea.NewView(lipgloss.NewStyle().
		Width(cardW).
		Height(cardH).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderFg).
		BorderBackground(bg).
		Background(bg).
		Render(padded))
}

func (m Model) cardWidth() int {
	carouselW := max(m.w, listMinTermWidth) * listRatio / pctDivisor

	return carouselW / cols
}

func (m Model) cardHeight() int {
	h := min(m.cardWidth()/terminalCellAspect, maxCardH)
	if h%2 == 0 {
		h--
	}

	return max(h, 1)
}

func (m Model) needsScroll() bool {
	return len(m.def.Name) > m.cardWidth()-BorderW
}

func (m Model) maxScrollOffset() int {
	return len(m.def.Name) - (m.cardWidth() - BorderW)
}
