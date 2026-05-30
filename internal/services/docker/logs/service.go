// Package logs implements the LogStreamer service, which streams Docker
// container log output over the Docker Unix socket using raw net/http.
package logs

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
)

const (
	socketPath     = "/var/run/docker.sock"
	channelCap     = 64
	lineBufferCap  = 5000
	tailLines      = "1000"
	errorBodyLimit = 512
)

var ErrUnexpectedStatus = errors.New("logs: unexpected status")

// ContainerName resolves the Docker container name for a service.
// If containerNameOverride is non-empty it is returned verbatim; otherwise the
// Compose v2 convention "<project>-<service>-1" is used.
//
// Compose v1 used underscores (project_service_1); v2 uses dashes. Only v2 is supported.
func ContainerName(project, service, containerNameOverride string) string {
	if containerNameOverride != "" {
		return containerNameOverride
	}

	return project + "-" + service + "-1"
}

// Option configures a LogStreamer.
type Option func(*LogStreamer)

// WithHTTPClient sets the HTTP client used for Docker API requests.
// If not set, a default client dialing /var/run/docker.sock is used.
func WithHTTPClient(client *http.Client) Option {
	return func(s *LogStreamer) {
		s.client = client
	}
}

// LogStreamer streams Docker container logs over the Unix socket. Normal log
// lines flow through lineCh; errors flow through ch.
type LogStreamer struct {
	client      *http.Client
	cancel      context.CancelFunc
	ch          chan tea.Msg
	lineCh      chan string
	done        chan struct{}
	wg          sync.WaitGroup
	serviceName string
}

// New returns an idle LogStreamer. Call Start before calling Next.
// Pass WithHTTPClient to inject a custom HTTP client for testing.
func New(serviceName string, opts ...Option) *LogStreamer {
	s := &LogStreamer{
		client:      nil,
		cancel:      nil,
		ch:          make(chan tea.Msg, channelCap),
		lineCh:      make(chan string, lineBufferCap),
		done:        make(chan struct{}),
		wg:          sync.WaitGroup{},
		serviceName: serviceName,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Start begins streaming logs for the named container. Close must be called
// before calling Start again on a reused LogStreamer. Start may be called
// after Close — the same channels are reused. It returns immediately;
// the HTTP connection and all I/O run in a single background goroutine. On
// 404 or non-200 the goroutine writes a single error message to ch and exits.
// On 200 it writes log line strings to lineCh until the stream ends or Close
// is called.
func (s *LogStreamer) Start(appCtx context.Context, containerName string) {
	ctx, cancel := context.WithCancel(appCtx)
	s.cancel = cancel
	s.done = make(chan struct{})

	s.wg.Go(func() {
		s.startStream(ctx, containerName)
	})
}

// startStream runs in a background goroutine. It connects to the Docker socket,
// requests the log stream, and dispatches log lines or error messages.
func (s *LogStreamer) startStream(ctx context.Context, containerName string) {
	client := s.client
	if client == nil {
		transport := &http.Transport{
			DialContext: func(dialCtx context.Context, _, _ string) (net.Conn, error) {
				d := net.Dialer{}

				return d.DialContext(dialCtx, "unix", socketPath)
			},
		}
		client = &http.Client{Transport: transport}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf(
			"http://localhost/containers/%s/logs?follow=true&stdout=1&stderr=1&tail=%s",
			url.PathEscape(containerName), tailLines,
		),
		nil,
	)
	if err != nil {
		s.sendError(ctx, err)

		return
	}

	resp, err := client.Do(req)
	if err != nil {
		s.sendError(ctx, err)

		return
	}

	if resp.StatusCode == http.StatusNotFound {
		_ = resp.Body.Close()

		client.CloseIdleConnections()

		s.sendContainerNotFound(ctx)

		return
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, errorBodyLimit))
		_ = resp.Body.Close()

		client.CloseIdleConnections()

		s.sendError(ctx, fmt.Errorf("%w %d: %s", ErrUnexpectedStatus, resp.StatusCode, body))

		return
	}

	defer client.CloseIdleConnections()

	err = ReadFrames(ctx, resp.Body, s.lineCh, s.ch)
	if err != nil && ctx.Err() == nil {
		s.sendError(ctx, err)
	}
}

func (s *LogStreamer) sendError(ctx context.Context, err error) {
	select {
	case s.ch <- msgs.LogStreamError{Err: err, ServiceName: s.serviceName}:
	case <-ctx.Done():
	}
}

func (s *LogStreamer) sendContainerNotFound(ctx context.Context) {
	select {
	case s.ch <- msgs.LogStreamContainerNotFound{ServiceName: s.serviceName}:
	case <-ctx.Done():
	}
}

// ReadFrames reads the Docker multiplexed log stream from r until the context
// is cancelled or the reader is exhausted. Each log line is trimmed of a
// trailing newline and sent to lines; a LogLinesAvailable signal is sent to
// signals after each line. Read errors are returned — the caller is responsible
// for wrapping them into msgs.LogStreamError. The reader r is closed before
// ReadFrames returns.
func ReadFrames(
	ctx context.Context,
	r io.ReadCloser,
	lines chan<- string,
	signals chan<- tea.Msg,
) error {
	defer r.Close()

	for {
		var header [8]byte

		if _, readErr := io.ReadFull(r, header[:]); readErr != nil {
			select {
			case <-ctx.Done():
				return ctx.Err() //nolint:wrapcheck // sentinel errors, callers check via errors.Is
			default:
				return readErr //nolint:wrapcheck // std lib errors, callers check via errors.Is
			}
		}

		size := binary.BigEndian.Uint32(header[4:])
		payload := make([]byte, size)

		if _, readErr := io.ReadFull(r, payload); readErr != nil {
			select {
			case <-ctx.Done():
				return ctx.Err() //nolint:wrapcheck // sentinel errors, callers check via errors.Is
			default:
				return readErr //nolint:wrapcheck // std lib errors, callers check via errors.Is
			}
		}

		select {
		case lines <- strings.TrimRight(string(payload), "\n"):
			select {
			case signals <- msgs.LogLinesAvailable{}:
			case <-ctx.Done():
				return ctx.Err() //nolint:wrapcheck // sentinel errors, callers check via errors.Is
			}
		case <-ctx.Done():
			return ctx.Err() //nolint:wrapcheck // sentinel errors, callers check via errors.Is
		}
	}
}

// Lines returns a read-only channel that delivers log line strings. The
// channel is closed in Close.
func (s *LogStreamer) Lines() <-chan string {
	return s.lineCh
}

// Next returns a tea.Cmd that blocks until the next message arrives on the
// internal channel. The caller must call Next again after each received message
// to re-subscribe — there is no automatic re-subscription. If the streamer is
// closed while Next is in-flight, the cmd returns nil and the goroutine exits.
func (s *LogStreamer) Next() tea.Cmd {
	done := s.done

	return func() tea.Msg {
		select {
		case msg := <-s.ch:
			return msg
		case <-done:
			return nil
		}
	}
}

// Close cancels the streaming context, waits for the reader goroutine to exit,
// rotates the done channel to unblock any in-flight Next cmd, then drains any
// buffered error messages. Unlike Start, Close does not close the line channel
// — the LogStreamer may be restarted with a subsequent Start call. Close is a
// no-op when no stream is active.
func (s *LogStreamer) Close() {
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil

		s.wg.Wait()

		old := s.done
		s.done = make(chan struct{})

		close(old)
		close(s.done)

		for {
			select {
			case <-s.ch:
			default:
				return
			}
		}
	}
}
