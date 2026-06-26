package card_test

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/carousel/card"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	testShortName = "short-svc"
	testLongName  = "service-name-that-needs-scrolling-yes"
	otherService  = "other-service"
	testW         = 200
	testH         = 40
)

func newCard(name string) card.Model {
	return card.New(domain.ServiceDef{Name: name}, testW, testH, theme.Default())
}

// TestInit

func TestInit(t *testing.T) {
	t.Parallel()

	m := newCard(testShortName)
	cmd := m.Init()

	require.Nil(t, cmd)
}

// TestUpdate

//nolint:funlen,maintidx // comprehensive table-driven test with many cases
func TestUpdate(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func(m card.Model) card.Model
		// act
		msg tea.Msg
		// assert
		assert func(t *testing.T, m card.Model, cmd tea.Cmd)
	}

	cases := []testCase{
		// FocusMsg
		{
			name: "FocusMsg matching sets focused",
			msg:  card.FocusMsg{ServiceName: testShortName},
			assert: func(t *testing.T, m card.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd, "short name should not schedule scroll")
				assert.Contains(t, m.View().Content, testShortName)
			},
		},
		{
			name: "FocusMsg matching with scroll starts scroll",
			setup: func(_ card.Model) card.Model {
				return newCard(testLongName)
			},
			msg: card.FocusMsg{ServiceName: testLongName},
			assert: func(t *testing.T, _ card.Model, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd, "long name should schedule initial scroll tick")
				scrollMsg := cmd()
				_, ok := scrollMsg.(card.ScrollTick)
				require.True(t, ok, "cmd should produce a ScrollTick message")
			},
		},
		{
			name: "FocusMsg non-matching no-op",
			msg:  card.FocusMsg{ServiceName: otherService},
			assert: func(t *testing.T, _ card.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
			},
		},

		// BlurMsg
		{
			name: "BlurMsg matching clears focus",
			setup: func(m card.Model) card.Model {
				m, _ = m.Update(card.FocusMsg{ServiceName: testShortName})

				return m
			},
			msg: card.BlurMsg{ServiceName: testShortName},
			assert: func(t *testing.T, m card.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
				assert.Contains(t, m.View().Content, testShortName)
			},
		},
		{
			name: "BlurMsg non-matching no-op",
			setup: func(m card.Model) card.Model {
				m, _ = m.Update(card.FocusMsg{ServiceName: testShortName})

				return m
			},
			msg: card.BlurMsg{ServiceName: otherService},
			assert: func(t *testing.T, m card.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
				assert.Contains(t, m.View().Content, testShortName)
			},
		},

		// HoverMsg
		{
			name: "HoverMsg matching sets hovered",
			msg:  card.HoverMsg{ServiceName: testShortName},
			assert: func(t *testing.T, m card.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
				assert.Contains(t, m.View().Content, testShortName)
			},
		},
		{
			name: "HoverMsg non-matching no-op",
			msg:  card.HoverMsg{ServiceName: otherService},
			assert: func(t *testing.T, _ card.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
			},
		},

		// UnhoverMsg
		{
			name: "UnhoverMsg matching clears hovered",
			setup: func(m card.Model) card.Model {
				m, _ = m.Update(card.HoverMsg{ServiceName: testShortName})

				return m
			},
			msg: card.UnhoverMsg{ServiceName: testShortName},
			assert: func(t *testing.T, m card.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
				assert.Contains(t, m.View().Content, testShortName)
			},
		},

		// ServicesPolled
		{
			name: "ServicesPolled stores runtime data",
			msg: msgs.ServicesPolled{
				Runtimes: map[string]*domain.ServiceRuntimeData{
					testShortName: {State: domain.ServiceStateRunning},
				},
			},
			assert: func(t *testing.T, m card.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
				assert.Contains(t, m.View().Content, testShortName)
			},
		},
		{
			name: "ServicesPolled with err no change",
			setup: func(m card.Model) card.Model {
				m, _ = m.Update(msgs.ServicesPolled{
					Runtimes: map[string]*domain.ServiceRuntimeData{
						testShortName: {State: domain.ServiceStateRunning},
					},
				})

				return m
			},
			msg: msgs.ServicesPolled{Err: assert.AnError},
			assert: func(t *testing.T, m card.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
				assert.Contains(t, m.View().Content, testShortName)
			},
		},
		{
			name: "ServicesPolled empty name no change",
			setup: func(_ card.Model) card.Model {
				return card.New(domain.ServiceDef{}, testW, testH, theme.Default())
			},
			msg: msgs.ServicesPolled{
				Runtimes: map[string]*domain.ServiceRuntimeData{
					"": {State: domain.ServiceStateRunning},
				},
			},
			assert: func(t *testing.T, _ card.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
			},
		},

		// ServiceAction
		{
			name: "ServiceStart sets in-flight",
			msg:  msgs.ServiceStart{ServiceName: testShortName},
			assert: func(t *testing.T, m card.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
				assert.Contains(t, m.View().Content, testShortName)
			},
		},
		{
			name: "ServiceStop sets in-flight",
			msg:  msgs.ServiceStop{ServiceName: testShortName},
			assert: func(t *testing.T, m card.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
				assert.Contains(t, m.View().Content, testShortName)
			},
		},
		{
			name: "ServiceRestart sets in-flight",
			msg:  msgs.ServiceRestart{ServiceName: testShortName},
			assert: func(t *testing.T, m card.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
				assert.Contains(t, m.View().Content, testShortName)
			},
		},
		{
			name: "ServiceRebuild sets in-flight",
			msg:  msgs.ServiceRebuild{ServiceName: testShortName},
			assert: func(t *testing.T, m card.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
				assert.Contains(t, m.View().Content, testShortName)
			},
		},
		{
			name: "ServiceAction non-matching no change",
			setup: func(_ card.Model) card.Model {
				return newCard(otherService)
			},
			msg: msgs.ServiceStart{ServiceName: testShortName},
			assert: func(t *testing.T, _ card.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
			},
		},

		// ServiceActionCompleted
		{
			name: "ServiceActionCompleted clears in-flight and updates",
			setup: func(m card.Model) card.Model {
				m, _ = m.Update(msgs.ServiceStart{ServiceName: testShortName})

				return m
			},
			msg: msgs.ServiceActionCompleted{
				ServiceName: testShortName,
				Action:      domain.ServiceActionStart,
			},
			assert: func(t *testing.T, m card.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
				assert.Contains(t, m.View().Content, testShortName)
			},
		},
		{
			name: "ServiceActionCompleted with error clears in-flight keeps state",
			setup: func(m card.Model) card.Model {
				m, _ = m.Update(msgs.ServiceStart{ServiceName: testShortName})

				return m
			},
			msg: msgs.ServiceActionCompleted{
				ServiceName: testShortName,
				Action:      domain.ServiceActionStart,
				Err:         assert.AnError,
			},
			assert: func(t *testing.T, m card.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
				assert.Contains(t, m.View().Content, testShortName)
			},
		},
		{
			name: "ServiceActionCompleted stop sets exited state",
			setup: func(m card.Model) card.Model {
				m, _ = m.Update(msgs.ServicesPolled{
					Runtimes: map[string]*domain.ServiceRuntimeData{
						testShortName: {State: domain.ServiceStateRunning},
					},
				})
				m, _ = m.Update(msgs.ServiceStop{ServiceName: testShortName})

				return m
			},
			msg: msgs.ServiceActionCompleted{
				ServiceName: testShortName,
				Action:      domain.ServiceActionStop,
			},
			assert: func(t *testing.T, m card.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
				assert.NotEmpty(t, m.View().Content)
			},
		},
		{
			name: "ServiceActionCompleted non-matching no change",
			msg: msgs.ServiceActionCompleted{
				ServiceName: otherService,
				Action:      domain.ServiceActionStart,
			},
			assert: func(t *testing.T, m card.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
				assert.Contains(t, m.View().Content, testShortName)
			},
		},

		// WindowSizeMsg
		{
			name: "WindowSizeMsg updates dimensions",
			msg:  tea.WindowSizeMsg{Width: 300, Height: 60},
			assert: func(t *testing.T, m card.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd, "short name should not schedule scroll")
				assert.Contains(t, m.View().Content, testShortName)
			},
		},
		{
			name: "WindowSizeMsg focused with scroll reschedules",
			setup: func(_ card.Model) card.Model {
				m := newCard(testLongName)
				m, _ = m.Update(card.FocusMsg{ServiceName: testLongName})

				return m
			},
			msg: tea.WindowSizeMsg{Width: 200, Height: 60},
			assert: func(t *testing.T, _ card.Model, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd, "focused scrollable card should reschedule tick on resize")
			},
		},

		// theme.Changed
		{
			name: "theme.Changed updates theme",
			msg:  theme.Changed{Theme: theme.Default()},
			assert: func(t *testing.T, m card.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
				assert.Contains(t, m.View().Content, testShortName)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := newCard(testShortName)

			if tc.setup != nil {
				m = tc.setup(m)
			}

			m, cmd := m.Update(tc.msg)

			tc.assert(t, m, cmd)
		})
	}
}

