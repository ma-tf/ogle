package dashboard_test

import (
	"context"
	"testing"

	bubbleskey "charm.land/bubbles/v2/key"
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

type updateTestCase struct {
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

//nolint:funlen,maintidx // table-driven test cases with inline setup/check closures
func buildUpdateTestCases() []updateTestCase {
	return []updateTestCase{
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
		// --- keyboard: clear log buffer ---
		{
			name:        "key c emits ClearLogBuffer for selected service",
			msg:         key('c'),
			expectedMsg: msgs.ClearLogBuffer{ServiceName: svcWeb},
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
		// --- keyboard: clear log without selected service ---
		{
			name: "key c with no selected service no-ops",
			setup: func(
				m dashboard.Model, _ *dockermocks.MockDocker, _ *parsermocks.MockParser,
			) dashboard.Model {
				m, _ = m.Update(msgs.ServiceSelected{ServiceName: ""})

				return m
			},
			msg: key('c'),
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
			expectCmd: false,
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
			name: "ServiceSelected emits TopbarContext with service name",
			msg:  msgs.ServiceSelected{ServiceName: svcAPI},
			check: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd)

				tcMsg, found := findInBatch[msgs.TopbarContext](cmd())
				require.True(t, found, "expected TopbarContext in batch")
				assert.Equal(t, svcAPI, tcMsg.Service)
				assert.Equal(t, "dashboard", tcMsg.Phase)
				assert.Equal(t, "compose.yaml", tcMsg.File)
			},
		},
		{
			name: "ServiceSelected emits ReportWrapStatus for selected service",
			msg:  msgs.ServiceSelected{ServiceName: svcWeb},
			check: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd)

				rws, found := findInBatch[msgs.ReportWrapStatus](cmd())
				require.True(t, found, "expected ReportWrapStatus in batch")
				assert.Equal(t, svcWeb, rws.ServiceName)
			},
		},
		// --- labels: ServicesPolled triggers Inspect for selected service ---
		{
			name: "ServicesPolled triggers Inspect for selected service",
			setup: func(
				m dashboard.Model, mockD *dockermocks.MockDocker, _ *parsermocks.MockParser,
			) dashboard.Model {
				mockD.EXPECT().Inspect(mock.Anything, "abc123").
					Return(tea.Cmd(func() tea.Msg {
						return msgs.ContainerLabelsPolled{
							Labels: map[string]string{"ogle.foo": "bar"},
						}
					}))

				return m
			},
			msg: msgs.ServicesPolled{
				Runtimes: map[string]*domain.ServiceRuntimeData{
					svcWeb: {ContainerID: "abc123", State: domain.ServiceStateRunning},
				},
			},
			check: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd)
				msg := cmd()
				batch, ok := msg.(tea.BatchMsg)
				require.True(t, ok)

				found := false

				for _, entry := range batch {
					if entry == nil {
						continue
					}

					if polled, isPolled := entry().(msgs.ContainerLabelsPolled); isPolled {
						require.Equal(t, map[string]string{"ogle.foo": "bar"}, polled.Labels)
						require.NoError(t, polled.Err)

						found = true

						break
					}
				}

				assert.True(t, found, "expected ContainerLabelsPolled in batch")
			},
		},
		// --- FrameHeight ---
		{
			name: "FrameHeight updates layout",
			msg:  msgs.FrameHeight{Height: 5},
			check: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()

				assert.Nil(t, cmd, "FrameHeight should not produce a command")
			},
		},
		// --- labels: ServiceSelected without runtime does not trigger Inspect ---
		{
			name: "ServiceSelected without runtime data does not trigger Inspect",
			setup: func(
				m dashboard.Model, _ *dockermocks.MockDocker, _ *parsermocks.MockParser,
			) dashboard.Model {
				return m
			},
			msg: msgs.ServiceSelected{ServiceName: svcAPI},
			check: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd)
				msg := cmd()
				batch, isBatch := msg.(tea.BatchMsg)
				require.True(t, isBatch)

				for _, entry := range batch {
					if entry == nil {
						continue
					}

					if _, isPolled := entry().(msgs.ContainerLabelsPolled); isPolled {
						t.Error(
							"unexpected ContainerLabelsPolled" +
								" - Inspect should not have been called",
						)
					}
				}
			},
		},
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	for _, tc := range buildUpdateTestCases() {
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
		expectedResult     string
		expectedNotPresent string
	}

	cases := []testCase{
		{
			name:           "normal dashboard renders carousel and service content",
			expectedResult: svcWeb,
		},
		{
			name: "settings overlay compositor visible when showingSettings",
			setup: func(m dashboard.Model) dashboard.Model {
				m, _ = m.Update(msgs.SettingsVisibilityChanged{Visible: true})

				return m
			},
			expectedResult: "Settings",
		},
		{
			name:           "labels accordion header visible in view",
			expectedResult: "▶ Labels",
		},
		{
			name: "FrameHeight updates usable height in view",
			setup: func(m dashboard.Model) dashboard.Model {
				m, _ = m.Update(msgs.FrameHeight{Height: 10})

				return m
			},
			expectedResult: svcWeb,
		},
		{
			name: "service list hidden when terminal narrower than SidebarMinTermWidth",
			setup: func(m dashboard.Model) dashboard.Model {
				m, _ = m.Update(tea.WindowSizeMsg{Width: 70, Height: 50})

				return m
			},
			expectedNotPresent: "web",
		},
		{
			name: "service list visible when terminal at SidebarMinTermWidth",
			setup: func(m dashboard.Model) dashboard.Model {
				m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 50})

				return m
			},
			expectedResult: "web",
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

			if tc.expectedNotPresent != "" {
				assert.NotContains(t, m.View().Content, tc.expectedNotPresent)
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
// TestKeymap
// ---------------------------------------------------------------------------

func extractBindingsMsg(t *testing.T, msg tea.Msg) (msgs.BindingsMsg, bool) {
	t.Helper()

	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return msgs.BindingsMsg{}, false
	}

	for _, entry := range batch {
		if bm, bmOk := entry().(msgs.BindingsMsg); bmOk {
			return bm, true
		}
	}

	return msgs.BindingsMsg{}, false
}

