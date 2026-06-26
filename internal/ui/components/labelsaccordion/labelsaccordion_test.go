package labelsaccordion_test

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/accordion/value"
	"github.com/ma-tf/ogle/internal/ui/components/labelsaccordion"
	"github.com/ma-tf/ogle/internal/ui/layout"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	collapsedHeader  = "▶ Labels"
	zoneLabelsHeader = "labels-header"
	testLabelValue   = "bar"
	testLabelKey     = "ogle.foo"
	testKey1         = "key1"
	testVal1         = "value1"
	testKey2         = "key2"
	testVal2         = "value2"
	testLongKey      = "a-very-long-label-key-that-exceeds-the-cap"
	testFoo          = "foo"
	testShortVal     = "short"
	testLongVal      = "a-long-value-that-requires-scrolling"
)

func expandAccordion(
	t *testing.T,
	m labelsaccordion.Model,
	zm *zone.Manager,
) (labelsaccordion.Model, tea.Cmd) {
	t.Helper()

	view := m.View()
	zm.Scan(view.Content)

	require.Eventually(t, func() bool {
		zi := zm.Get(zoneLabelsHeader)

		return zi != nil && !zi.IsZero()
	}, time.Second, 10*time.Millisecond, "labels header zone should become available")

	zi := zm.Get(zoneLabelsHeader)

	return m.Update(tea.MouseClickMsg{X: zi.StartX, Y: zi.StartY})
}

func TestView_ColumnWidth(t *testing.T) {
	t.Parallel()

	m := labelsaccordion.New(theme.Default(), 100, nil)
	_ = m.Init()

	w := lipgloss.Width(m.View().Content)
	assert.Equal(t, layout.SidebarWidth, w,
		"view should render at sidebar width, not full terminal width")
}

func TestView_CollapsedStates(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name   string
		setup  func() labelsaccordion.Model
		expect string
	}

	cases := []testCase{
		{
			name: "empty on zero width",
			setup: func() labelsaccordion.Model {
				return labelsaccordion.New(theme.Default(), 0, nil)
			},
			expect: "",
		},
		{
			name: "collapsed indicator by default",
			setup: func() labelsaccordion.Model {
				return labelsaccordion.New(theme.Default(), 100, nil)
			},
			expect: collapsedHeader,
		},
		{
			name: "header visible when labels received but collapsed",
			setup: func() labelsaccordion.Model {
				m := labelsaccordion.New(theme.Default(), 100, nil)
				m, _ = m.Update(msgs.ContainerLabelsPolled{
					Labels: map[string]string{testLabelKey: testLabelValue},
				})

				return m
			},
			expect: collapsedHeader,
		},
		{
			name: "header visible on error",
			setup: func() labelsaccordion.Model {
				m := labelsaccordion.New(theme.Default(), 100, nil)
				m, _ = m.Update(msgs.ContainerLabelsPolled{Err: assert.AnError})

				return m
			},
			expect: collapsedHeader,
		},
		{
			name: "header visible when no labels",
			setup: func() labelsaccordion.Model {
				m := labelsaccordion.New(theme.Default(), 100, nil)
				m, _ = m.Update(msgs.ContainerLabelsPolled{Labels: nil})

				return m
			},
			expect: collapsedHeader,
		},
		{
			name: "header visible after service switch clears labels",
			setup: func() labelsaccordion.Model {
				m := labelsaccordion.New(theme.Default(), 100, nil)
				m, _ = m.Update(msgs.ContainerLabelsPolled{
					Labels: map[string]string{testLabelKey: testLabelValue},
				})
				m, _ = m.Update(msgs.ServiceSelected{ServiceName: "other"})

				return m
			},
			expect: collapsedHeader,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := tc.setup()
			_ = m.Init()

			if tc.expect == "" {
				assert.Empty(t, m.View().Content)
			} else {
				assert.Contains(t, m.View().Content, tc.expect)
			}
		})
	}
}

func TestView_ExpandedStates(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		setup  func(*testing.T) labelsaccordion.Model
		expect string
	}{
		{
			name: "key visible when expanded",
			//nolint:thelper // setup factory, not a test helper
			setup: func(t *testing.T) labelsaccordion.Model {
				zm := zone.New()
				m := labelsaccordion.New(theme.Default(), 200, zm)
				m, _ = m.Update(msgs.ContainerLabelsPolled{
					Labels: map[string]string{testKey1: testVal1, testKey2: testVal2},
				})
				m, _ = expandAccordion(t, m, zm)

				return m
			},
			expect: testKey1,
		},
		{
			name: "value visible when expanded",
			//nolint:thelper // setup factory, not a test helper
			setup: func(t *testing.T) labelsaccordion.Model {
				zm := zone.New()
				m := labelsaccordion.New(theme.Default(), 200, zm)
				m, _ = m.Update(msgs.ContainerLabelsPolled{
					Labels: map[string]string{testKey1: testVal1, testKey2: testVal2},
				})
				m, _ = expandAccordion(t, m, zm)

				return m
			},
			expect: testVal2,
		},
		{
			name: "expanded header indicator",
			//nolint:thelper // setup factory, not a test helper
			setup: func(t *testing.T) labelsaccordion.Model {
				zm := zone.New()
				m := labelsaccordion.New(theme.Default(), 200, zm)
				m, _ = m.Update(msgs.ContainerLabelsPolled{
					Labels: map[string]string{testKey1: testVal1},
				})
				m, _ = expandAccordion(t, m, zm)

				return m
			},
			expect: "▼ Labels",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := tc.setup(t)
			_ = m.Init()

			assert.Contains(t, m.View().Content, tc.expect)
		})
	}
}

