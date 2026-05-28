//nolint:goconst // test fixture strings repeat across many test cases
package topbar_test

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/docker/connection"
	"github.com/ma-tf/ogle/internal/services/docker/mocks"
	"github.com/ma-tf/ogle/internal/ui/components/topbar"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

func newModel(t *testing.T) topbar.Model {
	t.Helper()
	mockD := mocks.NewMockDocker(t)
	mockD.EXPECT().Connect(mock.Anything).
		RunAndReturn(func(_ context.Context) tea.Cmd {
			return func() tea.Msg { return msgs.DaemonConnected{} }
		}).Maybe()

	return topbar.New(context.Background(), connection.New(), theme.Default(), mockD, zone.New())
}

func TestInit(t *testing.T) {
	t.Parallel()

	m := newModel(t)
	cmd := m.Init()
	require.NotNil(t, cmd)
}

func TestUpdate(t *testing.T) { //nolint:funlen // long table-driven test
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func(m topbar.Model) topbar.Model

		// act
		msg tea.Msg

		// assert
		expectedMsg tea.Msg
		expectCmd   bool
		assert      func(t *testing.T, m topbar.Model, cmd tea.Cmd)
	}

	cases := []testCase{
		{
			name: "TopbarContext produces no command",
			msg:  msgs.TopbarContext{Phase: "dashboard", File: "docker-compose.yml"},
		},
		{
			name: "LogWrapStatus wrap on produces no command",
			msg:  msgs.LogWrapStatus{On: true, Overflow: false},
		},
		{
			name: "LogWrapStatus overflow on produces no command",
			msg:  msgs.LogWrapStatus{On: false, Overflow: true},
		},
		{
			name:      "DaemonConnected schedules poll command",
			msg:       msgs.DaemonConnected{},
			expectCmd: true,
		},
		{
			name: "DaemonUnavailable schedules retry command",
			msg:  msgs.DaemonUnavailable{},
			// assert
			expectCmd: true,
		},
		{
			name: "DaemonGraceExpired while connecting schedules retry command",
			msg:  msgs.DaemonGraceExpired{},
			// assert
			expectCmd: true,
		},
		{
			name: "DaemonGraceExpired while connected produces no command",
			// arrange
			setup: func(m topbar.Model) topbar.Model {
				m.Update(msgs.DaemonConnected{})

				return m
			},
			// act
			msg: msgs.DaemonGraceExpired{},
		},
		{
			name: "DaemonTick with retry not due schedules next tick",
			// arrange
			setup: func(m topbar.Model) topbar.Model {
				m.Update(msgs.DaemonUnavailable{})

				return m
			},
			// act
			msg: msgs.DaemonTick{},
			// assert
			expectCmd: true,
		},
		{
			name: "DaemonPoll while Connected fires docker.Connect",
			setup: func(m topbar.Model) topbar.Model {
				m, _ = m.Update(msgs.DaemonConnected{})

				return m
			},
			msg:         msgs.DaemonPoll{},
			expectedMsg: msgs.DaemonConnected{},
		},
		{
			name: "DaemonPoll while not Connected produces no command",
			msg:  msgs.DaemonPoll{},
		},
		{
			name: "theme.Changed is reflected in View output",
			msg:  theme.Changed{Theme: theme.DefaultLight()},
			assert: func(t *testing.T, m topbar.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)

				view := m.View()
				assert.Contains(t, view.Content, "ogle")
				assert.Contains(t, view.Content, "scanning for compose files")
			},
		},
		{
			name: "WindowSizeMsg affects View padding",
			msg:  tea.WindowSizeMsg{Width: 120},
			assert: func(t *testing.T, m topbar.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)

				view := m.View()
				assert.Contains(t, view.Content, "ogle")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := newModel(t)
			if tc.setup != nil {
				m = tc.setup(m)
			}

			m, cmd := m.Update(tc.msg)

			if tc.assert != nil {
				tc.assert(t, m, cmd)

				return
			}

			switch {
			case tc.expectedMsg != nil:
				require.NotNil(t, cmd)
				require.Equal(t, tc.expectedMsg, cmd())
			case tc.expectCmd:
				require.NotNil(t, cmd)
			default:
				require.Nil(t, cmd)
			}
		})
	}
}

