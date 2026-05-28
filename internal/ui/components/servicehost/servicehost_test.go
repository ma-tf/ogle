package servicehost_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	logsmocks "github.com/ma-tf/ogle/internal/services/docker/logs/mocks"
	"github.com/ma-tf/ogle/internal/ui/components/servicehost"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	testProject = "testproj"
	svcName     = "web"
)

var svcDef = domain.ServiceDef{Name: svcName} //nolint:gochecknoglobals // shared test fixture

func newModel(t *testing.T) (servicehost.Model, *logsmocks.MockStreamer) {
	t.Helper()

	s := logsmocks.NewMockStreamer(t)
	s.EXPECT().Lines().Return((<-chan string)(make(chan string)))

	return servicehost.New(theme.Default(), svcDef, testProject, 120, 100, 100, s), s
}

func TestUpdate_Lifecycle(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name  string
		setup func(*testing.T) servicehost.Model
		msg   tea.Msg
		check func(*testing.T, servicehost.Model)
	}

	cases := []testCase{
		{
			name: "ServiceSelected matching name sets selected",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, _ := newModel(t)

				return m
			},
			msg: msgs.ServiceSelected{ServiceName: svcName},
			check: func(t *testing.T, m servicehost.Model) {
				t.Helper()
				assert.Contains(t, m.View().Content, "╭")
			},
		},

		{
			name: "ServiceSelected non-matching name clears selected",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, _ := newModel(t)
				m, _ = m.Update(msgs.ServiceSelected{ServiceName: svcName})

				return m
			},
			msg: msgs.ServiceSelected{ServiceName: "db"},
			check: func(t *testing.T, m servicehost.Model) {
				t.Helper()
				assert.Empty(t, m.View().Content)
			},
		},

		{
			name: "DaemonConnected is no-op",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, _ := newModel(t)

				return m
			},
			msg: msgs.DaemonConnected{},
		},

		{
			name: "KeyPressMsg when not selected is no-op",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, _ := newModel(t)

				return m
			},
			msg: tea.KeyPressMsg{},
		},

		{
			name: "theme.Changed updates stored theme",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, _ := newModel(t)

				return m
			},
			msg: theme.Changed{Theme: theme.DefaultLight()},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := tc.setup(t)
			m, cmd := m.Update(tc.msg)
			require.Nil(t, cmd)

			if tc.check != nil {
				tc.check(t, m)
			}
		})
	}
}

