// Package helpbar provides a self-contained help bar component that renders a
// keymap received via BindingsMsg.
package helpbar

import (
	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// Model is a value-type sub-model that renders a help bar.
type Model struct {
	help   help.Model
	keymap help.KeyMap
	th     *theme.Theme
	width  int
}

// New returns a Model styled with th.
func New(th *theme.Theme) Model {
	m := Model{
		help:   help.New(),
		keymap: nil,
		th:     th,
		width:  0,
	}

	m.help = m.applyStyles()

	return m
}

// applyStyles pushes the current theme colours into the underlying help.Model.
// The background is applied to each slot directly so that the characters
// rendered by help.Model carry the correct background in their own ANSI
// sequences — the outer Width wrapper alone cannot override sequences already
// embedded in the rendered string.
func (m Model) applyStyles() help.Model {
	bg := m.th.HelpBackground
	m.help.Styles = help.Styles{
		ShortKey:       m.th.HelpKey.Background(bg),
		ShortDesc:      m.th.HelpDesc.Background(bg),
		ShortSeparator: m.th.HelpSep.Background(bg),
		FullKey:        m.th.HelpKey.Background(bg),
		FullDesc:       m.th.HelpDesc.Background(bg),
		FullSeparator:  m.th.HelpSep.Background(bg),
		Ellipsis:       m.th.HelpSep.Background(bg),
	}

	return m.help
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
		m.help.SetWidth(msg.Width)
	case msgs.BindingsMsg:
		m.keymap = msg.Keymap
	case theme.Changed:
		m.th = msg.Theme
		m.help = m.applyStyles()
	case ToggleMsg:
		m.help.ShowAll = !m.help.ShowAll
	}

	return m, nil
}

// ShowAll reports whether the help is in full (ShowAll=true) or compact mode.
func (m Model) ShowAll() bool {
	return m.help.ShowAll
}

// View renders the help bar with the current keymap.
func (m Model) View() tea.View {
	if m.keymap == nil {
		return tea.NewView("")
	}

	content := lipgloss.NewStyle().
		Background(m.th.HelpBackground).
		Width(m.width).
		Render(m.help.View(m.keymap))

	return tea.NewView(content)
}
