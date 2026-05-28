package startup_test

import (
	"errors"
	"testing"

	bubbleskey "charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/parser/mocks"
	"github.com/ma-tf/ogle/internal/ui/flows/startup"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

//nolint:gochecknoglobals // shared test fixtures
var (
	project  = &domain.Project{Name: "myapp", File: "/path/to/compose.yml"}
	errParse = errors.New("parse error")
)

func newModel(t *testing.T) (startup.Model, *mocks.MockParser) {
	t.Helper()
	mockP := mocks.NewMockParser(t)

	return startup.New(100, 50, zone.New(), theme.Default(), mockP), mockP
}

func TestShortHelp(t *testing.T) {
	t.Parallel()

	m, _ := newModel(t)
	cmd := m.Init()
	require.NotNil(t, cmd)

	bindingsMsg, ok := cmd().(msgs.BindingsMsg)
	require.True(t, ok)

	bindings := bindingsMsg.Keymap.ShortHelp()
	require.Len(t, bindings, 3)

	assert.Equal(t, "↑/k", bindings[0].Help().Key)
	assert.Equal(t, "up", bindings[0].Help().Desc)
	assert.Equal(t, "↓/j", bindings[1].Help().Key)
	assert.Equal(t, "down", bindings[1].Help().Desc)
	assert.Equal(t, "enter", bindings[2].Help().Key)
	assert.Equal(t, "select", bindings[2].Help().Desc)
}

func TestFullHelp(t *testing.T) {
	t.Parallel()

	m, _ := newModel(t)
	cmd := m.Init()
	require.NotNil(t, cmd)

	bindingsMsg, ok := cmd().(msgs.BindingsMsg)
	require.True(t, ok)

	bindings := bindingsMsg.Keymap.FullHelp()
	require.Len(t, bindings, 1)
	require.Len(t, bindings[0], 6)

	assert.Equal(t, "↑/k", bindings[0][0].Help().Key)
	assert.Equal(t, "up", bindings[0][0].Help().Desc)
	assert.Equal(t, "↓/j", bindings[0][1].Help().Key)
	assert.Equal(t, "down", bindings[0][1].Help().Desc)
	assert.Equal(t, "enter", bindings[0][2].Help().Key)
	assert.Equal(t, "select", bindings[0][2].Help().Desc)
	assert.Equal(t, "?", bindings[0][3].Help().Key)
	assert.Equal(t, "toggle help", bindings[0][3].Help().Desc)
	assert.Equal(t, "q", bindings[0][4].Help().Key)
	assert.Equal(t, "quit", bindings[0][4].Help().Desc)
	assert.Equal(t, "f1", bindings[0][5].Help().Key)
	assert.Equal(t, "about", bindings[0][5].Help().Desc)
}

func TestKeymapPinnedHelp(t *testing.T) {
	t.Parallel()

	m, _ := newModel(t)
	cmd := m.Init()
	require.NotNil(t, cmd)

	bindMsg, ok := cmd().(msgs.BindingsMsg)
	require.True(t, ok)

	km, ok := bindMsg.Keymap.(interface {
		PinnedHelp() []bubbleskey.Binding
	})
	require.True(t, ok, "keymap should implement PinnedHelp")

	pinned := km.PinnedHelp()
	require.Len(t, pinned, 2)
	assert.Equal(t, "?", pinned[0].Help().Key)
	assert.Equal(t, "toggle help", pinned[0].Help().Desc)
	assert.Equal(t, "q", pinned[1].Help().Key)
	assert.Equal(t, "quit", pinned[1].Help().Desc)
}

func TestKeymapShortHelpExcludesQuit(t *testing.T) {
	t.Parallel()

	m, _ := newModel(t)
	cmd := m.Init()
	require.NotNil(t, cmd)

	bindMsg, ok := cmd().(msgs.BindingsMsg)
	require.True(t, ok)

	bindings := bindMsg.Keymap.ShortHelp()
	for _, b := range bindings {
		assert.NotEqual(t, "q", b.Help().Key, "ShortHelp should not include quit")
		assert.NotEqual(t, "?", b.Help().Key, "ShortHelp should not include help toggle")
		assert.NotEqual(t, "f1", b.Help().Key, "ShortHelp should not include about")
	}
}

func TestInit(t *testing.T) {
	t.Parallel()

	m, _ := newModel(t)
	cmd := m.Init()
	require.NotNil(t, cmd)
	require.IsType(t, msgs.BindingsMsg{}, cmd())
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func(m startup.Model, p *mocks.MockParser) startup.Model

		// act
		msg tea.Msg

		// assert
		expectedMsg tea.Msg
	}

	cases := []testCase{
		{
			name: "FileSelected emits ProjectLoaded",
			// arrange
			setup: func(m startup.Model, p *mocks.MockParser) startup.Model {
				p.EXPECT().Parse("test/path/file.yml").Return(project, nil)

				return m
			},
			// act
			msg: msgs.FileSelected{Path: "test/path/file.yml"},
			// assert
			expectedMsg: msgs.ProjectLoaded{Project: project},
		},
		{
			name: "FileSelected with parse error returns no command",
			// arrange
			setup: func(m startup.Model, p *mocks.MockParser) startup.Model {
				p.EXPECT().Parse("test/path/file.yml").Return(nil, errParse)

				return m
			},
			// act
			msg: msgs.FileSelected{Path: "test/path/file.yml"},
		},
		{
			name: "WindowSizeMsg forwards to fileselect",
			// act
			msg: tea.WindowSizeMsg{Width: 120, Height: 80},
		},
		{
			name: "Unknown message falls through to fileselect",
			// act
			msg: msgs.ToggleLogWrap{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m, mockP := newModel(t)
			if tc.setup != nil {
				m = tc.setup(m, mockP)
			}

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

func TestView(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func(m startup.Model) startup.Model

		// assert
		expectedResult string
	}

	cases := []testCase{
		{
			name:           "delegates to fileselect view",
			expectedResult: "file",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m, _ := newModel(t)
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