//nolint:funlen
func TestUpdate_LogStream(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name        string
		setup       func(*testing.T) servicehost.Model
		msg         tea.Msg
		expectedMsg tea.Msg
		expectCmd   bool
		check       func(*testing.T, servicehost.Model)
	}

	cases := []testCase{
		{
			name: "ServicesPolled with running container starts streamer",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, s := newModel(t)
				s.EXPECT().Start(mock.Anything, testProject+"-"+svcName+"-1").Return()
				s.EXPECT().Next().Return(func() tea.Msg {
					return msgs.LogLinesAvailable{}
				})

				return m
			},
			msg: msgs.ServicesPolled{
				Runtimes: map[string]*domain.ServiceRuntimeData{
					svcName: {State: domain.ServiceStateRunning},
				},
			},
			expectedMsg: msgs.LogLinesAvailable{},
		},

		{
			name: "ServicesPolled without runtime data does not start streamer",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, _ := newModel(t)

				return m
			},
			msg: msgs.ServicesPolled{
				Runtimes: map[string]*domain.ServiceRuntimeData{},
			},
			expectCmd: false,
		},

		{
			name: "ServicesPolled with non-running state does not start streamer",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, _ := newModel(t)

				return m
			},
			msg: msgs.ServicesPolled{
				Runtimes: map[string]*domain.ServiceRuntimeData{
					svcName: {State: domain.ServiceStateExited},
				},
			},
			expectCmd: false,
		},

		{
			name: "ServicesPolled with running container closes streamer on state change",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, s := newModel(t)

				s.EXPECT().Start(mock.Anything, testProject+"-"+svcName+"-1").Return()
				s.EXPECT().Next().Return(func() tea.Msg { return nil })

				m, _ = m.Update(msgs.ServicesPolled{
					Runtimes: map[string]*domain.ServiceRuntimeData{
						svcName: {State: domain.ServiceStateRunning},
					},
				})

				s.EXPECT().Close().Return()

				return m
			},
			msg: msgs.ServicesPolled{
				Runtimes: map[string]*domain.ServiceRuntimeData{
					svcName: {State: domain.ServiceStateExited},
				},
			},
			expectCmd: false,
		},

		{
			name: "LogLinesAvailable emits streamer.Next and resets retryCount",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, s := newModel(t)
				s.EXPECT().Next().Return(func() tea.Msg {
					return msgs.LogLinesAvailable{}
				})

				return m
			},
			msg:         msgs.LogLinesAvailable{},
			expectedMsg: msgs.LogLinesAvailable{},
		},

		{
			name: "LogStreamError closes streamer and schedules retry",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, s := newModel(t)
				s.EXPECT().Close().Return()

				return m
			},
			msg:       msgs.LogStreamError{Err: nil, ServiceName: svcName},
			expectCmd: true,
		},

		{
			name: "LogStreamContainerNotFound closes streamer without retry",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, s := newModel(t)
				s.EXPECT().Close().Return()

				return m
			},
			msg:       msgs.LogStreamContainerNotFound{ServiceName: svcName},
			expectCmd: false,
		},

		{
			name: "LogStreamRetryTick starts streamer when stopped",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, s := newModel(t)

				s.EXPECT().Close().Return()

				m, _ = m.Update(msgs.LogStreamError{Err: nil, ServiceName: svcName})

				s.EXPECT().Start(mock.Anything, testProject+"-"+svcName+"-1").Return()
				s.EXPECT().Next().Return(func() tea.Msg {
					return msgs.LogLinesAvailable{}
				})

				return m
			},
			msg:         msgs.LogStreamRetryTick{},
			expectedMsg: msgs.LogLinesAvailable{},
		},

		{
			name: "LogStreamRetryTick is no-op when streamer is already started",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, s := newModel(t)

				s.EXPECT().Start(mock.Anything, testProject+"-"+svcName+"-1").Return()
				s.EXPECT().Next().Return(func() tea.Msg { return nil })

				m, _ = m.Update(msgs.ServicesPolled{
					Runtimes: map[string]*domain.ServiceRuntimeData{
						svcName: {State: domain.ServiceStateRunning},
					},
				})

				return m
			},
			msg:       msgs.LogStreamRetryTick{},
			expectCmd: false,
		},

		{
			name: "ServicesPolled error is no-op",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, _ := newModel(t)

				return m
			},
			msg: msgs.ServicesPolled{
				Err: assert.AnError,
			},
			expectCmd: false,
		},

		{
			name: "ToggleLogWrap emits LogWrapStatus with service name injected",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, _ := newModel(t)

				return m
			},
			msg:         msgs.ToggleLogWrap{},
			expectedMsg: msgs.LogWrapStatus{On: true, Overflow: false, ServiceName: svcName},
		},

		{
			name: "ToggleLogWrap off emits LogWrapStatus with service name",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, _ := newModel(t)
				m, _ = m.Update(msgs.ToggleLogWrap{})

				return m
			},
			msg:         msgs.ToggleLogWrap{},
			expectedMsg: msgs.LogWrapStatus{On: false, Overflow: false, ServiceName: svcName},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := tc.setup(t)
			m, cmd := m.Update(tc.msg)

			switch {
			case tc.expectedMsg != nil:
				require.NotNil(t, cmd)
				require.Equal(t, tc.expectedMsg, cmd())
			case tc.expectCmd:
				require.NotNil(t, cmd)
			default:
				require.Nil(t, cmd)
			}

			if tc.check != nil {
				tc.check(t, m)
			}
		})
	}
}

func TestUpdate_LogStreamErrorRecoveryCycle(t *testing.T) {
	t.Parallel()

	t.Run("error to retry to restart cycle via mock", func(t *testing.T) {
		t.Parallel()

		m, s := newModel(t)

		// Step 1: ServicesPolled with running container starts the streamer
		s.EXPECT().Start(mock.Anything, testProject+"-"+svcName+"-1").Return().Once()
		s.EXPECT().Next().Return(func() tea.Msg {
			return msgs.LogLinesAvailable{}
		}).Once()

		m, cmd := m.Update(msgs.ServicesPolled{
			Runtimes: map[string]*domain.ServiceRuntimeData{
				svcName: {State: domain.ServiceStateRunning},
			},
		})
		require.NotNil(t, cmd)
		require.Equal(t, msgs.LogLinesAvailable{}, cmd())

		// Step 2: LogStreamError → Close + retry tick
		s.EXPECT().Close().Return().Once()

		m, cmd = m.Update(msgs.LogStreamError{Err: nil, ServiceName: svcName})
		require.NotNil(t, cmd) // tea.Tick cmd — non-deterministic, skip calling it

		// Step 3: LogStreamRetryTick → Start + Next
		s.EXPECT().Start(mock.Anything, testProject+"-"+svcName+"-1").Return().Once()
		s.EXPECT().Next().Return(func() tea.Msg {
			return msgs.LogLinesAvailable{}
		}).Once()

		m, cmd = m.Update(msgs.LogStreamRetryTick{})
		require.NotNil(t, cmd)
		result := cmd()
		require.Equal(t, msgs.LogLinesAvailable{}, result)

		// Streamer started again — verify LogLinesAvailable re-subscribes and resets retryCount
		s.EXPECT().Next().Return(func() tea.Msg {
			return msgs.LogLinesAvailable{}
		}).Once()

		_, cmd = m.Update(msgs.LogLinesAvailable{})
		require.NotNil(t, cmd)
		require.Equal(t, msgs.LogLinesAvailable{}, cmd())
	})
}

