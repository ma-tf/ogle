package accordion_test

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/accordion"
	"github.com/ma-tf/ogle/internal/ui/components/accordion/value"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	dash        = "—"
	imageNginx  = "nginx:latest"
	containerID = "abc123def4567890"
	upTwoHours  = "Up 2 hours"
	serviceWeb  = "web"
)

//nolint:gochecknoglobals // shared test fixtures
var testProject = &domain.Project{
	Name: "testproj",
	File: "/path/to/compose.yaml",
	Services: []domain.ServiceDef{
		{Name: serviceWeb, Image: imageNginx, Ports: []string{"80:80", "443:443"}},
	},
}

//nolint:gochecknoglobals // shared test fixtures
var multiServiceProject = &domain.Project{
	Name: "testproj",
	File: "/path/to/compose.yaml",
	Services: []domain.ServiceDef{
		{Name: serviceWeb, Image: imageNginx, Ports: []string{"80:80"}},
		{Name: "api", Image: "api:latest", Ports: []string{"8080:8080"}},
	},
}

//nolint:funlen // test table with many cases
func TestView(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name           string
		setup          func() accordion.Model
		expectedResult string
	}

	cases := []testCase{
		{
			name: "empty when no services",
			setup: func() accordion.Model {
				return accordion.New(&domain.Project{}, 100, 24, theme.Default(), nil)
			},
			expectedResult: "",
		},
		{
			name: "empty when width is zero",
			setup: func() accordion.Model {
				return accordion.New(testProject, 0, 24, theme.Default(), nil)
			},
			expectedResult: "",
		},
		{
			name: "expanded header indicator",
			setup: func() accordion.Model {
				return accordion.New(testProject, 100, 24, theme.Default(), nil)
			},
			expectedResult: "▼",
		},
		{
			name: "image label rendered",
			setup: func() accordion.Model {
				return accordion.New(testProject, 100, 24, theme.Default(), nil)
			},
			expectedResult: "Image:",
		},
		{
			name: "container id label rendered",
			setup: func() accordion.Model {
				return accordion.New(testProject, 100, 24, theme.Default(), nil)
			},
			expectedResult: "Container ID:",
		},
		{
			name: "created label rendered",
			setup: func() accordion.Model {
				return accordion.New(testProject, 100, 24, theme.Default(), nil)
			},
			expectedResult: "Created:",
		},
		{
			name: "state label rendered",
			setup: func() accordion.Model {
				return accordion.New(testProject, 100, 24, theme.Default(), nil)
			},
			expectedResult: "State:",
		},
		{
			name: "ports label rendered",
			setup: func() accordion.Model {
				return accordion.New(testProject, 100, 24, theme.Default(), nil)
			},
			expectedResult: "Ports:",
		},
		{
			name: "image value from service def",
			setup: func() accordion.Model {
				return accordion.New(testProject, 100, 24, theme.Default(), nil)
			},
			expectedResult: imageNginx,
		},
		{
			name: "ports value from service def",
			setup: func() accordion.Model {
				return accordion.New(testProject, 100, 24, theme.Default(), nil)
			},
			expectedResult: "80:80, 443:443",
		},
		{
			name: "placeholders when runtime is nil",
			setup: func() accordion.Model {
				return accordion.New(testProject, 100, 24, theme.Default(), nil)
			},
			expectedResult: dash,
		},
		{
			name: "container id truncated from runtime",
			setup: func() accordion.Model {
				m := accordion.New(testProject, 100, 24, theme.Default(), nil)
				m, _ = m.Update(msgs.ServicesPolled{
					Runtimes: map[string]*domain.ServiceRuntimeData{
						serviceWeb: {
							ContainerID: containerID,
							State:       domain.ServiceStateRunning,
							Status:      upTwoHours,
							CreatedAt:   time.Now().Add(-2 * time.Hour),
						},
					},
				})

				return m
			},
			expectedResult: "abc123def456",
		},
		{
			name: "state string from runtime",
			setup: func() accordion.Model {
				m := accordion.New(testProject, 100, 24, theme.Default(), nil)
				m, _ = m.Update(msgs.ServicesPolled{
					Runtimes: map[string]*domain.ServiceRuntimeData{
						serviceWeb: {
							ContainerID: containerID,
							State:       domain.ServiceStateRunning,
							Status:      upTwoHours,
							CreatedAt:   time.Now().Add(-2 * time.Hour),
						},
					},
				})

				return m
			},
			expectedResult: "Up 2 hours",
		},
		{
			name: "created age from runtime",
			setup: func() accordion.Model {
				m := accordion.New(testProject, 100, 24, theme.Default(), nil)
				m, _ = m.Update(msgs.ServicesPolled{
					Runtimes: map[string]*domain.ServiceRuntimeData{
						serviceWeb: {
							ContainerID: containerID,
							State:       domain.ServiceStateRunning,
							Status:      upTwoHours,
							CreatedAt:   time.Now().Add(-2 * time.Hour),
						},
					},
				})

				return m
			},
			expectedResult: "h ago",
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
		name        string
		msg         tea.Msg
		expectedMsg tea.Msg
	}

	cases := []testCase{
		{
			name:        "ServiceSelected triggers sync",
			msg:         msgs.ServiceSelected{ServiceName: "api"},
			expectedMsg: value.StartMsg{Gen: 2},
		},
		{
			name: "ServicesPolled stores runtime and triggers sync",
			msg: msgs.ServicesPolled{
				Runtimes: map[string]*domain.ServiceRuntimeData{
					serviceWeb: {
						ContainerID: "abc123def456",
						State:       domain.ServiceStateRunning,
						Status:      upTwoHours,
					},
				},
			},
			expectedMsg: value.StartMsg{Gen: 2},
		},
		{
			name:        "ServicesPolled error does not update runtime",
			msg:         msgs.ServicesPolled{Err: assert.AnError},
			expectedMsg: nil,
		},
		{
			name:        "WindowSizeMsg updates dimensions",
			msg:         tea.WindowSizeMsg{Width: 200, Height: 50},
			expectedMsg: nil,
		},
		{
			name:        "theme.Changed updates theme and triggers sync",
			msg:         theme.Changed{Theme: theme.DefaultLight()},
			expectedMsg: value.StartMsg{Gen: 2},
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

			m := accordion.New(multiServiceProject, 100, 24, theme.Default(), nil)
			_ = m.Init()

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