func TestView_KeyTruncation(t *testing.T) {
	t.Parallel()

	zm := zone.New()
	m := labelsaccordion.New(theme.Default(), 200, zm)
	_ = m.Init()

	m, _ = m.Update(msgs.ContainerLabelsPolled{
		Labels: map[string]string{
			testLongKey: testVal1,
			testKey2:    testVal2,
		},
	})
	m, _ = expandAccordion(t, m, zm)

	assert.Contains(t, m.View().Content, "…",
		"long key should be truncated with ellipsis")
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name        string
		setup       func() labelsaccordion.Model
		msg         tea.Msg
		expectedMsg tea.Msg
	}

	cases := []testCase{
		{
			name: "ContainerLabelsPolled stores labels",
			msg: msgs.ContainerLabelsPolled{
				Labels: map[string]string{testLabelKey: testLabelValue},
			},
			expectedMsg: nil,
		},
		{
			name:        "ContainerLabelsPolled error clears labels",
			msg:         msgs.ContainerLabelsPolled{Err: assert.AnError},
			expectedMsg: nil,
		},
		{
			name:        "ServiceSelected clears labels",
			msg:         msgs.ServiceSelected{ServiceName: "other"},
			expectedMsg: nil,
		},
		{
			name:        "WindowSizeMsg updates width",
			msg:         tea.WindowSizeMsg{Width: 200},
			expectedMsg: nil,
		},
		{
			name:        "theme.Changed updates theme",
			msg:         theme.Changed{Theme: theme.DefaultLight()},
			expectedMsg: nil,
		},
		{
			name:        "MouseClickMsg no-op with nil zone manager",
			msg:         tea.MouseClickMsg{},
			expectedMsg: nil,
		},
		{
			name:        "MouseMotionMsg no-op with nil zone manager",
			msg:         tea.MouseMotionMsg{},
			expectedMsg: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := labelsaccordion.New(theme.Default(), 100, nil)
			_ = m.Init()

			if tc.setup != nil {
				m = tc.setup()
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

func TestUpdate_ContainerLabelsPolled_Expanded_ReturnsStartMsg(t *testing.T) {
	t.Parallel()

	zm := zone.New()
	m := labelsaccordion.New(theme.Default(), 100, zm)
	_ = m.Init()

	m, _ = m.Update(msgs.ContainerLabelsPolled{
		Labels: map[string]string{testFoo: testShortVal},
	})

	m, expandCmd := expandAccordion(t, m, zm)
	require.NotNil(t, expandCmd, "expand should emit a command")

	_, ok := expandCmd().(value.StartMsg)
	require.True(t, ok, "expand cmd should be value.StartMsg")

	_, cmd := m.Update(msgs.ContainerLabelsPolled{
		Labels: map[string]string{testFoo: testLongVal},
	})
	require.NotNil(t, cmd, "content change while expanded should emit StartMsg")
	require.IsType(t, value.StartMsg{}, cmd())
}

func TestUpdate_ContainerLabelsPolled_SameLabels_SkipsRecompute(t *testing.T) {
	t.Parallel()

	zm := zone.New()
	m := labelsaccordion.New(theme.Default(), 100, zm)
	_ = m.Init()

	m, _ = m.Update(msgs.ContainerLabelsPolled{
		Labels: map[string]string{testFoo: testShortVal},
	})

	m, _ = expandAccordion(t, m, zm)

	_, cmd := m.Update(msgs.ContainerLabelsPolled{
		Labels: map[string]string{testFoo: testShortVal},
	})
	require.Nil(t, cmd, "same labels should not emit StartMsg or recompute values")
}

func TestUpdate_MouseClick_Expand_EmitsStartMsg(t *testing.T) {
	t.Parallel()

	zm := zone.New()
	m := labelsaccordion.New(theme.Default(), 100, zm)
	_ = m.Init()

	m, _ = m.Update(msgs.ContainerLabelsPolled{
		Labels: map[string]string{testLabelKey: testLabelValue},
	})

	_, cmd := expandAccordion(t, m, zm)
	require.NotNil(t, cmd, "expand should emit StartMsg")

	msg := cmd()
	require.IsType(t, value.StartMsg{}, msg)
	sm, ok := msg.(value.StartMsg)
	require.True(t, ok)
	require.Equal(t, 1, sm.Gen, "first expand should emit scrollGen=1")
}

func TestUpdate_Collapse_DoesNotEmitStartMsg(t *testing.T) {
	t.Parallel()

	zm := zone.New()
	m := labelsaccordion.New(theme.Default(), 100, zm)
	_ = m.Init()

	m, _ = m.Update(msgs.ContainerLabelsPolled{
		Labels: map[string]string{testLabelKey: testLabelValue},
	})

	m, _ = expandAccordion(t, m, zm)

	_, cmd := expandAccordion(t, m, zm)
	require.Nil(t, cmd, "collapse should not emit StartMsg")
}
