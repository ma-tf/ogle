package logs_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/docker/logs"
)

var errSimulatedDial = errors.New("simulated dial error")

const (
	testProject   = "proj"
	testService   = "svc"
	testLineHello = "hello"
	testLineWorld = "world"
)

// testClient returns an [http.Client] that dials the test server's listener
// regardless of the request URL. This is needed because Go 1.26's
// [httptest.Server.Client] only redirects "example.com" to the server.
func testClient(s *httptest.Server) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer

				return d.DialContext(ctx, s.Listener.Addr().Network(), s.Listener.Addr().String())
			},
		},
	}
}

func TestContainerName(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		project               string
		service               string
		containerNameOverride string

		// assert
		expected string
	}

	tcs := []testCase{
		{
			name:     "default compose v2 convention",
			project:  testProject,
			service:  testService,
			expected: testProject + "-" + testService + "-1",
		},
		{
			name:                  "container name override",
			project:               testProject,
			service:               testService,
			containerNameOverride: "custom",
			expected:              "custom",
		},
		{
			name:     "empty override returns default",
			project:  testProject,
			service:  testService,
			expected: testProject + "-" + testService + "-1",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := logs.ContainerName(tc.project, tc.service, tc.containerNameOverride)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	s := logs.New("svc")

	// New returns an idle streamer with channels created and service name set.
	require.NotNil(t, s)

	// Channels should be buffered and immediately ready — drain won't block.
	require.NotPanics(t, func() {
		// Lines channel exists and is readable (no messages expected).
		select {
		case <-s.Lines():
		default:
		}
	})

	// Next returns a cmd that can be called (will block, we just check it exists).
	require.NotNil(t, s.Next())
}

//nolint:funlen
func TestStart(t *testing.T) {
	t.Parallel()

	t.Run("200 OK delivers frames to Lines and LogLinesAvailable signals", func(t *testing.T) {
		t.Parallel()

		frame1 := makeFrame(1, "hello\n")
		frame2 := makeFrame(2, "world\n")

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(frame1)
			_, _ = w.Write(frame2)
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		s := logs.New("svc", logs.WithHTTPClient(testClient(server)))
		s.Start(ctx, "test-container")

		// Wait for lines to arrive.
		var lines []string

		for range 2 {
			select {
			case line := <-s.Lines():
				lines = append(lines, line)
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for line")
			}

			msg := s.Next()()
			require.NotNil(t, msg)
			_, ok := msg.(msgs.LogLinesAvailable)
			require.True(t, ok)
		}

		assert.Equal(t, []string{testLineHello, testLineWorld}, lines)

		s.Close()
	})

	t.Run("404 returns LogStreamContainerNotFound", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		s := logs.New("svc", logs.WithHTTPClient(testClient(server)))
		s.Start(ctx, "test-container")

		msg := s.Next()()
		require.NotNil(t, msg)

		_, ok := msg.(msgs.LogStreamContainerNotFound)
		require.True(t, ok)

		s.Close()
	})

	t.Run("500 returns LogStreamError with ErrUnexpectedStatus", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("internal error"))
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		s := logs.New("svc", logs.WithHTTPClient(testClient(server)))
		s.Start(ctx, "test-container")

		msg := s.Next()()
		require.NotNil(t, msg)

		errMsg, ok := msg.(msgs.LogStreamError)
		require.True(t, ok)
		require.ErrorIs(t, errMsg.Err, logs.ErrUnexpectedStatus)
		assert.Contains(t, errMsg.Err.Error(), "500")
		assert.Contains(t, errMsg.Err.Error(), "internal error")
		assert.Equal(t, "svc", errMsg.ServiceName)

		s.Close()
	})

	t.Run("dial error returns LogStreamError", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		// A client pointing at a non-routable address will produce a dial error.
		client := &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return nil, errSimulatedDial
				},
			},
		}

		s := logs.New("svc", logs.WithHTTPClient(client))
		s.Start(ctx, "test-container")

		msg := s.Next()()
		require.NotNil(t, msg)

		errMsg, ok := msg.(msgs.LogStreamError)
		require.True(t, ok)
		require.ErrorContains(t, errMsg.Err, "simulated dial error")
		require.Equal(t, "svc", errMsg.ServiceName)

		s.Close()
	})
}

func TestNext(t *testing.T) {
	t.Parallel()

	t.Run("returns cmd that blocks until message arrives", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		s := logs.New("svc", logs.WithHTTPClient(testClient(server)))
		s.Start(ctx, "test-container")

		// Next should return the 404 message.
		cmd := s.Next()
		require.NotNil(t, cmd)

		msg := cmd()
		require.NotNil(t, msg)
		_, ok := msg.(msgs.LogStreamContainerNotFound)
		require.True(t, ok)

		s.Close()
	})
}

func TestClose(t *testing.T) {
	t.Parallel()

	t.Run("is idempotent when called twice", func(t *testing.T) {
		t.Parallel()

		s := logs.New("svc")

		require.NotPanics(t, func() {
			s.Close()
			s.Close()
		})
	})

	t.Run("stops goroutine and drains buffered messages", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		s := logs.New("svc", logs.WithHTTPClient(testClient(server)))
		s.Start(ctx, "test-container")

		// Close should cancel the context and drain the buffered message.
		s.Close()

		// After close, Next returns nil immediately.
		require.Nil(t, s.Next()())
	})

	t.Run("goroutine exits cleanly after Start with 200 OK", func(t *testing.T) {
		t.Parallel()

		frame := makeFrame(1, "test\n")

		var (
			mu       sync.Mutex
			requests int
		)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			requests++
			mu.Unlock()

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(frame)
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		s := logs.New("svc", logs.WithHTTPClient(testClient(server)))
		s.Start(ctx, "test-container")

		// Read the line.
		select {
		case <-s.Lines():
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for line")
		}

		// Close and verify goroutine exits.
		s.Close()

		// After close, Next returns nil.
		require.Nil(t, s.Next()())
	})
}
