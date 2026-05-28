package docker_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	svcdocker "github.com/ma-tf/ogle/internal/services/docker"
)

// testServerClient returns an [http.Client] whose transport always dials the
// given test server, regardless of the URL in the request. This lets us send
// requests with the production pingPath (http://localhost/_ping) while still
// routing them to the test server.
func testServerClient(srv *httptest.Server) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer

				return d.DialContext(
					ctx,
					srv.Listener.Addr().Network(),
					srv.Listener.Addr().String(),
				)
			},
		},
	}
}

func TestConnect(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name    string
		handler http.HandlerFunc
		// arrange
		closeServer bool
		ctx         func() context.Context
		// assert
		expectedConnected   bool
		expectedErrWrapped  error
		expectedErrContains string
	}

	tt := []testCase{
		{
			name: "200 ok",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			ctx:               context.Background,
			expectedConnected: true,
		},
		{
			name: "non-200 status",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			ctx:                context.Background,
			expectedConnected:  false,
			expectedErrWrapped: svcdocker.ErrUnexpectedPingStatus,
		},
		{
			name: "dial error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			closeServer:         true,
			ctx:                 context.Background,
			expectedConnected:   false,
			expectedErrContains: "connect",
		},
		{
			name: "nil context",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			ctx: func() context.Context {
				return nil
			},
			expectedConnected:   false,
			expectedErrContains: "build ping request",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(tc.handler)
			t.Cleanup(srv.Close)

			client := testServerClient(srv)

			if tc.closeServer {
				srv.Close()
			}

			svc := svcdocker.New(svcdocker.WithHTTPClient(client))
			cmd := svc.Connect(tc.ctx())
			require.NotNil(t, cmd)

			msg := cmd()
			require.NotNil(t, msg)

			if tc.expectedConnected {
				_, ok := msg.(msgs.DaemonConnected)
				require.True(t, ok, "expected DaemonConnected, got %T", msg)
			} else {
				daemonUnavailable, ok := msg.(msgs.DaemonUnavailable)
				require.True(t, ok, "expected DaemonUnavailable, got %T", msg)
				require.Error(t, daemonUnavailable.Err)

				if tc.expectedErrWrapped != nil {
					require.ErrorIs(t, daemonUnavailable.Err, tc.expectedErrWrapped)
				}

				if tc.expectedErrContains != "" {
					require.ErrorContains(t, daemonUnavailable.Err, tc.expectedErrContains)
				}
			}
		})
	}
}

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

			result, err := svcdocker.ParsePsOutput(tc.input)

			if tc.name == "malformed json" {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Len(t, result, tc.expectedLen)
		})
	}
}

func TestInspect(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		handler     http.HandlerFunc
		closeServer bool
		// assert
		expectedLabels map[string]string
		expectedErr    bool
	}

	tt := []testCase{
		{
			name: "returns ogle labels from container inspect",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"Config": {
						"Labels": {
							"ogle.foo": "bar",
							"ogle.environment": "production",
							"com.docker.compose.project": "myproject"
						}
					}
				}`))
			},
			expectedLabels: map[string]string{
				"ogle.foo":         "bar",
				"ogle.environment": "production",
			},
		},
		{
			name: "no ogle labels returns nil",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"Config": {
						"Labels": {
							"com.docker.compose.project": "myproject"
						}
					}
				}`))
			},
			expectedLabels: nil,
		},
		{
			name: "empty labels returns nil",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"Config": {"Labels": {}}}`))
			},
			expectedLabels: nil,
		},
		{
			name: "nil labels returns nil",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"Config": {}}`))
			},
			expectedLabels: nil,
		},
		{
			name: "non-200 status returns error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectedErr: true,
		},
		{
			name: "dial error returns error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			closeServer:  true,
			expectedErr:  true,
		},
		{
			name: "malformed json returns error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{invalid}`))
			},
			expectedErr: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(tc.handler)
			t.Cleanup(srv.Close)

			client := testServerClient(srv)

			if tc.closeServer {
				srv.Close()
			}

			svc := svcdocker.New(svcdocker.WithHTTPClient(client))
			cmd := svc.Inspect(context.Background(), "abc123")
			require.NotNil(t, cmd)

			msg := cmd()
			require.NotNil(t, msg)

			labelsMsg, ok := msg.(msgs.ContainerLabelsPolled)
			require.True(t, ok, "expected ContainerLabelsPolled, got %T", msg)

			if tc.expectedErr {
				require.Error(t, labelsMsg.Err)
				require.Nil(t, labelsMsg.Labels)
			} else {
				require.NoError(t, labelsMsg.Err)
				assert.Equal(t, tc.expectedLabels, labelsMsg.Labels)
			}
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

			result, err := svcdocker.ParsePsOutput(tc.input)
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

			result := svcdocker.ParseState(tc.input)
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
	s := svcdocker.New(svcdocker.WithCommander(fc))

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