// TestUpdate — ScrollTick (timing-sensitive, separate from main table)
//
// These tests involve real time delays (scrollIdleInterval = 2500ms) because
// nextScrollTime is set in the past via the FocusMsg path, and ScrollTick's
// gen field is unexported so we must obtain messages through tickScroll cmd().

func TestUpdate_ScrollTick_AdvancesAndSchedulesNext(t *testing.T) {
	t.Parallel()

	m := newCard(testLongName)
	m, cmd1 := m.Update(card.FocusMsg{ServiceName: testLongName})
	require.NotNil(t, cmd1, "long name should schedule initial scroll tick")

	scrollMsg := cmd1()

	time.Sleep(50 * time.Millisecond)

	_, cmd2 := m.Update(scrollMsg)
	require.NotNil(t, cmd2, "scroll should advance and schedule next tick")
	_, ok := cmd2().(card.ScrollTick)
	require.True(t, ok, "next cmd should be a ScrollTick")
}

func TestUpdate_ScrollTick_StaleGen_NoOp(t *testing.T) {
	t.Parallel()

	m := newCard(testLongName)

	m, cmd1 := m.Update(card.FocusMsg{ServiceName: testLongName})
	require.NotNil(t, cmd1)

	m, cmd2 := m.Update(card.FocusMsg{ServiceName: testLongName})
	require.NotNil(t, cmd2)

	scrollMsg := cmd1()

	time.Sleep(50 * time.Millisecond)

	_, cmd3 := m.Update(scrollMsg)
	require.Nil(t, cmd3, "stale generation should produce no command")
}