func TestUpdate_ContainerStopsThenStarts(t *testing.T) {
	t.Parallel()

	t.Run("closes streamer on container stop, restarts on container start", func(t *testing.T) {
		t.Parallel()

		m, s := newModel(t)

		// Start streamer
		s.EXPECT().Start(mock.Anything, testProject+"-"+svcName+"-1").Return().Once()
		s.EXPECT().Next().Return(func() tea.Msg { return nil }).Once()

		runningState := msgs.ServicesPolled{
			Runtimes: map[string]*domain.ServiceRuntimeData{
				svcName: {State: domain.ServiceStateRunning},
			},
		}
		m, cmd := m.Update(runningState)
		require.NotNil(t, cmd)

		// Container stops
		s.EXPECT().Close().Return().Once()

		exitedState := msgs.ServicesPolled{
			Runtimes: map[string]*domain.ServiceRuntimeData{
				svcName: {State: domain.ServiceStateExited},
			},
		}
		m, cmd = m.Update(exitedState)
		require.Nil(t, cmd)

		// Container starts again
		s.EXPECT().Start(mock.Anything, testProject+"-"+svcName+"-1").Return().Once()
		s.EXPECT().Next().Return(func() tea.Msg { return nil }).Once()

		runningState2 := msgs.ServicesPolled{
			Runtimes: map[string]*domain.ServiceRuntimeData{
				svcName: {State: domain.ServiceStateRunning},
			},
		}
		_, cmd = m.Update(runningState2)
		require.NotNil(t, cmd)
	})
}

func TestUpdate_ClearLogBuffer(t *testing.T) {
	t.Parallel()

	t.Run("ClearLogBuffer with matching name clears log pane lines", func(t *testing.T) {
		t.Parallel()

		ch := make(chan string, 10)
		ch <- "visible line"

		ch <- "another line"

		s := logsmocks.NewMockStreamer(t)
		s.EXPECT().Lines().Return((<-chan string)(ch))
		s.EXPECT().Next().Return(func() tea.Msg { return nil })

		m := servicehost.New(theme.Default(), svcDef, testProject, 120, 100, 100, s)
		m, _ = m.Update(msgs.ServiceSelected{ServiceName: svcName})
		m, _ = m.Update(msgs.LogLinesAvailable{})

		assert.Contains(t, m.View().Content, "visible line")
		assert.Contains(t, m.View().Content, "another line")

		m, cmd := m.Update(msgs.ClearLogBuffer{ServiceName: svcName})
		require.Nil(t, cmd)
		assert.NotContains(t, m.View().Content, "visible line")
		assert.NotContains(t, m.View().Content, "another line")
	})

	t.Run("ClearLogBuffer with non-matching name preserves log pane lines", func(t *testing.T) {
		t.Parallel()

		ch := make(chan string, 10)
		ch <- "preserved line"

		s := logsmocks.NewMockStreamer(t)
		s.EXPECT().Lines().Return((<-chan string)(ch))
		s.EXPECT().Next().Return(func() tea.Msg { return nil })

		m := servicehost.New(theme.Default(), svcDef, testProject, 120, 100, 100, s)
		m, _ = m.Update(msgs.ServiceSelected{ServiceName: svcName})
		m, _ = m.Update(msgs.LogLinesAvailable{})

		assert.Contains(t, m.View().Content, "preserved line")

		m, cmd := m.Update(msgs.ClearLogBuffer{ServiceName: "other-service"})
		require.Nil(t, cmd)
		assert.Contains(t, m.View().Content, "preserved line")
	})
}

func TestView(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func(*testing.T) servicehost.Model

		// assert
		expectedResult string
	}

	cases := []testCase{
		{
			name: "empty when not selected",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, _ := newModel(t)

				return m
			},
			expectedResult: "",
		},

		{
			name: "log pane when selected",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, _ := newModel(t)
				m, _ = m.Update(msgs.ServiceSelected{ServiceName: svcName})

				return m
			},
			expectedResult: "╭",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := tc.setup(t)

			if tc.expectedResult == "" {
				assert.Empty(t, m.View().Content)
			} else {
				assert.Contains(t, m.View().Content, tc.expectedResult)
			}
		})
	}
}
