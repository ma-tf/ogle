package dashboard_test

import (
	"context"
	"log/slog"
	"testing"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/config"
	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	dockermocks "github.com/ma-tf/ogle/internal/services/docker/mocks"
	parsermocks "github.com/ma-tf/ogle/internal/services/parser/mocks"
	"github.com/ma-tf/ogle/internal/ui/flows/dashboard"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	svcWeb = "web"
	svcAPI = "api"
)

//nolint:gochecknoglobals // shared test fixtures
var testProject = &domain.Project{
	Name: "testproj",
	File: "/path/to/compose.yaml",
	Services: []domain.ServiceDef{
		{Name: svcWeb, Image: "nginx:latest"},
		{Name: svcAPI, Image: "api:latest"},
	},
}

func newModel(
	t *testing.T,
	mockD *dockermocks.MockDocker,
	mockP *parsermocks.MockParser,
) dashboard.Model {
	t.Helper()

	return dashboard.New(
		context.Background(),
		testProject,
		slog.Default(),
		theme.Default(),
		config.Defaults(),
		zone.New(),
		t.TempDir(),
		100,
		50,
		mockD,
		mockP,
	)
}

// ---------------------------------------------------------------------------
// TestInit
// ---------------------------------------------------------------------------

func TestInit(t *testing.T) {
	t.Parallel()

	t.Run("returns batch with carousel, panel, and bindings", func(t *testing.T) {
		t.Parallel()

		mockD, mockP := dockermocks.NewMockDocker(t), parsermocks.NewMockParser(t)
		m := newModel(t, mockD, mockP)
		cmd := m.Init()
		require.NotNil(t, cmd)

		msg := cmd()
		batch, ok := msg.(tea.BatchMsg)
		require.True(t, ok)

		found := false

		for _, entry := range batch {
			if _, isBindings := entry().(msgs.BindingsMsg); isBindings {
				found = true

				break
			}
		}

		assert.True(t, found, "expected BindingsMsg in Init batch")
	})
}

// ---------------------------------------------------------------------------
// TestUpdate
// ---------------------------------------------------------------------------

