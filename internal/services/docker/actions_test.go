package docker_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	svcdocker "github.com/ma-tf/ogle/internal/services/docker"
)

const (
	testProject    = "myproject"
	testService    = "web"
	testCmdName    = "docker"
	testCmdCompose = "compose"
	errStderr      = "service not found"
)

// fakeCommander records CommandContext invocations for testing.
// When fail is true, CommandContext returns a command that exits with code 1.
// If failStderr is set, the command also writes that text to stderr.
type fakeCommander struct {
	name       string
	args       []string
	cmd        *exec.Cmd
	fail       bool
	failStderr string
}

func (f *fakeCommander) CommandContext(_ context.Context, name string, arg ...string) *exec.Cmd {
	f.name = name

	f.args = append([]string{}, arg...)

	if f.fail {
		if f.failStderr != "" {
			f.cmd = exec.CommandContext(ctxTodo, "sh", "-c",
				"echo 'service not found' >&2 && exit 1")
		} else {
			f.cmd = exec.CommandContext(ctxTodo, "false")
		}
	} else {
		f.cmd = exec.CommandContext(ctxTodo, "true")
	}

	return f.cmd
}

var ctxTodo = context.TODO() //nolint:gochecknoglobals // shared test context

func TestActions(t *testing.T) { //nolint:funlen // table-driven with 4 actions
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		projectName string
		serviceName string
		action      domain.ServiceAction
		runAction   func(context.Context, svcdocker.Docker, string, string, string) tea.Cmd
		// assert
		expectedName string
		expectedArgs []string
	}

	cases := []testCase{
		{
			name:        "Stop",
			projectName: testProject,
			serviceName: testService,
			action:      domain.ServiceActionStop,
			runAction: func(ctx context.Context, s svcdocker.Docker, file, proj, svc string) tea.Cmd {
				return s.Stop(ctx, file, proj, svc)
			},
			expectedName: testCmdName,
			expectedArgs: []string{
				testCmdCompose,
				"-f",
				"%s",
				"-p",
				testProject,
				"stop",
				testService,
			},
		},
		{
			name:        "Start",
			projectName: testProject,
			serviceName: testService,
			action:      domain.ServiceActionStart,
			runAction: func(ctx context.Context, s svcdocker.Docker, file, proj, svc string) tea.Cmd {
				return s.Start(ctx, file, proj, svc)
			},
			expectedName: testCmdName,
			expectedArgs: []string{
				testCmdCompose,
				"-f",
				"%s",
				"-p",
				testProject,
				"up",
				"-d",
				testService,
			},
		},
		{
			name:        "Restart",
			projectName: testProject,
			serviceName: testService,
			action:      domain.ServiceActionRestart,
			runAction: func(ctx context.Context, s svcdocker.Docker, file, proj, svc string) tea.Cmd {
				return s.Restart(ctx, file, proj, svc)
			},
			expectedName: testCmdName,
			expectedArgs: []string{
				testCmdCompose,
				"-f",
				"%s",
				"-p",
				testProject,
				"restart",
				testService,
			},
		},
		{
			name:        "Rebuild",
			projectName: testProject,
			serviceName: testService,
			action:      domain.ServiceActionRebuild,
			runAction: func(ctx context.Context, s svcdocker.Docker, file, proj, svc string) tea.Cmd {
				return s.Rebuild(ctx, file, proj, svc)
			},
			expectedName: testCmdName,
			expectedArgs: []string{
				testCmdCompose,
				"-f",
				"%s",
				"-p",
				testProject,
				"up",
				"--build",
				"-d",
				testService,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			composeFile := filepath.Join(t.TempDir(), "compose.yaml")

			fc := &fakeCommander{}
			s := svcdocker.New(svcdocker.WithCommander(fc))

			ctx := context.Background()
			teaCmd := tc.runAction(ctx, s, composeFile, tc.projectName, tc.serviceName)
			require.NotNil(t, teaCmd)

			msg := teaCmd()
			require.NotNil(t, msg)

			expectedArgs := make([]string, len(tc.expectedArgs))
			for i, arg := range tc.expectedArgs {
				if arg == "%s" {
					expectedArgs[i] = composeFile
				} else {
					expectedArgs[i] = arg
				}
			}

			assert.Equal(t, tc.expectedName, fc.name)
			assert.Equal(t, expectedArgs, fc.args)

			expectedDir := filepath.Dir(composeFile)
			assert.Equal(t, expectedDir, fc.cmd.Dir)

			completed, ok := msg.(msgs.ServiceActionCompleted)
			require.True(t, ok, "expected ServiceActionCompleted, got %T", msg)
			assert.Equal(t, tc.serviceName, completed.ServiceName)
			assert.Equal(t, tc.action, completed.Action)
			assert.NoError(t, completed.Err)
		})
	}
}

