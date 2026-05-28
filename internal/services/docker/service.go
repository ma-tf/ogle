// Package docker provides the Docker daemon connectivity layer for ogle.
// The Docker interface abstracts all daemon interactions: connection, state
// polling (Ps), and service actions (Stop, Start, Restart, Rebuild). Service
// is the production adapter; mocks are generated via mockery.
package docker

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
)

// Docker interacts with the Docker daemon for connectivity, state polling,
// inspection, and service actions. All methods return tea.Cmd values that
// the Bubble Tea runtime executes asynchronously.
//
//mockery:generate: true
type Docker interface {
	Connect(ctx context.Context) tea.Cmd
	Ps(ctx context.Context, composeFile, projectName string) tea.Cmd
	Inspect(ctx context.Context, containerID string) tea.Cmd
	Stop(ctx context.Context, composeFile, projectName, serviceName string) tea.Cmd
	Start(ctx context.Context, composeFile, projectName, serviceName string) tea.Cmd
	Restart(ctx context.Context, composeFile, projectName, serviceName string) tea.Cmd
	Rebuild(ctx context.Context, composeFile, projectName, serviceName string) tea.Cmd
}

// Option configures a Service.
type Option func(*Service)

// WithCommander sets the commander on a Service for testing.
func WithCommander(c Commander) Option {
	return func(s *Service) {
		s.commander = c
	}
}

// WithHTTPClient sets the HTTP client used for daemon connectivity.
// The default client dials the Docker Unix socket. Tests supply a client
// pointed at an httptest.NewServer.
func WithHTTPClient(c *http.Client) Option {
	return func(s *Service) {
		s.httpClient = c
	}
}

// Service implements Docker using the Docker Unix socket and docker compose CLI.
type Service struct {
	commander  Commander
	httpClient *http.Client
}

// New returns a Service ready for use.
func New(opts ...Option) *Service {
	transport := &http.Transport{
		DialContext: func(dialCtx context.Context, _, _ string) (net.Conn, error) {
			d := net.Dialer{Timeout: dialTimeout}

			conn, err := d.DialContext(dialCtx, "unix", socketPath)
			if err != nil {
				return nil, fmt.Errorf("dial docker socket: %w", err)
			}

			return conn, nil
		},
	}

	s := &Service{
		commander: realCommander{},
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   requestTimeout,
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

var _ Docker = (*Service)(nil)

var (
	ErrUnexpectedPingStatus    = errors.New("docker ping returned unexpected status")
	ErrUnexpectedInspectStatus = errors.New("inspect container returned unexpected status")
)

const (
	socketPath     = "/var/run/docker.sock"
	pingPath       = "http://localhost/_ping"
	dialTimeout    = 2 * time.Second
	requestTimeout = 5 * time.Second
)

// Connect returns a Cmd that attempts to ping the Docker daemon by issuing an
// HTTP GET to /_ping over the Unix socket at /var/run/docker.sock. On success
// it returns msgs.DaemonConnected; on any failure it returns
// msgs.DaemonUnavailable with a wrapped error.
//
// The context passed to the returned Cmd controls the request lifetime.
// Connect itself does not start a long-running goroutine — callers are
// responsible for scheduling retries.
func (s *Service) Connect(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, pingPath, nil)
		if err != nil {
			return msgs.DaemonUnavailable{Err: fmt.Errorf("build ping request: %w", err)}
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return msgs.DaemonUnavailable{Err: fmt.Errorf("ping docker daemon: %w", err)}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return msgs.DaemonUnavailable{
				Err: fmt.Errorf("%w: %d", ErrUnexpectedPingStatus, resp.StatusCode),
			}
		}

		return msgs.DaemonConnected{}
	}
}