//nolint:funlen,maintidx // long test with many table-driven cases
func TestUpdate(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func(dashboard.Model, *dockermocks.MockDocker, *parsermocks.MockParser) dashboard.Model
		// act
		msg tea.Msg
		// assert
		expectedMsg tea.Msg
		expectCmd   bool
		check       func(*testing.T, tea.Cmd)
	}

	cases := []testCase{
		// --- keyboard: quit ---
		{
			name:        "key q produces tea.QuitMsg",
			msg:         key('q'),
			expectedMsg: tea.QuitMsg{},
		},
		{
			name:        "key ctrl+c produces tea.QuitMsg",
			msg:         tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl},
			expectedMsg: tea.QuitMsg{},
		},
		// --- keyboard: settings toggle ---
		{
			name: "key , shows settings overlay",
			msg:  key(','),
		},
		// --- keyboard: toggle wrap ---
		{
			name:        "key w emits ToggleLogWrap",
			msg:         key('w'),
			expectedMsg: msgs.ToggleLogWrap{},
		},
		// --- keyboard: restart with selected service ---
		{
			name:        "key r emits ServiceRestart for selected service",
			msg:         key('r'),
			expectedMsg: msgs.ServiceRestart{ServiceName: svcWeb},
		},
		// --- keyboard: rebuild with selected service ---
		{
			name:        "key b emits ServiceRebuild for selected service",
			msg:         key('b'),
			expectedMsg: msgs.ServiceRebuild{ServiceName: svcWeb},
		},
		// --- keyboard: restart without selected service ---
		{
			name: "key r with no selected service no-ops",
			setup: func(
				m dashboard.Model, _ *dockermocks.MockDocker, _ *parsermocks.MockParser,
			) dashboard.Model {
				m, _ = m.Update(msgs.ServiceSelected{ServiceName: ""})

				return m
			},
			msg: key('r'),
		},
		// --- keyboard: rebuild without selected service ---
		{
			name: "key b with no selected service no-ops",
			setup: func(
				m dashboard.Model, _ *dockermocks.MockDocker, _ *parsermocks.MockParser,
			) dashboard.Model {
				m, _ = m.Update(msgs.ServiceSelected{ServiceName: ""})

				return m
			},
			msg: key('b'),
		},
		// --- keyboard: scroll keys ---
		{
			name: "scroll up forwarded to panel produces no command",
			msg:  key(tea.KeyUp),
		},
		{
			name: "scroll down forwarded to panel produces no command",
			msg:  key(tea.KeyDown),
		},
		{
			name: "scroll left forwarded to panel produces no command",
			msg:  key(tea.KeyLeft),
		},
		{
			name: "scroll right forwarded to panel produces no command",
			msg:  key(tea.KeyRight),
		},
		{
			name: "scroll k forwarded to panel produces no command",
			msg:  key('k'),
		},
		{
			name: "scroll j forwarded to panel produces no command",
			msg:  key('j'),
		},
		{
			name: "scroll h forwarded to panel produces no command",
			msg:  key('h'),
		},
		{
			name: "scroll l forwarded to panel produces no command",
			msg:  key('l'),
		},
		// --- service action: Stop ---
		{
			name: "ServiceStop emits DisplayStatus and forwards to docker.Stop",
			setup: func(
				m dashboard.Model, mockD *dockermocks.MockDocker, _ *parsermocks.MockParser,
			) dashboard.Model {
				mockD.EXPECT().Stop(mock.Anything, mock.Anything, mock.Anything, svcWeb).
					Return(func() tea.Msg {
						return msgs.ServiceActionCompleted{
							ServiceName: svcWeb,
							Action:      domain.ServiceActionStop,
						}
					})

				return m
			},
			msg: msgs.ServiceStop{ServiceName: svcWeb},
			check: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				assertServiceActionBatch(t, cmd, "web stopping", domain.ServiceActionStop)
			},
		},
		// --- service action: Start ---
		{
			name: "ServiceStart emits DisplayStatus and forwards to docker.Start",
			setup: func(
				m dashboard.Model, mockD *dockermocks.MockDocker, _ *parsermocks.MockParser,
			) dashboard.Model {
				mockD.EXPECT().Start(mock.Anything, mock.Anything, mock.Anything, svcWeb).
					Return(func() tea.Msg {
						return msgs.ServiceActionCompleted{
							ServiceName: svcWeb,
							Action:      domain.ServiceActionStart,
						}
					})

				return m
			},
			msg: msgs.ServiceStart{ServiceName: svcWeb},
			check: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				assertServiceActionBatch(t, cmd, "web starting", domain.ServiceActionStart)
			},
		},
		// --- service action: Restart ---
		{
			name: "ServiceRestart emits DisplayStatus and forwards to docker.Restart",
			setup: func(
				m dashboard.Model, mockD *dockermocks.MockDocker, _ *parsermocks.MockParser,
			) dashboard.Model {
				mockD.EXPECT().Restart(mock.Anything, mock.Anything, mock.Anything, svcWeb).
					Return(func() tea.Msg {
						return msgs.ServiceActionCompleted{
							ServiceName: svcWeb,
							Action:      domain.ServiceActionRestart,
						}
					})

				return m
			},
			msg: msgs.ServiceRestart{ServiceName: svcWeb},
			check: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				assertServiceActionBatch(t, cmd, "web restarting", domain.ServiceActionRestart)
			},
		},
		// --- service action: Rebuild ---
		{
			name: "ServiceRebuild emits DisplayStatus and forwards to docker.Rebuild",
			setup: func(
				m dashboard.Model, mockD *dockermocks.MockDocker, _ *parsermocks.MockParser,
			) dashboard.Model {
				mockD.EXPECT().Rebuild(mock.Anything, mock.Anything, mock.Anything, svcWeb).
					Return(func() tea.Msg {
						return msgs.ServiceActionCompleted{
							ServiceName: svcWeb,
							Action:      domain.ServiceActionRebuild,
						}
					})

				return m
			},
			msg: msgs.ServiceRebuild{ServiceName: svcWeb},
			check: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				assertServiceActionBatch(t, cmd, "web rebuilding", domain.ServiceActionRebuild)
			},
		},
		// --- service action: completed with error ---
		{
			name: "ServiceActionCompleted with error emits DisplayError",
			setup: func(
				m dashboard.Model, _ *dockermocks.MockDocker, _ *parsermocks.MockParser,
			) dashboard.Model {
				return m
			},
			msg: msgs.ServiceActionCompleted{
				ServiceName: svcWeb,
				Action:      domain.ServiceActionStop,
				Err:         assert.AnError,
			},
			check: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd)
				msg := cmd()

				if batch, ok := msg.(tea.BatchMsg); ok {
					require.Len(t, batch, 1)
					msg = batch[0]()
				}

				errMsg, ok := msg.(msgs.DisplayError)
				require.True(t, ok)
				assert.Contains(t, errMsg.Err, assert.AnError.Error())
			},
		},
		// --- service action: completed without error ---
		{
			name: "ServiceActionCompleted without error produces no status cmd",
			msg: msgs.ServiceActionCompleted{
				ServiceName: svcWeb,
				Action:      domain.ServiceActionStop,
			},
		},
		// --- file availability: file changed ---
		{
			name: "FileAvailabilityChanged with project file re-parses and rebuilds dashboard",
			setup: func(
				m dashboard.Model, _ *dockermocks.MockDocker, mockP *parsermocks.MockParser,
			) dashboard.Model {
				mockP.EXPECT().Parse(testProject.File).Return(&domain.Project{
					Name: "newproj",
					File: testProject.File,
					Services: []domain.ServiceDef{
						{Name: "new-service"},
					},
				}, nil)

				return m
			},
			msg:       msgs.FileAvailabilityChanged{Files: []string{testProject.File}},
			expectCmd: true,
		},
		// --- file availability: file removed ---
		{
			name:        "FileAvailabilityChanged without project file emits FileRemoved",
			msg:         msgs.FileAvailabilityChanged{Files: []string{"other-file.yaml"}},
			expectedMsg: msgs.FileRemoved{File: testProject.File},
		},
		// --- settings: applied ---
		{
			name: "SettingsApplied returns no command",
			msg:  msgs.SettingsApplied{Theme: "default_light", LogBufferCap: 2000},
		},
		// --- settings: visibility changed ---
		{
			name: "SettingsVisibilityChanged sets showingSettings flag",
			msg:  msgs.SettingsVisibilityChanged{Visible: true},
		},
		// --- mouse/key blocked when settings visible ---
		{
			name: "mouse events blocked when settings overlay visible",
			setup: func(
				m dashboard.Model, _ *dockermocks.MockDocker, _ *parsermocks.MockParser,
			) dashboard.Model {
				m, _ = m.Update(msgs.SettingsVisibilityChanged{Visible: true})

				return m
			},
			msg: tea.MouseClickMsg{},
		},
		{
			name: "key events blocked when settings overlay visible",
			setup: func(
				m dashboard.Model, _ *dockermocks.MockDocker, _ *parsermocks.MockParser,
			) dashboard.Model {
				m, _ = m.Update(msgs.SettingsVisibilityChanged{Visible: true})

				return m
			},
			msg:         key('q'),
			expectedMsg: msgs.SettingsVisibilityChanged{Visible: false},
		},
		// --- window resize ---
		{
			name:      "WindowSizeMsg stores dimensions",
			msg:       tea.WindowSizeMsg{Width: 200, Height: 100},
			expectCmd: true,
		},
		// --- theme changed ---
		{
			name:      "theme.Changed updates theme pointer",
			msg:       theme.Changed{Theme: theme.DefaultLight()},
			expectCmd: true,
		},
		// --- state poll tick ---
		{
			name: "StatePollTick triggers docker.Ps and forwards to panel",
			setup: func(
				m dashboard.Model, mockD *dockermocks.MockDocker, _ *parsermocks.MockParser,
			) dashboard.Model {
				mockD.EXPECT().Ps(mock.Anything, mock.Anything, mock.Anything).Maybe().
					Return(tea.Cmd(func() tea.Msg { return msgs.ServicesPolled{} }))

				return m
			},
			msg:       msgs.StatePollTick{},
			expectCmd: true,
		},
		// --- services polled ---
		{
			name: "ServicesPolled stores runtime data",
			msg: msgs.ServicesPolled{
				Runtimes: map[string]*domain.ServiceRuntimeData{
					svcWeb: {State: domain.ServiceStateRunning},
				},
			},
			expectCmd: true,
		},
		{
			name: "ServicesPolled error does not update runtime data",
			msg: msgs.ServicesPolled{
				Err: assert.AnError,
			},
		},
		// --- ServiceSelected ---
		{
			name:      "ServiceSelected stores selected name",
			msg:       msgs.ServiceSelected{ServiceName: "api"},
			expectCmd: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockD, mockP := dockermocks.NewMockDocker(t), parsermocks.NewMockParser(t)

			m := newModel(t, mockD, mockP)
			if tc.setup != nil {
				m = tc.setup(m, mockD, mockP)
			}

			_, cmd := m.Update(tc.msg)

			switch {
			case tc.check != nil:
				tc.check(t, cmd)
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

// ---------------------------------------------------------------------------
// TestView
// ---------------------------------------------------------------------------

func TestView(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func(dashboard.Model) dashboard.Model
		// assert
		expectedResult string
	}

	cases := []testCase{
		{
			name:           "normal dashboard renders carousel and service content",
			expectedResult: "web",
		},
		{
			name: "settings overlay compositor visible when showingSettings",
			setup: func(m dashboard.Model) dashboard.Model {
				m, _ = m.Update(msgs.SettingsVisibilityChanged{Visible: true})

				return m
			},
			expectedResult: "Settings",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockD, mockP := dockermocks.NewMockDocker(t), parsermocks.NewMockParser(t)
			m := newModel(t, mockD, mockP)

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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func key(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r}
}

func assertServiceActionBatch(
	t *testing.T,
	cmd tea.Cmd,
	expectedStatus string,
	expectedAction domain.ServiceAction,
) {
	t.Helper()
	require.NotNil(t, cmd)
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	require.True(t, ok)
	require.Len(t, batch, 2)

	statusMsg, ok := batch[0]().(msgs.DisplayStatus)
	require.True(t, ok)
	assert.Equal(t, expectedStatus, statusMsg.Msg)

	completedMsg, ok := batch[1]().(msgs.ServiceActionCompleted)
	require.True(t, ok)
	assert.Equal(t, svcWeb, completedMsg.ServiceName)
	assert.Equal(t, expectedAction, completedMsg.Action)
}
