package logpane_test

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/logpane"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const clearSvcName = "test"

//nolint:funlen,maintidx // long test with many table-driven cases
func TestUpdate(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func() logpane.Model

		// act
		msg tea.Msg

		// assert
		expectedMsg tea.Msg
		check       func(*testing.T, logpane.Model)
	}

	cases := []testCase{
		{
			name: "LogLinesAvailable drains channel and appends lines",
			setup: func() logpane.Model {
				ch := make(chan string, 3)
				ch <- "line a"

				ch <- "line b"

				ch <- "line c"

				return logpane.New(theme.Default(), 120, 100, 100, ch)
			},
			msg:         msgs.LogLinesAvailable{},
			expectedMsg: nil,
			check: func(t *testing.T, m logpane.Model) {
				t.Helper()

				v := m.View().Content
				assert.Contains(t, v, "line a")
				assert.Contains(t, v, "line b")
				assert.Contains(t, v, "line c")
			},
		},

		{
			name: "LogLinesAvailable with closed channel sets lineCh to nil",
			setup: func() logpane.Model {
				ch := make(chan string)
				close(ch)

				return logpane.New(theme.Default(), 120, 100, 100, ch)
			},
			msg:         msgs.LogLinesAvailable{},
			expectedMsg: nil,
			check: func(t *testing.T, m logpane.Model) {
				t.Helper()

				_, cmd := m.Update(msgs.LogLinesAvailable{})
				require.Nil(t, cmd)
			},
		},

		{
			name: "LogLinesAvailable scrolls viewport to bottom if was at bottom",
			setup: func() logpane.Model {
				ch := make(chan string, 100)
				for range 10 {
					ch <- "line content"
				}

				m := logpane.New(theme.Default(), 120, 8, 100, ch)
				m, _ = m.Update(msgs.LogLinesAvailable{})

				for range 5 {
					ch <- "new content"
				}

				return m
			},
			msg:         msgs.LogLinesAvailable{},
			expectedMsg: nil,
			check: func(t *testing.T, m logpane.Model) {
				t.Helper()

				v := m.View().Content
				assert.Contains(t, v, "new content")
				assert.NotContains(t, v, "line content")
			},
		},

		{
			name: "ToggleLogWrap toggles wrap and restores scroll position",
			setup: func() logpane.Model {
				ch := make(chan string, 1)

				longLine := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
				ch <- longLine

				m := logpane.New(theme.Default(), 120, 100, 100, ch)
				m, _ = m.Update(msgs.LogLinesAvailable{})

				return m
			},
			msg:         msgs.ToggleLogWrap{},
			expectedMsg: nil,
			check: func(t *testing.T, m logpane.Model) {
				t.Helper()

				v1 := m.View().Content
				m2, cmd := m.Update(msgs.ToggleLogWrap{})
				require.Nil(t, cmd)

				assert.NotEqual(t, v1, m2.View().Content)
			},
		},

		{
			name: "WindowSizeMsg recalculates dimensions",
			setup: func() logpane.Model {
				ch := make(chan string, 1)
				ch <- "hello"

				return logpane.New(theme.Default(), 100, 100, 100, ch)
			},
			msg:         tea.WindowSizeMsg{Width: 200, Height: 200},
			expectedMsg: nil,
		},

		{
			name: "WindowSizeMsg scrolls to bottom when at bottom",
			setup: func() logpane.Model {
				ch := make(chan string, 30)
				for i := range 20 {
					ch <- fmt.Sprintf("line %d", i)
				}

				m := logpane.New(theme.Default(), 120, 7, 100, ch)
				m, _ = m.Update(msgs.LogLinesAvailable{})

				return m
			},
			msg:         tea.WindowSizeMsg{Width: 200, Height: 12},
			expectedMsg: nil,
			check: func(t *testing.T, m logpane.Model) {
				t.Helper()

				v := m.View().Content
				assert.Contains(t, v, "line 19")
				assert.NotContains(t, v, "line 0")
			},
		},

		{
			name: "ClearLogBuffer clears all lines from view",
			setup: func() logpane.Model {
				ch := make(chan string, 3)
				ch <- "line a"

				ch <- "line b"

				ch <- "line c"

				m := logpane.New(theme.Default(), 120, 100, 100, ch)
				m, _ = m.Update(msgs.LogLinesAvailable{})

				return m
			},
			msg:         msgs.ClearLogBuffer{ServiceName: clearSvcName},
			expectedMsg: nil,
			check: func(t *testing.T, m logpane.Model) {
				t.Helper()

				v := m.View().Content
				assert.NotContains(t, v, "line a")
				assert.NotContains(t, v, "line b")
				assert.NotContains(t, v, "line c")
			},
		},

		{
			name: "ClearLogBuffer resets scroll to bottom so new lines appear",
			setup: func() logpane.Model {
				ch := make(chan string, 30)
				for i := range 20 {
					ch <- fmt.Sprintf("line %d", i)
				}

				m := logpane.New(theme.Default(), 120, 7, 100, ch)
				m, _ = m.Update(msgs.LogLinesAvailable{})

				// scroll up so bottom lines are not visible
				m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
				m, _ = m.Update(msgs.ClearLogBuffer{ServiceName: clearSvcName})

				for i := 20; i < 30; i++ {
					ch <- fmt.Sprintf("new line %d", i)
				}

				return m
			},
			msg:         msgs.LogLinesAvailable{},
			expectedMsg: nil,
			check: func(t *testing.T, m logpane.Model) {
				t.Helper()

				v := m.View().Content
				assert.Contains(t, v, "new line 29")
				assert.NotContains(t, v, "line 0")
				assert.NotContains(t, v, "line 19")
			},
		},

		{
			name: "ClearLogBuffer on empty buffer no-ops",
			setup: func() logpane.Model {
				return logpane.New(theme.Default(), 120, 100, 100, make(chan string, 1))
			},
			msg:         msgs.ClearLogBuffer{ServiceName: clearSvcName},
			expectedMsg: nil,
			check: func(t *testing.T, m logpane.Model) {
				t.Helper()

				assert.NotPanics(t, func() { m.View() })
			},
		},

		{
			name: "ClearLogBuffer discards buffered channel lines",
			setup: func() logpane.Model {
				ch := make(chan string, 10)
				ch <- "stale line 1"

				ch <- "stale line 2"

				m := logpane.New(theme.Default(), 120, 100, 100, ch)
				m, _ = m.Update(msgs.LogLinesAvailable{})

				ch <- "stale line 3"

				ch <- "stale line 4"

				m, _ = m.Update(msgs.ClearLogBuffer{ServiceName: clearSvcName})

				return m
			},
			msg:         msgs.LogLinesAvailable{},
			expectedMsg: nil,
			check: func(t *testing.T, m logpane.Model) {
				t.Helper()

				v := m.View().Content
				assert.NotContains(t, v, "stale")
			},
		},

		{
			name: "theme.Changed updates theme pointer",
			setup: func() logpane.Model {
				ch := make(chan string, 1)
				ch <- "hello"

				return logpane.New(theme.Default(), 120, 100, 100, ch)
			},
			msg:         theme.Changed{Theme: theme.DefaultLight()},
			expectedMsg: nil,
			check: func(t *testing.T, m logpane.Model) {
				t.Helper()
				assert.NotPanics(t, func() { m.View() })
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := tc.setup()
			m, cmd := m.Update(tc.msg)

			if tc.expectedMsg != nil {
				require.NotNil(t, cmd)
				require.Equal(t, tc.expectedMsg, cmd())
			} else {
				require.Nil(t, cmd)
			}

			if tc.check != nil {
				tc.check(t, m)
			}
		})
	}
}

func TestView(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func() logpane.Model

		// assert
		expectedResult string
	}

	cases := []testCase{
		{
			name: "empty log content renders border",
			setup: func() logpane.Model {
				return logpane.New(theme.Default(), 120, 100, 100, make(chan string, 1))
			},
			expectedResult: "╭",
		},
		{
			name: "non-empty log content shows lines",
			setup: func() logpane.Model {
				ch := make(chan string, 2)
				ch <- "visible line"

				ch <- "another line"

				m := logpane.New(theme.Default(), 120, 100, 100, ch)
				m, _ = m.Update(msgs.LogLinesAvailable{})

				return m
			},
			expectedResult: "visible line",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := tc.setup()

			if tc.expectedResult == "" {
				assert.Empty(t, m.View().Content)
			} else {
				assert.Contains(t, m.View().Content, tc.expectedResult)
			}
		})
	}
}
