package helpbar

import (
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// PinnedKeyMap extends help.KeyMap with pinned bindings that are always
// visible in compact mode, rendered right-aligned and never truncated.
type PinnedKeyMap interface {
	ShortHelp() []key.Binding
	FullHelp() [][]key.Binding
	PinnedHelp() []key.Binding
}

// Model is a value-type sub-model that renders a help bar.
type Model struct {
	keymap  help.KeyMap
	th      *theme.Theme
	width   int
	showAll bool
}

// New returns a Model styled with th.
func New(th *theme.Theme) Model {
	return Model{th: th, keymap: nil, width: 0, showAll: false}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// ToggleMsg toggles the help bar between compact and full view.
type ToggleMsg struct{}

// Update satisfies tea.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case msgs.BindingsMsg:
		m.keymap = msg.Keymap
	case theme.Changed:
		m.th = msg.Theme
	case ToggleMsg:
		m.showAll = !m.showAll
	}

	return m, nil
}

// ShowAll reports whether the help is in full mode.
func (m Model) ShowAll() bool {
	return m.showAll
}

// View renders the help bar with the current keymap.
func (m Model) View() tea.View {
	if m.keymap == nil {
		return tea.NewView("")
	}

	var content string
	if m.showAll {
		content = m.renderFull()
	} else {
		content = m.renderCompact()
	}

	rendered := lipgloss.NewStyle().
		Background(m.th.HelpBackground).
		Width(m.width).
		Render(content)

	return tea.NewView(rendered)
}

func (m Model) renderCompact() string {
	var pinned []key.Binding

	if pk, ok := m.keymap.(PinnedKeyMap); ok {
		pinned = pk.PinnedHelp()
	}

	pinnedSet := make(map[string]bool, len(pinned))

	for _, p := range pinned {
		pinnedSet[p.Help().Key] = true
	}

	shortHelp := m.keymap.ShortHelp()

	var truncatable []key.Binding

	for _, b := range shortHelp {
		if !pinnedSet[b.Help().Key] {
			truncatable = append(truncatable, b)
		}
	}

	pinnedText := renderBindings(pinned, m.th)
	pinnedWidth := lipgloss.Width(pinnedText)

	gap := 2
	availWidth := m.width - pinnedWidth - gap

	truncText := renderTruncatable(truncatable, m.th, availWidth)
	truncWidth := lipgloss.Width(truncText)

	fill := max(0, m.width-truncWidth-pinnedWidth)

	return truncText + strings.Repeat(" ", fill) + pinnedText
}

func (m Model) renderFull() string {
	fullHelp := m.keymap.FullHelp()

	if len(fullHelp) == 0 {
		return ""
	}

	maxKeyWidths := make([]int, len(fullHelp))

	for ci, col := range fullHelp {
		for _, b := range col {
			if kw := lipgloss.Width(b.Help().Key); kw > maxKeyWidths[ci] {
				maxKeyWidths[ci] = kw
			}
		}
	}

	var columns []string

	for ci, col := range fullHelp {
		var lines []string

		for _, b := range col {
			k := b.Help().Key + strings.Repeat(" ", maxKeyWidths[ci]-lipgloss.Width(b.Help().Key))
			line := m.th.HelpKey.Render(k) + m.th.HelpDesc.Render(" "+b.Help().Desc)

			lines = append(lines, line)
		}

		columns = append(columns, lipgloss.JoinVertical(lipgloss.Top, lines...))
	}

	sep := m.th.HelpSep.Render("  ")

	return lipgloss.JoinHorizontal(lipgloss.Top, columns[0], sep,
		strings.Join(columns[1:], sep))
}

func renderBindings(bindings []key.Binding, th *theme.Theme) string {
	if len(bindings) == 0 {
		return ""
	}

	var parts []string

	for idx, b := range bindings {
		parts = append(parts, th.HelpKey.Render(b.Help().Key)+th.HelpDesc.Render(" "+b.Help().Desc))

		if idx < len(bindings)-1 {
			parts = append(parts, th.HelpSep.Render("  •  "))
		}
	}

	return strings.Join(parts, "")
}

func renderTruncatable(bindings []key.Binding, th *theme.Theme, maxWidth int) string {
	if maxWidth <= 0 || len(bindings) == 0 {
		return ""
	}

	sep := th.HelpSep.Render("  •  ")
	sepWidth := lipgloss.Width(sep)
	ellipsis := th.HelpSep.Render("…")
	ellipsisWidth := lipgloss.Width(ellipsis)

	var parts []string

	for _, b := range bindings {
		rendered := th.HelpKey.Render(b.Help().Key) + th.HelpDesc.Render(" "+b.Help().Desc)
		itemWidth := lipgloss.Width(rendered)

		var totalWidth int
		if len(parts) > 0 {
			totalWidth = itemWidth + sepWidth + lipgloss.Width(strings.Join(parts, ""))
		} else {
			totalWidth = itemWidth
		}

		if totalWidth > maxWidth && len(parts) > 0 {
			totalAfterEllipsis := lipgloss.Width(strings.Join(parts, "")) + ellipsisWidth

			if totalAfterEllipsis <= maxWidth {
				parts = append(parts, ellipsis)
			}

			break
		}

		if totalWidth > maxWidth {
			break
		}

		if len(parts) > 0 {
			parts = append(parts, sep)
		}

		parts = append(parts, rendered)
	}

	return strings.Join(parts, "")
}
