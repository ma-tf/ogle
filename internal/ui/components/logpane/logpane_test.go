package logpane_test

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	clipmocks "github.com/ma-tf/ogle/internal/clipboard/mocks"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/logpane"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	clearSvcName = "test"
	longLine     = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

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
			name: "ToggleLogWrap emits LogWrapStatus with On=true when wrap turned ON",
			setup: func() logpane.Model {
				ch := make(chan string, 1)
				ch <- "short line"

				m := logpane.New(theme.Default(), 120, 100, 100, ch)
				m, _ = m.Update(msgs.LogLinesAvailable{})

				return m
			},
			msg:         msgs.ToggleLogWrap{},
			expectedMsg: msgs.LogWrapStatus{On: true, Overflow: false},
		},

		{
			name: "ToggleLogWrap emits LogWrapStatus with Overflow=true when OFF and lines overflow",
			setup: func() logpane.Model {
				ch := make(chan string, 1)
				ch <- longLine

				m := logpane.New(theme.Default(), 120, 100, 100, ch)
				m, _ = m.Update(msgs.LogLinesAvailable{})
				m, _ = m.Update(msgs.ToggleLogWrap{})

				return m
			},
			msg:         msgs.ToggleLogWrap{},
			expectedMsg: msgs.LogWrapStatus{On: false, Overflow: true},
		},

		{
			name: "ToggleLogWrap emits LogWrapStatus with Overflow=false when OFF and no overflow",
			setup: func() logpane.Model {
				ch := make(chan string, 1)
				ch <- "short line"

				m := logpane.New(theme.Default(), 120, 100, 100, ch)
				m, _ = m.Update(msgs.LogLinesAvailable{})
				m, _ = m.Update(msgs.ToggleLogWrap{})

				return m
			},
			msg:         msgs.ToggleLogWrap{},
			expectedMsg: msgs.LogWrapStatus{On: false, Overflow: false},
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
			name: "WindowSizeMsg emits LogWrapStatus when narrowing creates overflow",
			setup: func() logpane.Model {
				ch := make(chan string, 1)
				ch <- longLine

				m := logpane.New(theme.Default(), 200, 100, 100, ch)
				m, _ = m.Update(msgs.LogLinesAvailable{})

				return m
			},
			msg:         tea.WindowSizeMsg{Width: 50, Height: 100},
			expectedMsg: msgs.LogWrapStatus{On: false, Overflow: true},
		},

		{
			name: "WindowSizeMsg emits LogWrapStatus when widening removes overflow",
			setup: func() logpane.Model {
				ch := make(chan string, 1)
				ch <- longLine

				m := logpane.New(theme.Default(), 50, 100, 100, ch)
				m, _ = m.Update(msgs.LogLinesAvailable{})

				return m
			},
			msg:         tea.WindowSizeMsg{Width: 200, Height: 100},
			expectedMsg: msgs.LogWrapStatus{On: false, Overflow: false},
		},

		{
			name: "WindowSizeMsg emits no LogWrapStatus when overflow unchanged",
			setup: func() logpane.Model {
				ch := make(chan string, 1)
				ch <- longLine

				m := logpane.New(theme.Default(), 50, 100, 100, ch)
				m, _ = m.Update(msgs.LogLinesAvailable{})

				return m
			},
			msg:         tea.WindowSizeMsg{Width: 60, Height: 100},
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
			name: "LogLinesAvailable emits LogWrapStatus when overflow is detected",
			setup: func() logpane.Model {
				ch := make(chan string, 1)
				ch <- longLine

				return logpane.New(theme.Default(), 120, 100, 100, ch)
			},
			msg:         msgs.LogLinesAvailable{},
			expectedMsg: msgs.LogWrapStatus{On: false, Overflow: true},
		},

		{
			name: "LogLinesAvailable with wrap ON emits no LogWrapStatus",
			setup: func() logpane.Model {
				ch := make(chan string, 1)
				ch <- longLine

				m := logpane.New(theme.Default(), 120, 100, 100, ch)
				m, _ = m.Update(msgs.ToggleLogWrap{})

				return m
			},
			msg: msgs.LogLinesAvailable{},
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
			name: "ReportWrapStatus emits LogWrapStatus with wrap OFF and no overflow",
			setup: func() logpane.Model {
				ch := make(chan string, 1)
				ch <- "short line"

				return logpane.New(theme.Default(), 120, 100, 100, ch)
			},
			msg:         msgs.ReportWrapStatus{ServiceName: clearSvcName},
			expectedMsg: msgs.LogWrapStatus{On: false, Overflow: false, ServiceName: clearSvcName},
		},

		{
			name: "ReportWrapStatus emits LogWrapStatus with overflow when wrap OFF and lines overflow",
			setup: func() logpane.Model {
				ch := make(chan string, 1)
				ch <- longLine

				m := logpane.New(theme.Default(), 120, 100, 100, ch)
				m, _ = m.Update(msgs.LogLinesAvailable{})

				return m
			},
			msg:         msgs.ReportWrapStatus{ServiceName: clearSvcName},
			expectedMsg: msgs.LogWrapStatus{On: false, Overflow: true, ServiceName: clearSvcName},
		},

		{
			name: "ReportWrapStatus emits LogWrapStatus with wrap ON, no overflow",
			setup: func() logpane.Model {
				ch := make(chan string, 1)
				ch <- "short line"

				m := logpane.New(theme.Default(), 120, 100, 100, ch)
				m, _ = m.Update(msgs.LogLinesAvailable{})
				m, _ = m.Update(msgs.ToggleLogWrap{})

				return m
			},
			msg:         msgs.ReportWrapStatus{ServiceName: clearSvcName},
			expectedMsg: msgs.LogWrapStatus{On: true, Overflow: false, ServiceName: clearSvcName},
		},

		{
			name: "ReportWrapStatus is read-only does not toggle wrap",
			setup: func() logpane.Model {
				ch := make(chan string, 1)
				ch <- "short line"

				m := logpane.New(theme.Default(), 120, 100, 100, ch)
				m, _ = m.Update(msgs.LogLinesAvailable{})

				return m
			},
			msg:         msgs.ReportWrapStatus{ServiceName: clearSvcName},
			expectedMsg: msgs.LogWrapStatus{On: false, Overflow: false, ServiceName: clearSvcName},
			check: func(t *testing.T, m logpane.Model) {
				t.Helper()

				// Second ReportWrapStatus confirms first did not mutate state
				_, cmd := m.Update(msgs.ReportWrapStatus{ServiceName: "readonly-check"})
				require.NotNil(t, cmd)
				require.Equal(t,
					msgs.LogWrapStatus{On: false, Overflow: false, ServiceName: "readonly-check"},
					cmd())

				// Toggle confirms wrap was still OFF after ReportWrapStatus
				_, cmd = m.Update(msgs.ToggleLogWrap{})
				require.NotNil(t, cmd)
				require.Equal(t,
					msgs.LogWrapStatus{On: true, Overflow: false, ServiceName: ""},
					cmd())
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

		{
			name: "FrameHeight does not produce a command",
			setup: func() logpane.Model {
				ch := make(chan string, 1)
				ch <- "line"

				m := logpane.New(theme.Default(), 120, 24, 100, ch)
				m, _ = m.Update(msgs.LogLinesAvailable{})

				return m
			},
			msg:         msgs.FrameHeight{Height: 5},
			expectedMsg: nil,
			check: func(t *testing.T, m logpane.Model) {
				t.Helper()

				v := m.View().Content
				assert.NotEmpty(t, v, "view should render after FrameHeight update")
			},
		},

		{
			name: "FrameHeight after WindowSizeMsg adjusts layout",
			setup: func() logpane.Model {
				ch := make(chan string, 10)
				for i := range 5 {
					ch <- fmt.Sprintf("test_line_%d", i)
				}

				m := logpane.New(theme.Default(), 120, 24, 100, ch)
				m, _ = m.Update(msgs.LogLinesAvailable{})
				m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
				m, _ = m.Update(msgs.FrameHeight{Height: 4})

				return m
			},
			msg:         msgs.FrameHeight{Height: 8},
			expectedMsg: nil,
			check: func(t *testing.T, m logpane.Model) {
				t.Helper()

				v := m.View().Content
				assert.Contains(t, v, "test_line_0",
					"should still show lines after FrameHeight update")
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

func TestUpdate_YankLogLines(t *testing.T) {
	t.Parallel()

	t.Run("copies last N lines to clipboard and emits status", func(t *testing.T) {
		t.Parallel()

		mockClip := clipmocks.NewMockClipboard(t)

		ch := make(chan string, 3)
		ch <- "line 1"

		ch <- "line 2"

		ch <- "line 3"

		m := logpane.New(theme.Default(), 120, 100, 100, ch, logpane.WithClipboard(mockClip))
		m, _ = m.Update(msgs.LogLinesAvailable{})

		mockClip.EXPECT().WriteAll("line 2\nline 3").Return(nil)

		_, cmd := m.Update(msgs.YankLogLines{ServiceName: clearSvcName, Count: 2})
		require.NotNil(t, cmd)
		require.Equal(t, msgs.DisplayStatus{Msg: "Yanked 2 line(s)"}, cmd())
	})

	t.Run("count exceeding lines clamps to available lines", func(t *testing.T) {
		t.Parallel()

		mockClip := clipmocks.NewMockClipboard(t)

		ch := make(chan string, 2)
		ch <- "only line"

		m := logpane.New(theme.Default(), 120, 100, 100, ch, logpane.WithClipboard(mockClip))
		m, _ = m.Update(msgs.LogLinesAvailable{})

		mockClip.EXPECT().WriteAll("only line").Return(nil)

		_, cmd := m.Update(msgs.YankLogLines{ServiceName: clearSvcName, Count: 100})
		require.NotNil(t, cmd)
		require.Equal(t, msgs.DisplayStatus{Msg: "Yanked 1 line(s)"}, cmd())
	})

	t.Run("empty buffer shows no lines status and does not write clipboard", func(t *testing.T) {
		t.Parallel()

		mockClip := clipmocks.NewMockClipboard(t)
		m := logpane.New(
			theme.Default(), 120, 100, 100, make(chan string, 1),
			logpane.WithClipboard(mockClip),
		)

		_, cmd := m.Update(msgs.YankLogLines{ServiceName: clearSvcName, Count: 1})
		require.NotNil(t, cmd)
		require.Equal(t, msgs.DisplayStatus{Msg: "No log lines to yank"}, cmd())
	})

	t.Run("nil clipboard no-ops", func(t *testing.T) {
		t.Parallel()

		ch := make(chan string, 1)
		ch <- "some line"

		m := logpane.New(theme.Default(), 120, 100, 100, ch)
		m, _ = m.Update(msgs.LogLinesAvailable{})

		_, cmd := m.Update(msgs.YankLogLines{ServiceName: clearSvcName, Count: 1})
		require.Nil(t, cmd)
	})

	t.Run("single line with count 1 copies that line", func(t *testing.T) {
		t.Parallel()

		mockClip := clipmocks.NewMockClipboard(t)

		ch := make(chan string, 1)
		ch <- "single log line"

		m := logpane.New(theme.Default(), 120, 100, 100, ch, logpane.WithClipboard(mockClip))
		m, _ = m.Update(msgs.LogLinesAvailable{})

		mockClip.EXPECT().WriteAll("single log line").Return(nil)

		_, cmd := m.Update(msgs.YankLogLines{ServiceName: clearSvcName, Count: 1})
		require.NotNil(t, cmd)
		require.Equal(t, msgs.DisplayStatus{Msg: "Yanked 1 line(s)"}, cmd())
	})
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
