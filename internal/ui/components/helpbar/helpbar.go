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

	if content == "" {
		return tea.NewView("")
	}

	bg := m.th.HelpBackground

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w < m.width {
			pad := m.width - w
			lines[i] = line + lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", pad))
		}
	}

	return tea.NewView(strings.Join(lines, "\n"))
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

	bg := m.th.HelpBackground
	keyStyle := m.th.HelpKey.Background(bg)
	descStyle := m.th.HelpDesc.Background(bg)
	sepStyle := m.th.HelpSep.Background(bg)

	pinnedText := renderBindings(pinned, keyStyle, descStyle, sepStyle)
	pinnedWidth := lipgloss.Width(pinnedText)

	gap := 2
	availWidth := m.width - pinnedWidth - gap

	truncText := renderTruncatable(truncatable, keyStyle, descStyle, sepStyle, availWidth)
	truncWidth := lipgloss.Width(truncText)

	fill := max(0, m.width-truncWidth-pinnedWidth)

	if fill > 0 {
		fillStyle := lipgloss.NewStyle().Background(bg)
		truncText += fillStyle.Render(strings.Repeat(" ", fill))
	}

	return truncText + pinnedText
}

func (m Model) renderFull() string {
	fullHelp := m.keymap.FullHelp()

	if len(fullHelp) == 0 {
		return ""
	}

	bg := m.th.HelpBackground
	keyStyle := m.th.HelpKey.Background(bg)
	descStyle := m.th.HelpDesc.Background(bg)
	sepStyle := m.th.HelpSep.Background(bg)
	padStyle := lipgloss.NewStyle().Background(bg)

	cells, colWidths, maxRows := buildHelpCells(fullHelp, keyStyle, descStyle)

	if len(fullHelp) == 1 {
		for ri, cell := range cells[0] {
			if w := lipgloss.Width(cell); w < colWidths[0] {
				cells[0][ri] = cell + padStyle.Render(strings.Repeat(" ", colWidths[0]-w))
			}
		}

		return strings.Join(cells[0], "\n")
	}

	sep := sepStyle.Render("  ")

	return assembleHelpRows(cells, colWidths, maxRows, padStyle, sep)
}

func buildHelpCells(
	fullHelp [][]key.Binding,
	keyStyle, descStyle lipgloss.Style,
) ([][]string, []int, int) {
	numCols := len(fullHelp)
	maxKeyWidths := make([]int, numCols)
	colWidths := make([]int, numCols)
	cells := make([][]string, numCols)
	maxRows := 0

	for ci, col := range fullHelp {
		for _, b := range col {
			if kw := lipgloss.Width(b.Help().Key); kw > maxKeyWidths[ci] {
				maxKeyWidths[ci] = kw
			}
		}
	}

	for ci, col := range fullHelp {
		cells[ci] = make([]string, len(col))

		for ri, b := range col {
			k := b.Help().Key + strings.Repeat(" ", maxKeyWidths[ci]-lipgloss.Width(b.Help().Key))
			cell := keyStyle.Render(k) + descStyle.Render(" "+b.Help().Desc)
			cells[ci][ri] = cell

			if w := lipgloss.Width(cell); w > colWidths[ci] {
				colWidths[ci] = w
			}
		}

		if len(col) > maxRows {
			maxRows = len(col)
		}
	}

	return cells, colWidths, maxRows
}

func assembleHelpRows(
	cells [][]string,
	colWidths []int,
	maxRows int,
	padStyle lipgloss.Style,
	sep string,
) string {
	rows := make([]string, 0, maxRows)

	for ri := range maxRows {
		var sb strings.Builder

		for ci := range cells {
			if ci > 0 {
				sb.WriteString(sep)
			}

			if ri < len(cells[ci]) {
				cell := cells[ci][ri]

				if w := lipgloss.Width(cell); w < colWidths[ci] {
					cell += padStyle.Render(strings.Repeat(" ", colWidths[ci]-w))
				}

				sb.WriteString(cell)
			} else {
				sb.WriteString(padStyle.Render(strings.Repeat(" ", colWidths[ci])))
			}
		}

		rows = append(rows, sb.String())
	}

	return strings.Join(rows, "\n")
}

func renderBindings(bindings []key.Binding, keyStyle, descStyle, sepStyle lipgloss.Style) string {
	if len(bindings) == 0 {
		return ""
	}

	var parts []string

	for idx, b := range bindings {
		parts = append(parts, keyStyle.Render(b.Help().Key)+descStyle.Render(" "+b.Help().Desc))

		if idx < len(bindings)-1 {
			parts = append(parts, sepStyle.Render("  •  "))
		}
	}

	return strings.Join(parts, "")
}

func renderTruncatable(
	bindings []key.Binding,
	keyStyle, descStyle, sepStyle lipgloss.Style,
	maxWidth int,
) string {
	if maxWidth <= 0 || len(bindings) == 0 {
		return ""
	}

	sep := sepStyle.Render("  •  ")
	sepWidth := lipgloss.Width(sep)
	ellipsis := sepStyle.Render("…")
	ellipsisWidth := lipgloss.Width(ellipsis)

	var parts []string

	for _, b := range bindings {
		rendered := keyStyle.Render(b.Help().Key) + descStyle.Render(" "+b.Help().Desc)
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