//nolint:funlen,goconst // long test with many table-driven cases
func TestView(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func(m topbar.Model) topbar.Model

		// assert
		expectedResult string
		assert         func(t *testing.T, m topbar.Model)
	}

	cases := []testCase{
		{
			name:           "startup phase shows ogle brand and scanning text",
			expectedResult: "ogle",
		},
		{
			name:           "startup phase shows scanning text",
			expectedResult: "scanning for compose files",
		},
		{
			name: "dashboard phase shows project file",
			setup: func(m topbar.Model) topbar.Model {
				m, _ = m.Update(msgs.TopbarContext{Phase: "dashboard", File: "compose.yaml"})

				return m
			},
			expectedResult: "compose.yaml",
		},
		{
			name: "watching phase shows disconnected text",
			setup: func(m topbar.Model) topbar.Model {
				m, _ = m.Update(msgs.TopbarContext{Phase: "watching", File: ""})

				return m
			},
			expectedResult: "disconnected",
		},
		{
			name:           "connecting daemon status shows RECONNECTING",
			expectedResult: "RECONNECTING",
		},
		{
			name: "connected daemon status shows LIVE",
			setup: func(m topbar.Model) topbar.Model {
				m, _ = m.Update(msgs.DaemonConnected{})

				return m
			},
			expectedResult: "LIVE",
		},
		{
			name: "unavailable daemon status shows DISCONNECTED",
			setup: func(m topbar.Model) topbar.Model {
				m, _ = m.Update(msgs.DaemonUnavailable{})

				return m
			},
			expectedResult: "DISCONNECTED",
		},
		{
			name: "selected service name shown after arrow in dashboard phase",
			setup: func(m topbar.Model) topbar.Model {
				m, _ = m.Update(
					msgs.TopbarContext{Phase: "dashboard", File: "compose.yaml", Service: "api"},
				)

				return m
			},
			expectedResult: "compose.yaml → api",
		},
		{
			name: "no arrow when service name is empty",
			setup: func(m topbar.Model) topbar.Model {
				m, _ = m.Update(msgs.TopbarContext{Phase: "dashboard", File: "compose.yaml"})

				return m
			},
			expectedResult: "compose.yaml",
			assert: func(t *testing.T, m topbar.Model) {
				t.Helper()
				assert.NotContains(t, m.View().Content, "→")
			},
		},
		{
			name: "WRAP badge shown when LogWrapStatus matches active service",
			setup: func(m topbar.Model) topbar.Model {
				m, _ = m.Update(
					msgs.TopbarContext{Phase: "dashboard", File: "compose.yaml", Service: "api"},
				)
				m, _ = m.Update(msgs.LogWrapStatus{On: true, Overflow: false, ServiceName: "api"})

				return m
			},
			expectedResult: "WRAP",
		},
		{
			name: "WRAP badge not shown when LogWrapStatus matches different service",
			setup: func(m topbar.Model) topbar.Model {
				m, _ = m.Update(
					msgs.TopbarContext{Phase: "dashboard", File: "compose.yaml", Service: "api"},
				)
				m, _ = m.Update(msgs.LogWrapStatus{On: true, Overflow: false, ServiceName: "db"})

				return m
			},
			assert: func(t *testing.T, m topbar.Model) {
				t.Helper()

				view := m.View().Content
				assert.NotContains(t, view, "WRAP")
				assert.NotContains(t, view, ">>")
			},
		},
		{
			name: "badge cleared when switching to a different service",
			setup: func(m topbar.Model) topbar.Model {
				m, _ = m.Update(
					msgs.TopbarContext{Phase: "dashboard", File: "compose.yaml", Service: "api"},
				)
				m, _ = m.Update(msgs.LogWrapStatus{On: true, Overflow: false, ServiceName: "api"})
				m, _ = m.Update(
					msgs.TopbarContext{Phase: "dashboard", File: "compose.yaml", Service: "db"},
				)

				return m
			},
			assert: func(t *testing.T, m topbar.Model) {
				t.Helper()

				view := m.View().Content
				assert.NotContains(t, view, "WRAP")
				assert.NotContains(t, view, ">>")
			},
		},
		{
			name: "WRAP badge shown when wrapping is ON",
			setup: func(m topbar.Model) topbar.Model {
				m, _ = m.Update(msgs.TopbarContext{Phase: "dashboard", File: "compose.yaml"})
				m, _ = m.Update(msgs.LogWrapStatus{On: true, Overflow: false})

				return m
			},
			expectedResult: "WRAP",
		},
		{
			name: "badge not shown when wrapping OFF and no overflow",
			setup: func(m topbar.Model) topbar.Model {
				m, _ = m.Update(msgs.TopbarContext{Phase: "dashboard", File: "compose.yaml"})
				m, _ = m.Update(msgs.LogWrapStatus{On: false, Overflow: false})

				return m
			},
			expectedResult: "compose.yaml",
			assert: func(t *testing.T, m topbar.Model) {
				t.Helper()

				view := m.View().Content
				assert.NotContains(t, view, "WRAP")
				assert.NotContains(t, view, ">>")
			},
		},
		{
			name: "truncation badge shown when overflow and wrap OFF",
			setup: func(m topbar.Model) topbar.Model {
				m, _ = m.Update(msgs.TopbarContext{Phase: "dashboard", File: "compose.yaml"})
				m, _ = m.Update(msgs.LogWrapStatus{On: false, Overflow: true})

				return m
			},
			expectedResult: ">>",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := newModel(t)
			_ = m.Init()

			if tc.setup != nil {
				m = tc.setup(m)
			}

			switch {
			case tc.assert != nil:
				tc.assert(t, m)
			case tc.expectedResult == "":
				assert.Empty(t, m.View().Content)
			default:
				assert.Contains(t, m.View().Content, tc.expectedResult)
			}
		})
	}
}