func TestActionsErrorPaths(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		projectName string
		serviceName string
		action      domain.ServiceAction
		runAction   func(context.Context, svcdocker.Docker, string, string, string) tea.Cmd
		failStderr  string
	}

	cases := []testCase{
		{
			name:        "Stop",
			projectName: testProject,
			serviceName: testService,
			action:      domain.ServiceActionStop,
			runAction: func(ctx context.Context, s svcdocker.Docker, file, proj, svc string) tea.Cmd {
				return s.Stop(ctx, file, proj, svc)
			},
			failStderr: errStderr,
		},
		{
			name:        "Start",
			projectName: testProject,
			serviceName: testService,
			action:      domain.ServiceActionStart,
			runAction: func(ctx context.Context, s svcdocker.Docker, file, proj, svc string) tea.Cmd {
				return s.Start(ctx, file, proj, svc)
			},
			failStderr: errStderr,
		},
		{
			name:        "Restart",
			projectName: testProject,
			serviceName: testService,
			action:      domain.ServiceActionRestart,
			runAction: func(ctx context.Context, s svcdocker.Docker, file, proj, svc string) tea.Cmd {
				return s.Restart(ctx, file, proj, svc)
			},
			failStderr: errStderr,
		},
		{
			name:        "Rebuild",
			projectName: testProject,
			serviceName: testService,
			action:      domain.ServiceActionRebuild,
			runAction: func(ctx context.Context, s svcdocker.Docker, file, proj, svc string) tea.Cmd {
				return s.Rebuild(ctx, file, proj, svc)
			},
			failStderr: errStderr,
		},
		{
			name:        "empty stderr",
			projectName: testProject,
			serviceName: testService,
			action:      domain.ServiceActionStop,
			runAction: func(ctx context.Context, s svcdocker.Docker, file, proj, svc string) tea.Cmd {
				return s.Stop(ctx, file, proj, svc)
			},
			failStderr: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			composeFile := filepath.Join(t.TempDir(), "compose.yaml")

			fc := &fakeCommander{fail: true, failStderr: tc.failStderr}
			s := svcdocker.New(svcdocker.WithCommander(fc))

			ctx := context.Background()
			teaCmd := tc.runAction(ctx, s, composeFile, tc.projectName, tc.serviceName)
			require.NotNil(t, teaCmd)

			msg := teaCmd()
			require.NotNil(t, msg)

			completed, ok := msg.(msgs.ServiceActionCompleted)
			require.True(t, ok, "expected ServiceActionCompleted, got %T", msg)
			assert.Equal(t, tc.serviceName, completed.ServiceName)
			assert.Equal(t, tc.action, completed.Action)
			require.Error(t, completed.Err)

			if tc.failStderr != "" {
				require.ErrorContains(t, completed.Err, tc.failStderr)
			}

			var exitErr *exec.ExitError
			require.ErrorAs(t, completed.Err, &exitErr)
			assert.Equal(t, 1, exitErr.ExitCode())
		})
	}
}
