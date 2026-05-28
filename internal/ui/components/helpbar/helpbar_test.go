package helpbar_test

import (
	"testing"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/helpbar"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

type testKeymap struct{}

func (k testKeymap) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}
}

func (k testKeymap) FullHelp() [][]key.Binding { return nil }

var _ help.KeyMap = testKeymap{}

type fullKeymap struct{}

func (k fullKeymap) ShortHelp() []key.Binding {
	return []key.Binding{key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit"))}
}

func (k fullKeymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
			key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		},
		{
			key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "up")),
			key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "down")),
		},
	}
}

var _ help.KeyMap = fullKeymap{}

// pinnedKeymap implements PinnedKeyMap with pinned and truncatable bindings.
type pinnedKeymap struct{}

func (k pinnedKeymap) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "toggle wrap")),
		key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
		key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}
}

func (k pinnedKeymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "toggle wrap")),
			key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
			key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		},
	}
}

func (k pinnedKeymap) PinnedHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
		key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}
}

var _ helpbar.PinnedKeyMap = pinnedKeymap{}

// narrowKeymap has a short binding that won't fit at width 10.
type narrowKeymap struct{}

func (k narrowKeymap) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "focus next")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "start/stop")),
	}
}

func (k narrowKeymap) FullHelp() [][]key.Binding { return nil }

func (k narrowKeymap) PinnedHelp() []key.Binding {
	return []key.Binding{key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit"))}
}

var _ helpbar.PinnedKeyMap = narrowKeymap{}

func TestInit(t *testing.T) {
	t.Parallel()

	m := helpbar.New(theme.Default())
	cmd := m.Init()

	require.Nil(t, cmd)
}

func TestView(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func(m helpbar.Model) helpbar.Model
		// assert
		expectedResult string
	}

	cases := []testCase{
		{
			name:           "initial state returns empty view",
			expectedResult: "",
		},
		{
			name: "bindings msg populates view",
			setup: func(m helpbar.Model) helpbar.Model {
				m, _ = m.Update(tea.WindowSizeMsg{Width: 100})
				m, _ = m.Update(msgs.BindingsMsg{Keymap: testKeymap{}})

				return m
			},
			expectedResult: "quit",
		},
		{
			name: "window resize does not affect empty view",
			setup: func(m helpbar.Model) helpbar.Model {
				m, _ = m.Update(tea.WindowSizeMsg{Width: 80})

				return m
			},
			expectedResult: "",
		},
		{
			name: "keymap persists across window resize",
			setup: func(m helpbar.Model) helpbar.Model {
				m, _ = m.Update(msgs.BindingsMsg{Keymap: testKeymap{}})
				m, _ = m.Update(tea.WindowSizeMsg{Width: 80})

				return m
			},
			expectedResult: "q",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := helpbar.New(theme.Default())
			_ = m.Init()

			if tc.setup != nil {
				m = tc.setup(m)
			}

			if tc.expectedResult == "" {
				assert.Empty(t, m.View().Content)
			} else {
				assert.Contains(t, m.View().Content, tc.expectedResult)
			}
		})
	}
}

func TestToggle(t *testing.T) {
	t.Parallel()

	m := helpbar.New(theme.Default())
	m, _ = m.Update(msgs.BindingsMsg{Keymap: fullKeymap{}})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100})
	compact := m.View().Content

	m, _ = m.Update(helpbar.ToggleMsg{})
	expanded := m.View().Content

	assert.NotEqual(t, compact, expanded, "toggle should change view from compact to expanded")
}

func TestToggle_BackAndForth(t *testing.T) {
	t.Parallel()

	m := helpbar.New(theme.Default())
	m, _ = m.Update(msgs.BindingsMsg{Keymap: fullKeymap{}})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100})
	first := m.View().Content

	m, _ = m.Update(helpbar.ToggleMsg{})
	expanded := m.View().Content
	assert.NotEqual(t, first, expanded, "expanded help should differ from compact")

	m, _ = m.Update(helpbar.ToggleMsg{})
	collapsed := m.View().Content
	assert.Equal(t, first, collapsed, "toggle twice should restore original")
}

func TestToggle_ViewShowsFullHelp(t *testing.T) {
	t.Parallel()

	m := helpbar.New(theme.Default())
	m, _ = m.Update(msgs.BindingsMsg{Keymap: fullKeymap{}})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100})

	m, _ = m.Update(helpbar.ToggleMsg{})
	content := m.View().Content
	assert.Contains(t, content, "quit")
	assert.Contains(t, content, "help")
	assert.Contains(t, content, "up")
	assert.Contains(t, content, "down")
}

func TestCompact_PinnedRightAligned(t *testing.T) {
	t.Parallel()

	m := helpbar.New(theme.Default())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100})
	m, _ = m.Update(msgs.BindingsMsg{Keymap: pinnedKeymap{}})
	content := m.View().Content

	assert.Contains(t, content, "toggle wrap")
	assert.Contains(t, content, "toggle help")
	assert.Contains(t, content, "quit")
}

func TestCompact_PinnedNeverTruncated(t *testing.T) {
	t.Parallel()

	m := helpbar.New(theme.Default())
	// Narrow width that only fits pinned
	m, _ = m.Update(tea.WindowSizeMsg{Width: 30})
	m, _ = m.Update(msgs.BindingsMsg{Keymap: narrowKeymap{}})
	content := m.View().Content

	assert.Contains(t, content, "quit",
		"pinned bindings should be visible even at narrow width")
}

func TestCompact_BackwardsCompat(t *testing.T) {
	t.Parallel()

	m := helpbar.New(theme.Default())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100})
	m, _ = m.Update(msgs.BindingsMsg{Keymap: testKeymap{}})
	content := m.View().Content

	assert.Contains(t, content, "quit",
		"non-PinnedKeyMap keymap should still render bindings")
}

func TestCompact_EllipsisOnOverflow(t *testing.T) {
	t.Parallel()

	m := helpbar.New(theme.Default())
	// Very narrow width
	m, _ = m.Update(tea.WindowSizeMsg{Width: 15})
	m, _ = m.Update(msgs.BindingsMsg{Keymap: pinnedKeymap{}})
	content := m.View().Content

	assert.Contains(t, content, "quit",
		"pinned quit should be visible")
	assert.Contains(t, content, "toggle help",
		"pinned help toggle should be visible")
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// act
		msg tea.Msg
		// assert
		expectedMsg tea.Msg
	}

	cases := []testCase{
		{
			name:        "WindowSizeMsg returns no command",
			msg:         tea.WindowSizeMsg{Width: 80},
			expectedMsg: nil,
		},
		{
			name:        "BindingsMsg returns no command",
			msg:         msgs.BindingsMsg{Keymap: testKeymap{}},
			expectedMsg: nil,
		},
		{
			name:        "theme.Changed returns no command",
			msg:         theme.Changed{Theme: theme.Default()},
			expectedMsg: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := helpbar.New(theme.Default())
			_ = m.Init()

			_, cmd := m.Update(tc.msg)

			if tc.expectedMsg != nil {
				require.NotNil(t, cmd)
				require.Equal(t, tc.expectedMsg, cmd())
			} else {
				require.Nil(t, cmd)
			}
		})
	}
}