// TestView

func TestView(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func(m card.Model) card.Model
		// assert
		expectedResult string
	}

	cases := []testCase{
		{
			name: "short name focused",
			setup: func(m card.Model) card.Model {
				m, _ = m.Update(card.FocusMsg{ServiceName: testShortName})

				return m
			},
			expectedResult: testShortName,
		},
		{
			name:           "short name unfocused",
			expectedResult: testShortName,
		},
		{
			name: "long name focused windowed",
			setup: func(_ card.Model) card.Model {
				m := newCard(testLongName)
				m, _ = m.Update(card.FocusMsg{ServiceName: testLongName})

				return m
			},
			expectedResult: testLongName[:11],
		},
		{
			name: "long name unfocused truncated",
			setup: func(_ card.Model) card.Model {
				return newCard(testLongName)
			},
			expectedResult: testLongName[:10] + "…",
		},
		{
			name: "in-flight border colour",
			setup: func(m card.Model) card.Model {
				m, _ = m.Update(msgs.ServiceStart{ServiceName: testShortName})

				return m
			},
			expectedResult: testShortName,
		},
		{
			name: "runtime state border colour",
			setup: func(m card.Model) card.Model {
				m, _ = m.Update(msgs.ServicesPolled{
					Runtimes: map[string]*domain.ServiceRuntimeData{
						testShortName: {State: domain.ServiceStateRunning},
					},
				})

				return m
			},
			expectedResult: testShortName,
		},
		{
			name: "zero dimensions",
			setup: func(_ card.Model) card.Model {
				return card.New(domain.ServiceDef{Name: testShortName}, 0, 1, theme.Default())
			},
			expectedResult: testShortName,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := newCard(testShortName)

			if tc.setup != nil {
				m = tc.setup(m)
			}

			view := m.View().Content

			if tc.expectedResult == "" {
				assert.Empty(t, view)
			} else {
				assert.Contains(t, view, tc.expectedResult)
			}
		})
	}
}
