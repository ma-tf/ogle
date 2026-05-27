package docker_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/docker"
)

func TestParsePsOutput(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name        string
		input       []byte
		expectedLen int
	}

	tt := []testCase{
		{
			name:        "empty input",
			input:       []byte(""),
			expectedLen: 0,
		},
		{
			name:        "whitespace only",
			input:       []byte("   \n  \n"),
			expectedLen: 0,
		},
		{
			name: "single running service",
			input: []byte(`{"id":"abc123","service":"web","state":"running",` +
				`"createdat":"2026-05-25 12:00:00 +0000 UTC","status":"Up 1h"}` + "\n"),
			expectedLen: 1,
		},
		{
			name: "multiple services",
			input: []byte(`{"id":"abc","service":"web","state":"running",` +
				`"createdat":"2026-05-25 12:00:00 +0000 UTC","status":"Up 1h"}` + "\n" +
				`{"id":"def","service":"db","state":"exited",` +
				`"createdat":"2026-05-25 10:00:00 +0000 UTC","status":"Exited (0) 2h ago"}` + "\n"),
			expectedLen: 2,
		},
		{
			name: "empty service name skipped",
			input: []byte(`{"id":"abc","service":"","state":"running",` +
				`"createdat":"2026-05-25 12:00:00 +0000 UTC","status":"Up 1h"}` + "\n"),
			expectedLen: 0,
		},
		{
			name:        "malformed json",
			input:       []byte(`{invalid json}` + "\n"),
			expectedLen: 0,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := docker.ParsePsOutput(tc.input)

			if tc.name == "malformed json" {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Len(t, result, tc.expectedLen)
		})
	}
}

func TestParsePsOutputServiceRuntimeData(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		input    []byte
		expected map[string]*domain.ServiceRuntimeData
	}

	tt := []testCase{
		{
			name: "running service",
			input: []byte(`{"id":"abc","service":"web","state":"running",` +
				`"createdat":"2026-05-25 12:00:00 +0000 UTC","status":"Up 1h"}` + "\n"),
			expected: map[string]*domain.ServiceRuntimeData{
				"web": {
					ContainerID: "abc",
					State:       domain.ServiceStateRunning,
					Status:      "Up 1h",
				},
			},
		},
		{
			name: "exited service",
			input: []byte(`{"id":"def","service":"db","state":"exited",` +
				`"createdat":"2026-05-25 10:00:00 +0000 UTC","status":"Exited (0)"}` + "\n"),
			expected: map[string]*domain.ServiceRuntimeData{
				"db": {
					ContainerID: "def",
					State:       domain.ServiceStateExited,
					Status:      "Exited (0)",
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := docker.ParsePsOutput(tc.input)
			require.NoError(t, err)

			for serviceName, expectedRuntime := range tc.expected {
				rt, found := result[serviceName]
				require.True(t, found, "service %s not found in result", serviceName)
				assert.Equal(t, expectedRuntime.ContainerID, rt.ContainerID)
				assert.Equal(t, expectedRuntime.State, rt.State)
				assert.Equal(t, expectedRuntime.Status, rt.Status)
			}
		})
	}
}

