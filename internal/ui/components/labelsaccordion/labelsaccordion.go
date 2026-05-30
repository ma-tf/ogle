// Package labelsaccordion renders the ogle.* container labels as a separate
// collapsible accordion section below the Service Details accordion in the
// Service Inspector. Collapsed by default; toggled open/closed by mouse click.
package labelsaccordion

import (
	"maps"
	"sort"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/accordion/value"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	zoneLabelsHeader   = "labels-header"
	listMinTermWidth   = 80
	listRatio          = 30
	pctDivisor         = 100
	keyWidthCapDivisor = 2
)

// Model is the labels accordion component state.
type Model struct {
	labels     map[string]string
	collapsed  bool
	hovered    bool
	w          int
	columnW    int
	values     []value.Model
	scrollGen  int
	lastLabels map[string]string
	lastWidth  int
	th         *theme.Theme
	zm         *zone.Manager
}

// New returns a Model. Collapsed by default.
func New(th *theme.Theme, w int, zm *zone.Manager) Model {
	return Model{
		labels:     nil,
		collapsed:  true,
		hovered:    false,
		w:          w,
		columnW:    max(w, listMinTermWidth) * listRatio / pctDivisor,
		values:     nil,
		scrollGen:  0,
		lastLabels: nil,
		lastWidth:  -1,
		th:         th,
		zm:         zm,
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

		return m.syncValues()

	case theme.Changed:
		m.th = msg.Theme

		return m.syncValues()

	case msgs.ContainerLabelsPolled:
		if msg.Err == nil {
			m.labels = msg.Labels
		} else {
			m.labels = nil
		}

		return m.syncValues()

	case msgs.ServiceSelected:
		m.labels = nil
		m.collapsed = true
		m.values = nil
		m.lastLabels = nil
		m.lastWidth = -1

		return m, nil

	case tea.MouseClickMsg:
		if m.zm != nil && m.zm.Get(zoneLabelsHeader).InBounds(msg) {
			wasCollapsed := m.collapsed
			m.collapsed = !m.collapsed

			if wasCollapsed && !m.collapsed {
				return m, func() tea.Msg {
					return value.StartMsg{Gen: m.scrollGen}
				}
			}
		}

		return m, nil

	case tea.MouseMotionMsg:
		m.hovered = m.zm != nil && m.zm.Get(zoneLabelsHeader).InBounds(msg)

		return m, nil
	}

	var cmds []tea.Cmd

	for i := range m.values {
		var cmd tea.Cmd

		m.values[i], cmd = m.values[i].Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the Labels accordion section. When collapsed or no labels
// exist, only the header is shown. When expanded with labels, each label
// is rendered as a two-column row: key (truncated if needed) and value
// (scrollable when overflowing).
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

	kw := m.keyWidth()
	vw := m.valueWidth()
	bg := m.th.AccordionBackground

	keys := sortedKeys(m.labels)

	labelParts := make([]string, 0, len(keys))
	valParts := make([]string, 0, len(keys))

	for i, k := range keys {
		keyStr := k + ": "

		if ansi.StringWidth(keyStr) > kw {
			keyStr = ansi.Truncate(keyStr, kw, "…")
		}

		labelParts = append(labelParts, keyStr)
		valParts = append(valParts, m.values[i].View().Content)
	}

	labelBlock := lipgloss.JoinVertical(lipgloss.Left, labelParts...)
	labelCol := lipgloss.NewStyle().
		Width(kw).
		Foreground(m.th.AccordionLabel).
		Background(bg).
		Render(labelBlock)

	valBlock := lipgloss.JoinVertical(lipgloss.Left, valParts...)
	valCol := lipgloss.NewStyle().
		Width(vw).
		Foreground(m.th.AccordionValue).
		Background(bg).
		Render(valBlock)

	content := lipgloss.JoinVertical(lipgloss.Top,
		headerStr,
		lipgloss.JoinHorizontal(lipgloss.Top, labelCol, valCol),
	)

	return tea.NewView(content)
}

func (m Model) keyWidth() int {
	if len(m.labels) == 0 {
		return 0
	}

	maxKeyLen := 0

	for k := range m.labels {
		keyLen := ansi.StringWidth(k + ": ")
		if keyLen > maxKeyLen {
			maxKeyLen = keyLen
		}
	}

	return min(maxKeyLen, m.columnW/keyWidthCapDivisor)
}

func (m Model) valueWidth() int {
	return m.columnW - m.keyWidth()
}

func (m Model) syncValues() (Model, tea.Cmd) {
	vw := m.valueWidth()
	if m.columnW == 0 || len(m.labels) == 0 || vw <= 0 {
		m.values = nil
		m.lastLabels = nil
		m.lastWidth = -1

		return m, nil
	}

	if maps.Equal(m.labels, m.lastLabels) && vw == m.lastWidth {
		return m, nil
	}

	m.scrollGen++

	keys := sortedKeys(m.labels)
	m.values = make([]value.Model, len(keys))

	for i, k := range keys {
		m.values[i] = value.New(m.labels[k], m.th.AccordionValue, m.th.AccordionBackground, vw)
	}

	m.lastLabels = maps.Clone(m.labels)
	m.lastWidth = vw

	if m.collapsed {
		return m, nil
	}

	return m, func() tea.Msg {
		return value.StartMsg{Gen: m.scrollGen}
	}
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}
