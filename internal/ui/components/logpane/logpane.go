package logpane

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/layout"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	defaultCap       = 1000
	horizontalStep   = 8
	borderWidth      = 2
	listMinTermWidth = 80
	listRatio        = 30
	pctDivisor       = 100
)

// Model stores raw log text lines backed by a viewport for windowed rendering.
type Model struct {
	lines    []string
	cap      int
	viewport viewport.Model
	lineCh   <-chan string
	th       *theme.Theme
	w        int
	h        int
	wrap     bool
}

// New returns a Model reading from the given line channel. lineCap sets the
// maximum number of lines retained; values <= 0 fall back to defaultCap.
func New(th *theme.Theme, w, h, lineCap int, lineCh <-chan string) Model {
	if lineCap <= 0 {
		lineCap = defaultCap
	}

	carouselW := max(w, listMinTermWidth) * listRatio / pctDivisor
	panW := max(w-carouselW, 0)
	usableH := max(0, h-layout.FrameHeight)

	vp := viewport.New(
		viewport.WithWidth(max(panW-borderWidth, 0)),
		viewport.WithHeight(max(usableH-borderWidth, 0)),
	)
	vp.KeyMap = viewport.KeyMap{
		Up:    viewport.DefaultKeyMap().Up,
		Down:  viewport.DefaultKeyMap().Down,
		Left:  viewport.DefaultKeyMap().Left,
		Right: viewport.DefaultKeyMap().Right,
	}
	vp.SetHorizontalStep(horizontalStep)
	vp.MouseWheelEnabled = true

	return Model{
		lines:    nil,
		cap:      lineCap,
		viewport: vp,
		lineCh:   lineCh,
		th:       th,
		w:        panW,
		h:        usableH,
		wrap:     false,
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update drains the line channel on availability signals and delegates to the viewport.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.LogLinesAvailable:
		return m.drainLines()

	case msgs.ToggleLogWrap:
		m.wrap = !m.wrap
		realIdx := m.realLineIndex(m.viewport.YOffset())
		wasAtBottom := m.viewport.AtBottom()

		if m.wrap {
			m.viewport.SetXOffset(0)
		}

		m.viewport.SoftWrap = m.wrap
		m.viewport.SetContentLines(m.lines)

		if wasAtBottom || m.viewport.PastBottom() {
			m.viewport.GotoBottom()
		} else {
			m.viewport.SetYOffset(realIdx)
		}

		return m, nil

	case msgs.ClearLogBuffer:
		m.lines = nil
		m.viewport.SetContent("")
		m.viewport.GotoBottom()
		m = m.drainLineCh()

		return m, nil

	case theme.Changed:
		m.th = msg.Theme

	case tea.WindowSizeMsg:
		wasAtBottom := m.viewport.AtBottom()
		carouselW := max(msg.Width, listMinTermWidth) * listRatio / pctDivisor
		m.w = msg.Width - carouselW
		m.h = max(0, msg.Height-layout.FrameHeight)
		m.viewport.SetWidth(max(m.w-borderWidth, 0))
		m.viewport.SetHeight(max(m.h-borderWidth, 0))

		if wasAtBottom || m.viewport.PastBottom() {
			m.viewport.GotoBottom()
		}
	}

	var cmd tea.Cmd

	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
}

func (m Model) drainLineCh() Model {
	if m.lineCh == nil {
		return m
	}

	for {
		select {
		case _, ok := <-m.lineCh:
			if !ok {
				m.lineCh = nil

				return m
			}
		default:
			return m
		}
	}
}

func (m Model) drainLines() (Model, tea.Cmd) {
	if m.lineCh == nil {
		return m, nil
	}

	for {
		select {
		case line, ok := <-m.lineCh:
			if !ok {
				m.lineCh = nil

				return m, nil
			}

			m.lines = append(m.lines, line)
			if len(m.lines) > m.cap {
				m.lines = m.lines[len(m.lines)-m.cap:]
			}
		default:
			wasAtBottom := m.viewport.AtBottom()
			m.viewport.SetContentLines(m.lines)
			m.viewport.SetHeight(max(m.h-borderWidth, 0))

			if wasAtBottom {
				m.viewport.GotoBottom()
			}

			return m, nil
		}
	}
}

// realLineIndex returns the real-line index corresponding to the given virtual
// YOffset, accounting for soft wrapping. Mirrors the viewport's internal
// calculateLine logic for precise scroll restoration on wrap toggle.
func (m Model) realLineIndex(yOffset int) int {
	if len(m.lines) == 0 {
		return 0
	}

	if !m.wrap {
		return min(yOffset, len(m.lines)-1)
	}

	maxW := max(m.viewport.Width(), 1)

	var total int

	for i, line := range m.lines {
		vLines := max(1, (ansi.StringWidth(line)+maxW-1)/maxW)
		if yOffset < total+vLines {
			return i
		}

		total += vLines
	}

	return max(0, len(m.lines)-1)
}

// View returns the viewport-rendered window of log lines with border, background,
// and foreground styling from the theme.
func (m Model) View() tea.View {
	content := m.viewport.View()
	if content != "" {
		content = lipgloss.NewStyle().
			Background(m.th.LogPaneBackground).
			Render(content)
	}

	return tea.NewView(m.th.BorderBlurred.
		Border(lipgloss.RoundedBorder()).
		BorderBackground(m.th.LogPaneBackground).
		Width(m.w).
		Background(m.th.LogPaneBackground).
		Render(content))
}
