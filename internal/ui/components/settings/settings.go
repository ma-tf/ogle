package settings

import (
	"strconv"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/config"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

type savedClearMsg struct{}

const (
	boxWidth        = 50
	labelWidth      = 14
	fieldValueWidth = 24
)

const (
	fieldTheme = iota
	fieldLogBufferCap
	fieldCount
)

const (
	savedTimeTick = 1500 * time.Millisecond

	logBufCapMin  = 500
	logBufCapMax  = 5000
	logBufCapStep = 500
)

//nolint:gochecknoglobals // package-level bindings shared across instances
var (
	keyPrevField = key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "prev"))
	keyNextField = key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "next"))
	keyDec       = key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "dec"))
	keyInc       = key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "inc"))
	keyClose     = key.NewBinding(key.WithKeys("esc", "q", ","), key.WithHelp("esc", "close"))
)

// Model is the settings overlay with interactive fields.
type Model struct {
	th         *theme.Theme
	cfg        config.Config
	themeNames []string
	themeIdx   int
	focusField int
	w, h       int
	savedTime  time.Time
}

// New returns a Model initialised from cfg.
func New(th *theme.Theme, cfg config.Config, w, h int) Model {
	names := theme.BuiltinNames()
	idx := 0

	for i, n := range names {
		if n == cfg.Theme {
			idx = i

			break
		}
	}

	return Model{
		th:         th,
		cfg:        cfg,
		themeNames: names,
		themeIdx:   idx,
		focusField: 0,
		w:          w,
		h:          h,
		savedTime:  time.Time{},
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update satisfies tea.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height
	case savedClearMsg:
		if time.Since(m.savedTime) >= 1500*time.Millisecond {
			m.savedTime = time.Time{}
		}

		return m, nil
	case theme.Changed:
		m.th = msg.Theme
	}

	return m, nil
}

func (m Model) handleKeyPress(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keyPrevField):
		m.focusField = (m.focusField - 1 + fieldCount) % fieldCount

		return m, nil
	case key.Matches(msg, keyNextField):
		m.focusField = (m.focusField + 1) % fieldCount

		return m, nil
	case key.Matches(msg, keyClose):
		return m, func() tea.Msg {
			return msgs.SettingsVisibilityChanged{Visible: false}
		}
	}

	switch m.focusField {
	case fieldTheme:
		switch {
		case key.Matches(msg, keyDec):
			m.themeIdx = (m.themeIdx - 1 + len(m.themeNames)) % len(m.themeNames)
			m.savedTime = time.Now()

			return m, tea.Batch(
				m.settingsAppliedCmd(),
				tea.Tick(savedTimeTick, func(_ time.Time) tea.Msg {
					return savedClearMsg{}
				}),
			)
		case key.Matches(msg, keyInc):
			m.themeIdx = (m.themeIdx + 1) % len(m.themeNames)
			m.savedTime = time.Now()

			return m, tea.Batch(
				m.settingsAppliedCmd(),
				tea.Tick(savedTimeTick, func(_ time.Time) tea.Msg {
					return savedClearMsg{}
				}),
			)
		}

	case fieldLogBufferCap:
		switch {
		case key.Matches(msg, keyDec):
			m.cfg.LogBufferCap = max(m.cfg.LogBufferCap-logBufCapStep, logBufCapMin)
			m.savedTime = time.Now()

			return m, tea.Batch(
				m.settingsAppliedCmd(),
				tea.Tick(savedTimeTick, func(_ time.Time) tea.Msg {
					return savedClearMsg{}
				}),
			)
		case key.Matches(msg, keyInc):
			m.cfg.LogBufferCap = min(m.cfg.LogBufferCap+logBufCapStep, logBufCapMax)
			m.savedTime = time.Now()

			return m, tea.Batch(
				m.settingsAppliedCmd(),
				tea.Tick(savedTimeTick, func(_ time.Time) tea.Msg {
					return savedClearMsg{}
				}),
			)
		}
	}

	return m, nil
}

func (m Model) settingsAppliedCmd() tea.Cmd {
	return func() tea.Msg {
		return msgs.SettingsApplied{
			Theme:        m.themeNames[m.themeIdx],
			LogBufferCap: m.cfg.LogBufferCap,
		}
	}
}

// View renders the settings overlay.
func (m Model) View() tea.View {
	title := lipgloss.NewStyle().
		Bold(true).
		Width(boxWidth).
		AlignHorizontal(lipgloss.Center).
		Foreground(m.th.Text).
		Render("Settings")

	themeField := m.renderField("Theme",
		m.themeNames[m.themeIdx],
		m.focusField == fieldTheme,
	)

	capField := m.renderField("Log Buffer Cap",
		strconv.Itoa(m.cfg.LogBufferCap),
		m.focusField == fieldLogBufferCap,
	)

	helpTxt := "↑↓ nav · ← → adjust · esc close"
	if time.Since(m.savedTime) < savedTimeTick {
		helpTxt += lipgloss.NewStyle().Foreground(m.th.StateRunning).Render("  ✓")
	}

	help := lipgloss.NewStyle().
		Foreground(m.th.StateMuted).
		Width(boxWidth).
		AlignHorizontal(lipgloss.Center).
		Render(helpTxt)

	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		themeField,
		capField,
		"",
		help,
	)

	return tea.NewView(lipgloss.NewStyle().
		Width(boxWidth).
		Padding(0, 1).
		Background(m.th.BodyBackground).
		Render(content))
}

func (m Model) renderField(label, value string, focused bool) string {
	borderStyle := m.th.BorderBlurred
	if focused {
		borderStyle = m.th.BorderFocused
	}

	valFg := m.th.Text
	if !focused {
		valFg = m.th.Subtext
	}

	valBox := borderStyle.
		Foreground(valFg).
		BorderBackground(m.th.BodyBackground).
		Background(m.th.BodyBackground).
		Width(fieldValueWidth).
		Render(value)

	return lipgloss.JoinHorizontal(lipgloss.Center,
		lipgloss.NewStyle().
			Width(labelWidth).
			AlignHorizontal(lipgloss.Right).
			Foreground(m.th.Subtext).
			Background(m.th.BodyBackground).
			Render(label),
		lipgloss.NewStyle().Background(m.th.BodyBackground).Render("  "),
		valBox,
	)
}