func TestParseState(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		input    string
		expected domain.ServiceState
	}

	tt := []testCase{
		{name: "running", input: "running", expected: domain.ServiceStateRunning},
		{name: "exited", input: "exited", expected: domain.ServiceStateExited},
		{name: "paused", input: "paused", expected: domain.ServiceStatePaused},
		{name: "restarting", input: "restarting", expected: domain.ServiceStateRestarting},
		{name: "dead", input: "dead", expected: domain.ServiceStateDead},
		{name: "unknown state", input: "removing", expected: domain.ServiceStateUnknown},
		{name: "empty string", input: "", expected: domain.ServiceStateUnknown},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := docker.ParseState(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// psTestCommander records CommandContext invocations and returns a preset command.
type psTestCommander struct {
	name string
	args []string
	cmd  *exec.Cmd
}

func (f *psTestCommander) CommandContext(_ context.Context, name string, arg ...string) *exec.Cmd {
	f.name = name

	f.args = append([]string{}, arg...)

	return f.cmd
}

type psCase struct {
	name         string
	cmd          *exec.Cmd
	check        func(t *testing.T, msg any, fc *psTestCommander, composeFile string)
	expectedName string
	expectedArgs []string
}

func runPsTestCase(t *testing.T, tc psCase) {
	t.Helper()

	const composeFilePlaceholder = "%s"

	composeFile := filepath.Join(t.TempDir(), "compose.yaml")

	fc := &psTestCommander{cmd: tc.cmd}
	s := docker.New(docker.WithCommander(fc))

	ctx := context.Background()
	teaCmd := s.Ps(ctx, composeFile, testProject)
	require.NotNil(t, teaCmd)

	msg := teaCmd()
	require.NotNil(t, msg)

	if tc.expectedName != "" {
		assert.Equal(t, tc.expectedName, fc.name)
	}

	if tc.expectedArgs != nil {
		expectedArgs := make([]string, len(tc.expectedArgs))

		for i, arg := range tc.expectedArgs {
			if arg == composeFilePlaceholder {
				expectedArgs[i] = composeFile
			} else {
				expectedArgs[i] = arg
			}
		}

		assert.Equal(t, expectedArgs, fc.args)
	}

	if tc.check != nil {
		tc.check(t, msg, fc, composeFile)
	}
}

func TestPs(t *testing.T) { //nolint:funlen // table-driven with 5 cases
	t.Parallel()

	const composeFilePlaceholder = "%s"

	tt := []psCase{
		{
			name: "command construction and empty output",
			cmd:  exec.CommandContext(ctxTodo, "true"),
			check: func(t *testing.T, msg any, fc *psTestCommander, composeFile string) {
				t.Helper()

				polled, ok := msg.(msgs.ServicesPolled)
				require.True(t, ok, "expected ServicesPolled, got %T", msg)
				require.NoError(t, polled.Err)
				assert.Empty(t, polled.Runtimes)
				assert.Equal(t, filepath.Dir(composeFile), fc.cmd.Dir)
			},
			expectedName: "docker",
			expectedArgs: []string{
				"compose",
				"-f", composeFilePlaceholder,
				"-p", testProject,
				"ps", "--format", "json",
			},
		},
		{
			name: "valid json-lines output",
			cmd: exec.CommandContext(ctxTodo, "sh", "-c",
				`printf '{"id":"abc","service":"web","state":"running",`+
					`"createdat":"2026-05-25 12:00:00 +0000 UTC","status":"Up 1h"}\n'`),
			check: func(t *testing.T, msg any, _ *psTestCommander, _ string) {
				t.Helper()

				polled, ok := msg.(msgs.ServicesPolled)
				require.True(t, ok, "expected ServicesPolled, got %T", msg)
				require.NoError(t, polled.Err)
				require.Len(t, polled.Runtimes, 1)

				rt, found := polled.Runtimes["web"]
				require.True(t, found, "service web not found in runtimes")
				assert.Equal(t, "abc", rt.ContainerID)
				assert.Equal(t, domain.ServiceStateRunning, rt.State)
				assert.Equal(t, "Up 1h", rt.Status)
			},
		},
		{
			name: "multiple services",
			cmd: exec.CommandContext(ctxTodo, "sh", "-c",
				`printf '{"id":"abc","service":"web","state":"running",`+
					`"createdat":"2026-05-25 12:00:00 +0000 UTC","status":"Up 1h"}\n`+
					`{"id":"def","service":"db","state":"exited",`+
					`"createdat":"2026-05-25 10:00:00 +0000 UTC","status":"Exited (0) 2h ago"}\n'`),
			check: func(t *testing.T, msg any, _ *psTestCommander, _ string) {
				t.Helper()

				polled, ok := msg.(msgs.ServicesPolled)
				require.True(t, ok, "expected ServicesPolled, got %T", msg)
				require.NoError(t, polled.Err)
				require.Len(t, polled.Runtimes, 2)

				for _, name := range []string{"web", "db"} {
					_, found := polled.Runtimes[name]
					require.True(t, found, "service %s not found in runtimes", name)
				}
			},
		},
		{
			name: "malformed json from exec",
			cmd:  exec.CommandContext(ctxTodo, "sh", "-c", `printf '{invalid}\n'`),
			check: func(t *testing.T, msg any, _ *psTestCommander, _ string) {
				t.Helper()

				polled, ok := msg.(msgs.ServicesPolled)
				require.True(t, ok, "expected ServicesPolled, got %T", msg)
				require.Error(t, polled.Err)
				assert.Contains(t, polled.Err.Error(), "parse compose ps output:")
				assert.Nil(t, polled.Runtimes)
			},
		},
		{
			name: "exec error",
			cmd:  exec.CommandContext(ctxTodo, "false"),
			check: func(t *testing.T, msg any, _ *psTestCommander, _ string) {
				t.Helper()

				polled, ok := msg.(msgs.ServicesPolled)
				require.True(t, ok, "expected ServicesPolled, got %T", msg)
				require.Error(t, polled.Err)
				assert.Contains(t, polled.Err.Error(), "docker compose ps:")
				assert.Nil(t, polled.Runtimes)
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			runPsTestCase(t, tc)
		})
	}
}
