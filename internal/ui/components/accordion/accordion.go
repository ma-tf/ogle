package accordion

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/accordion/value"
	"github.com/ma-tf/ogle/internal/ui/layout"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	dash                = "—"
	shortIDLen          = 12
	labelWidth          = 14
	secsPerMinute       = 60
	secsPerHour         = 3600
	numFields           = 5
	zoneAccordionHeader = "accordion-header"
)

// Model is the accordion component state.
type Model struct {
	services     []domain.ServiceDef
	runtime      *domain.ServiceRuntimeData
	selectedName string
	w, h         int
	th           *theme.Theme
	values       [numFields]value.Model
	scrollGen    int
	lastRaws     [numFields]string
	lastColours  [numFields]color.Color
	lastWidth    int
	zm           *zone.Manager
	collapsed    bool
	hovered      bool
}

// New returns a Model with the given project, dimensions, theme, and zone
// manager.
func New(project *domain.Project, w, h int, th *theme.Theme, zm *zone.Manager) Model {
	selectedName := ""
	if len(project.Services) > 0 {
		selectedName = project.Services[0].Name
	}

	m := Model{
		services:     project.Services,
		selectedName: selectedName,
		runtime:      nil,
		w:            w,
		h:            h,
		th:           th,
		values:       [numFields]value.Model{},
		scrollGen:    0,
		lastRaws:     [numFields]string{},
		lastColours:  [numFields]color.Color{},
		lastWidth:    -1,
		zm:           zm,
		collapsed:    false,
		hovered:      false,
	}
	m, _ = m.syncValues()

	return m
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update handles selection, runtime data, resize, and theme changes.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height

		return m.syncValues()

	case theme.Changed:
		m.th = msg.Theme

		return m.syncValues()

	case msgs.ServiceSelected:
		m.selectedName = msg.ServiceName

		return m.syncValues()

	case msgs.ServicesPolled:
		if msg.Err == nil && m.selectedName != "" {
			m.runtime = msg.Runtimes[m.selectedName]
		}

		return m.syncValues()

	case tea.MouseClickMsg:
		if m.zm != nil && m.zm.Get(zoneAccordionHeader).InBounds(msg) {
			m.collapsed = !m.collapsed
		}

		return m, nil

	case tea.MouseMotionMsg:
		m.hovered = m.zm != nil && m.zm.Get(zoneAccordionHeader).InBounds(msg)

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

// View renders the inspector detail for the selected service.
func (m Model) View() tea.View {
	if m.selectedName == "" || m.w == 0 {
		return tea.NewView("")
	}

	vw := m.valueWidth()
	bg := m.th.AccordionBackground

	indicator := "▼"
	if m.collapsed {
		indicator = "▶"
	}

	headerBg := m.th.AccordionHeaderBackground
	if m.hovered {
		headerBg = m.th.AccordionHeaderHoverBackground
	}

	headerStr := lipgloss.NewStyle().
		Width(labelWidth + vw).
		Foreground(m.th.AccordionLabel).
		Background(headerBg).
		Render(" " + indicator + " Service Details")
	if m.zm != nil {
		headerStr = m.zm.Mark(zoneAccordionHeader, headerStr)
	}

	if m.collapsed {
		return tea.NewView(headerStr)
	}

	labels := []string{
		"Image:        ",
		"Container ID: ",
		"Created:      ",
		"State:        ",
		"Ports:        ",
	}

	def, ok := m.lookupDef(m.selectedName)
	skipImage := ok && def.Image == ""

	labelParts := make([]string, 0, len(labels))
	valParts := make([]string, 0, len(labels))

	for i := range numFields {
		if i == 0 && skipImage {
			continue
		}

		labelParts = append(labelParts, labels[i])
		valParts = append(valParts, m.values[i].View().Content)
	}

	labelBlock := lipgloss.JoinVertical(lipgloss.Left, labelParts...)
	labelCol := lipgloss.NewStyle().
		Width(labelWidth).
		Foreground(m.th.AccordionLabel).
		Background(bg).
		Render(labelBlock)

	valBlock := lipgloss.JoinVertical(lipgloss.Left, valParts...)
	valCol := lipgloss.NewStyle().
		Width(vw).
		Foreground(m.th.AccordionValue).
		Background(bg).
		Render(valBlock)

	fullContent := lipgloss.JoinVertical(lipgloss.Top,
		headerStr,
		lipgloss.JoinHorizontal(lipgloss.Top, labelCol, valCol),
	)

	return tea.NewView(fullContent)
}

func (m Model) syncValues() (Model, tea.Cmd) {
	vw := m.valueWidth()
	if m.selectedName == "" || vw <= 0 {
		for i := range m.values {
			m.values[i] = value.New("", m.th.AccordionValue, m.th.AccordionBackground, 0)
		}

		m.lastWidth = -1

		return m, nil
	}

	bg := m.th.AccordionBackground
	raws, colours := m.computeFieldContent()

	if raws == m.lastRaws && colours == m.lastColours && vw == m.lastWidth {
		return m, nil
	}

	m.scrollGen++

	for i := range m.values {
		m.values[i] = value.New(raws[i], colours[i], bg, vw)
	}

	cmds := []tea.Cmd{func() tea.Msg {
		return value.StartMsg{Gen: m.scrollGen}
	}}

	m.lastRaws = raws
	m.lastColours = colours
	m.lastWidth = vw

	return m, tea.Batch(cmds...)
}

func (m Model) computeFieldContent() ([numFields]string, [numFields]color.Color) {
	def, ok := m.lookupDef(m.selectedName)

	var (
		raws    [numFields]string
		colours [numFields]color.Color
	)

	if !ok {
		for i := range raws {
			raws[i] = ""
			colours[i] = m.th.AccordionValue
		}

		return raws, colours
	}

	stateStr := dash
	containerID := dash
	createdAt := dash

	if m.runtime != nil {
		if m.runtime.Status != "" {
			stateStr = m.runtime.Status
		}

		if m.runtime.ContainerID != "" {
			containerID = m.runtime.ContainerID
		}

		if len(containerID) > shortIDLen {
			containerID = containerID[:shortIDLen]
		}

		if !m.runtime.CreatedAt.IsZero() {
			createdAt = formatAge(time.Since(m.runtime.CreatedAt))
		}
	}

	stateColour := m.th.AccordionValue
	if m.runtime != nil {
		stateColour = m.th.ColourForState(m.runtime.State)
	}

	portsStr := strings.Join(def.Ports, ", ")
	if portsStr == "" {
		portsStr = dash
	}

	raws = [numFields]string{def.Image, containerID, createdAt, stateStr, portsStr}
	colours = [numFields]color.Color{
		m.th.AccordionValue,
		m.th.AccordionValue,
		m.th.AccordionValue,
		stateColour,
		m.th.AccordionValue,
	}

	return raws, colours
}

func (m Model) valueWidth() int {
	return max(0, layout.SidebarWidth-labelWidth)
}

func (m Model) lookupDef(name string) (domain.ServiceDef, bool) {
	for _, svc := range m.services {
		if svc.Name == name {
			return svc, true
		}
	}

	return domain.ServiceDef{}, false
}

func formatAge(d time.Duration) string {
	secs := max(int(d.Seconds()), 0)

	switch {
	case secs < secsPerMinute:
		return fmt.Sprintf("%ds ago", secs)
	case secs < secsPerHour:
		return fmt.Sprintf("%dm ago", secs/secsPerMinute)
	default:
		return fmt.Sprintf("%dh ago", secs/secsPerHour)
	}
}
