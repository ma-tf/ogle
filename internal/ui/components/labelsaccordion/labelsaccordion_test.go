package labelsaccordion_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/labelsaccordion"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	collapsedHeader = "▶ Labels"
	testLabelValue  = "bar"
	testLabelKey    = "ogle.foo"
)

func widthForTerm(w int) int {
	listMinTermWidth := 80
	listRatio := 30
	pctDivisor := 100

	return max(w, listMinTermWidth) * listRatio / pctDivisor
}

func TestView_ColumnWidth(t *testing.T) {
	t.Parallel()

	m := labelsaccordion.New(theme.Default(), 100, nil)
	_ = m.Init()

	w := lipgloss.Width(m.View().Content)
	columnW := widthForTerm(100)
	assert.Equal(t, columnW, w,
		"view should render at 30%% column width, not full terminal width")
}

func TestView(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func() labelsaccordion.Model
		// assert
		expectedResult string
	}

	cases := []testCase{
		{
			name: "empty on zero width",
			setup: func() labelsaccordion.Model {
				return labelsaccordion.New(theme.Default(), 0, nil)
			},
			expectedResult: "",
		},
		{
			name: "collapsed indicator by default",
			setup: func() labelsaccordion.Model {
				return labelsaccordion.New(theme.Default(), 100, nil)
			},
			expectedResult: collapsedHeader,
		},
		{
			name: "header visible when labels received but collapsed",
			setup: func() labelsaccordion.Model {
				m := labelsaccordion.New(theme.Default(), 100, nil)
				m, _ = m.Update(msgs.ContainerLabelsPolled{
					Labels: map[string]string{"ogle.foo": testLabelValue},
				})

				return m
			},
			expectedResult: collapsedHeader,
		},
		{
			name: "header visible on error",
			setup: func() labelsaccordion.Model {
				m := labelsaccordion.New(theme.Default(), 100, nil)
				m, _ = m.Update(msgs.ContainerLabelsPolled{Err: assert.AnError})

				return m
			},
			expectedResult: collapsedHeader,
		},
		{
			name: "header visible when no ogle labels",
			setup: func() labelsaccordion.Model {
				m := labelsaccordion.New(theme.Default(), 100, nil)
				m, _ = m.Update(msgs.ContainerLabelsPolled{Labels: nil})

				return m
			},
			expectedResult: collapsedHeader,
		},
		{
			name: "header visible after service switch clears labels",
			setup: func() labelsaccordion.Model {
				m := labelsaccordion.New(theme.Default(), 100, nil)
				m, _ = m.Update(msgs.ContainerLabelsPolled{
					Labels: map[string]string{"ogle.foo": testLabelValue},
				})
				m, _ = m.Update(msgs.ServiceSelected{ServiceName: "other"})

				return m
			},
			expectedResult: "▶ Labels",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := tc.setup()
			_ = m.Init()

			if tc.expectedResult == "" {
				assert.Empty(t, m.View().Content)
			} else {
				assert.Contains(t, m.View().Content, tc.expectedResult)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func() labelsaccordion.Model
		// act
		msg tea.Msg
		// assert
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
			name:        "ServiceSelected clears labels and resets collapsed",
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
