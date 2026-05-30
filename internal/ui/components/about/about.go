// Package about implements a read-only About overlay displaying version info,
// ASCII art branding, and a clickable GitHub URL.
package about

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ma-tf/ogle/internal/ui/theme"
	"github.com/ma-tf/ogle/internal/version"
)

const (
	boxExtraWidth = 10
	boxPadding    = 2
)

// Model is a read-only about overlay.
type Model struct {
	th *theme.Theme
	w  int
	h  int
}

// New returns a Model.
func New(th *theme.Theme) Model {
	return Model{th: th, w: 0, h: 0}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update satisfies tea.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height
	case theme.Changed:
		m.th = msg.Theme
	}

	return m, nil
}

// View renders the about overlay.
func (m Model) View() tea.View {
	url := ansi.SetHyperlink("https://github.com/ma-tf/ogle") +
		"github.com/ma-tf/ogle" +
		ansi.ResetHyperlink()

	// Determine content width before adding background.
	raw := []string{
		"ogle",
		"",
		version.ASCIIArt,
		"",
		version.Version + " (commit: " + version.Commit + ", built: " + version.Date + ")",
		"",
		url,
		"",
		"F1 / esc / q to close",
	}

	contentWidth := 0

	for _, item := range raw {
		if w := lipgloss.Width(item); w > contentWidth {
			contentWidth = w
		}
	}

	boxW := contentWidth + boxExtraWidth

	bg := lipgloss.NewStyle().
		Width(boxW).
		Align(lipgloss.Center).
		Background(m.th.AboutBackground)

	styled := []string{
		bg.Foreground(m.th.AboutTitleColour).Bold(true).Render(raw[0]),
		bg.Render(raw[1]),
		bg.Foreground(m.th.AboutArtColour).Render(raw[2]),
		bg.Render(raw[3]),
		bg.Foreground(m.th.AboutTextColour).Render(raw[4]),
		bg.Render(raw[5]),
		bg.Foreground(m.th.AboutLinkColour).Render(raw[6]),
		bg.Render(raw[7]),
		bg.Foreground(m.th.AboutHintColour).Render(raw[8]),
	}

	content := lipgloss.JoinVertical(lipgloss.Top, styled...)

	return tea.NewView(lipgloss.NewStyle().
		Padding(0, boxPadding).
		Background(m.th.AboutBackground).
		Render(content))
}
