// Package labelsaccordion renders the ogle.* container labels as a separate
// collapsible accordion section below the Service Details accordion in the
// Service Inspector. Collapsed by default; toggled open/closed by mouse click.
package labelsaccordion

import (
	"sort"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	zoneLabelsHeader = "labels-header"
	listMinTermWidth = 80
	listRatio        = 30
	pctDivisor       = 100
)

// Model is the labels accordion component state.
type Model struct {
	labels    map[string]string
	collapsed bool
	hovered   bool
	w         int
	columnW   int
	th        *theme.Theme
	zm        *zone.Manager
}

// New returns a Model. Collapsed by default.
func New(th *theme.Theme, w int, zm *zone.Manager) Model {
	return Model{
		labels:    nil,
		collapsed: true,
		hovered:   false,
		w:         w,
		columnW:   max(w, listMinTermWidth) * listRatio / pctDivisor,
		th:        th,
		zm:        zm,
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update handles container labels, service selection, mouse events, window
// resize, and theme changes.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.columnW = max(m.w, listMinTermWidth) * listRatio / pctDivisor

		return m, nil

	case theme.Changed:
		m.th = msg.Theme

		return m, nil

	case msgs.ContainerLabelsPolled:
		if msg.Err == nil {
			m.labels = msg.Labels
		} else {
			m.labels = nil
		}

		return m, nil

	case msgs.ServiceSelected:
		m.labels = nil
		m.collapsed = true

		return m, nil

	case tea.MouseClickMsg:
		if m.zm != nil && m.zm.Get(zoneLabelsHeader).InBounds(msg) {
			m.collapsed = !m.collapsed
		}

		return m, nil

	case tea.MouseMotionMsg:
		m.hovered = m.zm != nil && m.zm.Get(zoneLabelsHeader).InBounds(msg)

		return m, nil
	}

	return m, nil
}

// View renders the Labels accordion section. When collapsed or no labels
// exist, only the header is shown. When expanded with labels, each label
// is rendered as "key: value" on its own line.
func (m Model) View() tea.View {
	if m.w == 0 {
		return tea.NewView("")
	}

	indicator := "▶"
	if !m.collapsed {
		indicator = "▼"
	}

	headerBg := m.th.AccordionHeaderBackground
	if m.hovered {
		headerBg = m.th.AccordionHeaderHoverBackground
	}

	headerStr := lipgloss.NewStyle().
		Width(m.columnW).
		Foreground(m.th.AccordionLabel).
		Background(headerBg).
		Render(" " + indicator + " Labels")
	if m.zm != nil {
		headerStr = m.zm.Mark(zoneLabelsHeader, headerStr)
	}

	if m.collapsed || len(m.labels) == 0 {
		return tea.NewView(headerStr)
	}

	keys := make([]string, 0, len(m.labels))
	for k := range m.labels {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	rows := make([]string, 0, len(keys))
	for _, k := range keys {
		row := lipgloss.NewStyle().
			Width(m.columnW).
			Foreground(m.th.AccordionLabel).
			Background(m.th.AccordionBackground).
			Render(k + ": " + m.labels[k])
		rows = append(rows, row)
	}

	content := lipgloss.JoinVertical(lipgloss.Top, rows...)
	content = lipgloss.NewStyle().
		Width(m.columnW).
		Background(m.th.AccordionBackground).
		Render(content)

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Top, headerStr, content))
}