func TestKeymap_ImplementsPinnedHelp(t *testing.T) {
	t.Parallel()

	mockD, mockP := dockermocks.NewMockDocker(t), parsermocks.NewMockParser(t)
	m := newModel(t, mockD, mockP)
	cmd := m.Init()
	require.NotNil(t, cmd)

	bindMsg, ok := extractBindingsMsg(t, cmd())
	require.True(t, ok)

	km, ok := bindMsg.Keymap.(interface {
		ShortHelp() []bubbleskey.Binding
		FullHelp() [][]bubbleskey.Binding
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

func TestKeymap_ShortHelp(t *testing.T) {
	t.Parallel()

	mockD, mockP := dockermocks.NewMockDocker(t), parsermocks.NewMockParser(t)
	m := newModel(t, mockD, mockP)
	cmd := m.Init()
	require.NotNil(t, cmd)

	bindMsg, ok := extractBindingsMsg(t, cmd())
	require.True(t, ok)

	bindings := bindMsg.Keymap.ShortHelp()
	require.Len(t, bindings, 6)
	assert.Equal(t, "tab", bindings[0].Help().Key)
	assert.Equal(t, "shift+tab", bindings[1].Help().Key)
	assert.Equal(t, "enter", bindings[2].Help().Key)
	assert.Equal(t, "r", bindings[3].Help().Key)
	assert.Equal(t, "b", bindings[4].Help().Key)
	assert.Equal(t, "w", bindings[5].Help().Key)
}

func TestKeymap_FullHelp(t *testing.T) {
	t.Parallel()

	mockD, mockP := dockermocks.NewMockDocker(t), parsermocks.NewMockParser(t)
	m := newModel(t, mockD, mockP)
	cmd := m.Init()
	require.NotNil(t, cmd)

	bindMsg, ok := extractBindingsMsg(t, cmd())
	require.True(t, ok)

	fullHelp := bindMsg.Keymap.FullHelp()
	require.Len(t, fullHelp, 4)

	col1 := fullHelp[0]
	foundShiftTab := false

	for _, b := range col1 {
		if b.Help().Key == "shift+tab" {
			assert.Equal(t, "focus previous", b.Help().Desc)

			foundShiftTab = true

			break
		}
	}

	assert.True(t, foundShiftTab, "expected shift+tab binding in column 1")

	col4 := fullHelp[3]
	foundF1 := false

	for _, b := range col4 {
		if b.Help().Key == "f1" {
			assert.Equal(t, "about", b.Help().Desc)

			foundF1 = true

			break
		}
	}

	assert.True(t, foundF1, "expected f1 binding in column 4")
}

// ---------------------------------------------------------------------------
// TestView_AccordionDynamicHeight
// ---------------------------------------------------------------------------

func TestView_AccordionDynamicHeight(t *testing.T) {
	t.Parallel()

	t.Run("accordion visible with default vertical space", func(t *testing.T) {
		t.Parallel()

		mockD, mockP := dockermocks.NewMockDocker(t), parsermocks.NewMockParser(t)
		m := newModel(t, mockD, mockP)

		assert.Contains(t, m.View().Content, "Service Details")
	})

	t.Run("accordion hidden when insufficient vertical space", func(t *testing.T) {
		t.Parallel()

		mockD, mockP := dockermocks.NewMockDocker(t), parsermocks.NewMockParser(t)
		m := newModel(t, mockD, mockP)
		m, _ = m.Update(msgs.FrameHeight{Height: 100})

		assert.NotContains(t, m.View().Content, "Service Details")
	})

	t.Run("accordion uses natural height for guard condition", func(t *testing.T) {
		t.Parallel()

		mockD, mockP := dockermocks.NewMockDocker(t), parsermocks.NewMockParser(t)
		m := newModel(t, mockD, mockP)

		// usableH = h - frameHeight = 50 - 33 = 17
		// Carousel at w=100 renders approximately 10 lines
		// Accordion (expanded with Image) renders at 6 lines
		// With natural height: 10 + 6 = 16 <= 17 → accordion fits
		// With old fixed 8: 10 + 8 = 18 > 17 → accordion would not fit
		m, _ = m.Update(msgs.FrameHeight{Height: 33})

		assert.Contains(t, m.View().Content, "Service Details")
	})
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

func findInBatch[T any](msg tea.Msg) (T, bool) {
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		var zero T

		return zero, false
	}

	for _, entry := range batch {
		if entry == nil {
			continue
		}

		result := entry()

		if found, isType := result.(T); isType {
			return found, true
		}

		if nested, isNested := result.(tea.BatchMsg); isNested {
			if found, nestedOk := findInBatch[T](nested); nestedOk {
				return found, true
			}
		}
	}

	var zero T

	return zero, false
}
