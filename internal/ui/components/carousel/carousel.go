package carousel

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/paginator"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/carousel/card"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

type viewCache struct {
	gen        uint64
	lastGen    uint64
	cachedView tea.View
}

const (
	rows                 = 2
	cols                 = 3
	pageSize             = rows * cols
	listRatio            = 30
	listMinTermWidth     = 80
	pctDivisor           = 100
	maxCardH             = 12
	terminalCellAspect   = 2
	doubleClickThreshold = 350 * time.Millisecond
	zoneDotFmt           = "carousel-dot-%d"
	zoneCardFmt          = "carousel-card-%d"
)

// Model is the carousel component state.
type Model struct {
	all           []domain.ServiceDef
	cards         []card.Model
	w, h          int
	focus         int
	hovered       int
	hoveredDot    int
	paginator     paginator.Model
	th            *theme.Theme
	zm            *zone.Manager
	runtimeData   map[string]*domain.ServiceRuntimeData
	lastClickTime time.Time
	lastClickIdx  int
	cache         viewCache
}

// New returns a Model for the given project.
func New(project *domain.Project, w, h int, th *theme.Theme, zm *zone.Manager) Model {
	p := paginator.New(paginator.WithPerPage(pageSize))
	p.Type = paginator.Dots
	p.SetTotalPages(len(project.Services))
	p.KeyMap = paginator.KeyMap{
		PrevPage: KeyPgUp,
		NextPage: KeyPgDown,
	}
	p.ActiveDot = lipgloss.NewStyle().Foreground(th.CarouselFocused).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(th.CarouselBlurred).Render("○")

	_, end := p.GetSliceBounds(len(project.Services))
	n := end - p.Page*p.PerPage

	cards := make([]card.Model, n)
	for i := range n {
		cards[i] = card.New(project.Services[p.Page*p.PerPage+i], w, h, th)
	}

	dotCount := 0
	if p.TotalPages > 1 {
		dotCount = p.TotalPages
	}

	focus := 0
	if n > 0 {
		focus = dotCount
	}

	return Model{
		all:           project.Services,
		cards:         cards,
		w:             w,
		h:             h,
		focus:         focus,
		hovered:       -1,
		hoveredDot:    -1,
		paginator:     p,
		th:            th,
		zm:            zm,
		runtimeData:   nil,
		lastClickTime: time.Time{},
		lastClickIdx:  -1,
		cache: viewCache{
			gen:        1,
			lastGen:    0,
			cachedView: tea.View{}, //nolint:exhaustruct // zero value is sentinel for "not yet cached"
		},
	}
}

func (m Model) dotCount() int {
	if m.paginator.TotalPages > 1 {
		return m.paginator.TotalPages
	}

	return 0
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	if len(m.cards) > 0 {
		name := m.cardServiceName(0)

		return tea.Batch(
			func() tea.Msg { return card.FocusMsg{ServiceName: name} },
			func() tea.Msg { return msgs.ServiceSelected{ServiceName: name} },
		)
	}

	return nil
}

// Update handles key presses and window resize.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height
		m.cache.gen++

	case theme.Changed:
		m.th = msg.Theme
		m.paginator.ActiveDot = lipgloss.NewStyle().Foreground(m.th.CarouselFocused).Render("•")
		m.paginator.InactiveDot = lipgloss.NewStyle().Foreground(m.th.CarouselBlurred).Render("○")
		m.cache.gen++

	case msgs.ServicesPolled:
		if msg.Err == nil {
			m.runtimeData = msg.Runtimes
			m.cache.gen++
		}

	case tea.MouseClickMsg:
		return m.handleMouseClick(msg)

	case tea.MouseMotionMsg:
		return m.handleMouseMotion(msg)
	}

	for i := range m.cards {
		updated, cmd := m.cards[i].Update(msg)
		m.cards[i] = updated

		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleKeyPress(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if key.Matches(msg, KeyTab) {
		return m.handleTab()
	}

	if key.Matches(msg, KeyEnter) {
		return m.handleEnter()
	}

	prevPage := m.paginator.Page

	var pageCmd tea.Cmd

	m.paginator, pageCmd = m.paginator.Update(msg)

	if m.paginator.Page != prevPage {
		return m.rebuildCards()
	}

	return m, pageCmd
}

func (m Model) handleTab() (Model, tea.Cmd) {
	m.cache.gen++
	prevFocus := m.focus
	d := m.dotCount()
	total := m.totalSlots()
	m.focus = (m.focus + 1) % total

	for {
		onActiveDot := m.focus < d && m.focus == m.paginator.Page
		onEmptyCard := m.focus >= d && m.focus < d+pageSize && !m.slotHasCard(m.focus)

		if !onActiveDot && !onEmptyCard {
			break
		}

		m.focus = (m.focus + 1) % total
	}

	var cmds []tea.Cmd

	if prevFocus >= d && prevFocus < d+pageSize && m.slotHasCard(prevFocus) {
		idx := prevFocus - d

		cmds = append(cmds, func() tea.Msg {
			return card.BlurMsg{ServiceName: m.cardServiceName(idx)}
		})
	}

	if m.focus >= d && m.focus < d+pageSize && m.slotHasCard(m.focus) {
		idx := m.focus - d

		cmds = append(cmds,
			func() tea.Msg {
				return card.FocusMsg{ServiceName: m.cardServiceName(idx)}
			},
			func() tea.Msg {
				return msgs.ServiceSelected{ServiceName: m.cardServiceName(idx)}
			},
		)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleEnter() (Model, tea.Cmd) {
	d := m.dotCount()

	if m.focus < d {
		m.paginator.Page = m.focus

		return m.rebuildCards()
	}

	if m.focus >= d && m.focus < d+pageSize && !m.slotHasCard(m.focus) {
		return m, nil
	}

	if m.focus >= d && m.focus < d+pageSize {
		idx := m.focus - d
		name := m.cardServiceName(idx)

		return m, m.toggleServiceCmd(name)
	}

	return m, nil
}

func (m Model) handleMouseClick(msg tea.MouseClickMsg) (Model, tea.Cmd) {
	for i := range m.paginator.TotalPages {
		if m.zm.Get(fmt.Sprintf(zoneDotFmt, i)).InBounds(msg) {
			if m.paginator.Page == i {
				return m, nil
			}

			m.paginator.Page = i

			return m.rebuildCards()
		}
	}

	for i := range m.cards {
		if !m.zm.Get(fmt.Sprintf(zoneCardFmt, i)).InBounds(msg) {
			continue
		}

		return m.handleCardClick(i)
	}

	return m, nil
}

func (m Model) handleCardClick(i int) (Model, tea.Cmd) {
	d := m.dotCount()
	newFocus := i + d

	if m.focus == newFocus {
		if i == m.lastClickIdx && time.Since(m.lastClickTime) < doubleClickThreshold {
			return m, m.toggleServiceCmd(m.cardServiceName(i))
		}

		m.lastClickTime = time.Now()
		m.lastClickIdx = i

		return m, nil
	}

	var cmds []tea.Cmd

	if m.focus >= d && m.focus < d+pageSize {
		idx := m.focus - d

		cmds = append(cmds, func() tea.Msg {
			return card.BlurMsg{ServiceName: m.cardServiceName(idx)}
		})
	}

	m.cache.gen++
	m.focus = newFocus
	m.lastClickTime = time.Now()
	m.lastClickIdx = i

	cmds = append(cmds,
		func() tea.Msg {
			return card.FocusMsg{ServiceName: m.cardServiceName(i)}
		},
		func() tea.Msg {
			return msgs.ServiceSelected{ServiceName: m.cardServiceName(i)}
		},
	)

	return m, tea.Batch(cmds...)
}

func (m Model) handleMouseMotion(msg tea.MouseMotionMsg) (Model, tea.Cmd) {
	d := m.dotCount()
	hit := -1

	for i := range m.paginator.TotalPages {
		if m.zm.Get(fmt.Sprintf(zoneDotFmt, i)).InBounds(msg) {
			if m.hoveredDot == i {
				return m, nil
			}

			m.cache.gen++
			m.hoveredDot = i

			var unhoverCmd tea.Cmd

			m, unhoverCmd = m.unhoverCard()
			m.hovered = -1

			return m, unhoverCmd
		}
	}

	for i := range m.cards {
		if m.zm.Get(fmt.Sprintf(zoneCardFmt, i)).InBounds(msg) {
			hit = i + d

			break
		}
	}

	if m.hoveredDot >= 0 {
		m.hoveredDot = -1
	}

	if hit == m.hovered {
		return m, nil
	}

	var cmd tea.Cmd

	m.cache.gen++
	m, cmd = m.unhoverCard()
	m.hovered = hit

	if m.hovered >= d && m.hovered < d+pageSize {
		idx := m.hovered - d
		cmds := []tea.Cmd{cmd, func() tea.Msg {
			return card.HoverMsg{ServiceName: m.cardServiceName(idx)}
		}}

		return m, tea.Batch(cmds...)
	}

	return m, cmd
}

func (m Model) unhoverCard() (Model, tea.Cmd) {
	d := m.dotCount()
	if m.hovered >= d && m.hovered < d+pageSize {
		idx := m.hovered - d

		return m, func() tea.Msg {
			return card.UnhoverMsg{ServiceName: m.cardServiceName(idx)}
		}
	}

	return m, nil
}

// View renders the carousel with card grid, and nav bar below.
func (m Model) View() tea.View {
	if m.cache.lastGen > 0 && m.cache.gen == m.cache.lastGen {
		return m.cache.cachedView
	}

	carouselW := max(m.w, listMinTermWidth) * listRatio / pctDivisor
	cardW := carouselW / cols
	cardH := min(cardW/terminalCellAspect, maxCardH)

	if cardH%2 == 0 {
		cardH--
	}

	cardH = max(cardH, 1)

	rowStrs := make([]string, rows)

	for row := range rows {
		cells := make([]string, cols)

		for col := range cols {
			idx := row*cols + col

			if idx < len(m.cards) {
				cells[col] = m.zm.Mark(
					fmt.Sprintf(zoneCardFmt, idx),
					m.cards[idx].View().Content,
				)
			} else {
				innerW := cardW - card.BorderW
				innerH := cardH - card.BorderW
				padded := lipgloss.NewStyle().
					Width(innerW).
					Height(innerH).
					Background(m.th.CarouselBackground).
					Align(lipgloss.Center).
					AlignVertical(lipgloss.Center).
					Render("-")
				cells[col] = lipgloss.NewStyle().
					Width(cardW).
					Height(cardH).
					Border(lipgloss.RoundedBorder()).
					BorderForeground(m.th.CarouselEmpty).
					BorderBackground(m.th.CarouselBackground).
					Background(m.th.CarouselBackground).
					Render(padded)
			}
		}

		rowStrs[row] = lipgloss.JoinHorizontal(lipgloss.Top, cells...)
	}

	grid := lipgloss.JoinVertical(lipgloss.Left, rowStrs...)

	grid = lipgloss.NewStyle().
		Width(carouselW).
		Background(m.th.CarouselBackground).
		Render(grid)

	navBar := m.renderNavBar(carouselW)

	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, grid, navBar))
	m.cache.cachedView = v
	m.cache.lastGen = m.cache.gen

	return v
}

func (m Model) renderNavBar(carouselW int) string {
	if m.paginator.TotalPages <= 1 {
		return ""
	}

	focusedFg := m.th.CarouselFocused
	unfocusedFg := m.th.CarouselBlurred
	hoverFg := m.th.CarouselHover
	navBg := m.th.CarouselNavBackground

	totalPages := m.paginator.TotalPages
	dots := make([]string, totalPages)

	for i := range totalPages {
		dotChar := "○"
		dotColour := unfocusedFg

		switch {
		case m.paginator.Page == i:
			dotChar = "•"
			dotColour = focusedFg
		case m.focus == i:
			dotColour = focusedFg
		case m.hoveredDot == i:
			dotColour = hoverFg
		}

		dots[i] = m.zm.Mark(
			fmt.Sprintf(zoneDotFmt, i),
			lipgloss.NewStyle().
				Foreground(dotColour).
				Background(m.th.CarouselBackground).
				Render(dotChar),
		)
	}

	return lipgloss.NewStyle().
		Width(carouselW).
		Align(lipgloss.Center).
		Background(navBg).
		Render(strings.Join(dots, ""))
}

func (m Model) rebuildCards() (Model, tea.Cmd) {
	m.cache.gen++
	start, end := m.paginator.GetSliceBounds(len(m.all))
	n := end - start
	m.cards = make([]card.Model, n)

	for i := range n {
		m.cards[i] = card.New(m.all[start+i], m.w, m.h, m.th)
	}

	if n == 0 {
		m.focus = 0
		m.hovered = -1
		m.hoveredDot = -1
		m.lastClickTime = time.Time{}
		m.lastClickIdx = -1

		return m, nil
	}

	m.focus = m.dotCount()
	m.hovered = -1
	m.hoveredDot = -1
	m.lastClickTime = time.Time{}
	m.lastClickIdx = -1

	if len(m.cards) > 0 {
		return m, tea.Batch(func() tea.Msg {
			return card.FocusMsg{ServiceName: m.all[start].Name}
		}, func() tea.Msg {
			return msgs.ServiceSelected{ServiceName: m.all[start].Name}
		})
	}

	return m, nil
}

func (m Model) slotHasCard(slot int) bool {
	d := m.dotCount()

	cardIdx := slot - d
	if cardIdx < 0 || cardIdx >= pageSize {
		return false
	}

	return cardIdx < len(m.cards)
}

func (m Model) cardServiceName(idx int) string {
	start, _ := m.paginator.GetSliceBounds(len(m.all))

	return m.all[start+idx].Name
}

func (m Model) toggleServiceCmd(name string) tea.Cmd {
	rt := m.runtimeData[name]
	if rt != nil && rt.State == domain.ServiceStateRunning {
		return func() tea.Msg { return msgs.ServiceStop{ServiceName: name} }
	}

	return func() tea.Msg { return msgs.ServiceStart{ServiceName: name} }
}

func (m Model) totalSlots() int {
	return m.dotCount() + pageSize
}
