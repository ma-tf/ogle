package logs_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/docker/logs"
)

// makeFrame builds a single Docker multiplexed log frame:
// 8-byte header (stream + 3 bytes padding + 4-byte big-endian size) + payload.
func makeFrame(stream byte, payload string) []byte {
	buf := make([]byte, 8+len(payload))
	buf[0] = stream
	//nolint:gosec // test helper, payload is small
	binary.BigEndian.PutUint32(buf[4:8], uint32(len(payload)))
	copy(buf[8:], payload)

	return buf
}

// blockThenEOFReader returns data then blocks on ctx cancellation before EOF.
type blockThenEOFReader struct {
	ctx    context.Context
	data   []byte
	offset int
}

func (r *blockThenEOFReader) Read(p []byte) (int, error) {
	if r.offset >= len(r.data) {
		<-r.ctx.Done()

		return 0, fmt.Errorf("blocked read: %w", r.ctx.Err())
	}

	n := copy(p, r.data[r.offset:])
	r.offset += n

	return n, nil
}

func (r *blockThenEOFReader) Close() error { return nil }

func concat(frames ...[]byte) []byte {
	var out []byte
	for _, f := range frames {
		out = append(out, f...)
	}

	return out
}

//nolint:funlen
func TestReadFrames(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		input []byte

		// assert
		expectedLines   []string
		expectedSignals int
		expectedError   error
	}

	tcs := []testCase{
		{
			name:          "empty reader returns EOF",
			expectedError: io.EOF,
		},
		{
			name:            "single stdout frame",
			input:           makeFrame(1, "hello\n"),
			expectedLines:   []string{"hello"},
			expectedSignals: 1,
			expectedError:   io.EOF,
		},
		{
			name:            "single stderr frame",
			input:           makeFrame(2, "error output\n"),
			expectedLines:   []string{"error output"},
			expectedSignals: 1,
			expectedError:   io.EOF,
		},
		{
			name: "consecutive stdout then stderr",
			input: concat(
				makeFrame(1, "stdout line\n"),
				makeFrame(2, "stderr line\n"),
			),
			expectedLines:   []string{"stdout line", "stderr line"},
			expectedSignals: 2,
			expectedError:   io.EOF,
		},
		{
			name:          "truncated header",
			input:         []byte{0x01, 0x00, 0x00},
			expectedError: io.ErrUnexpectedEOF,
		},
		{
			name:            "partial header after first frame",
			input:           concat(makeFrame(1, "ok\n"), []byte{0x01, 0x00}),
			expectedLines:   []string{"ok"},
			expectedSignals: 1,
			expectedError:   io.ErrUnexpectedEOF,
		},
		{
			name:            "zero-length payload",
			input:           makeFrame(1, ""),
			expectedLines:   []string{""},
			expectedSignals: 1,
			expectedError:   io.EOF,
		},
		{
			name: "multiple frames mixed sizes",
			input: concat(
				makeFrame(1, "a\n"),
				makeFrame(1, "bb\n"),
				makeFrame(2, "ccc\n"),
			),
			expectedLines:   []string{"a", "bb", "ccc"},
			expectedSignals: 3,
			expectedError:   io.EOF,
		},
		{
			name:            "trims all trailing newlines",
			input:           makeFrame(1, "hello\n\n"),
			expectedLines:   []string{"hello"},
			expectedSignals: 1,
			expectedError:   io.EOF,
		},
		{
			name:            "does not trim internal newlines",
			input:           makeFrame(1, "line1\nline2\n"),
			expectedLines:   []string{"line1\nline2"},
			expectedSignals: 1,
			expectedError:   io.EOF,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			lines := make(chan string, tc.expectedSignals+1)
			signals := make(chan tea.Msg, tc.expectedSignals+1)

			r := io.NopCloser(strings.NewReader(string(tc.input)))

			errCh := make(chan error, 1)
			go func() {
				errCh <- logs.ReadFrames(ctx, r, lines, signals, "")
			}()

			var gotLines []string

			for range tc.expectedSignals {
				select {
				case line := <-lines:
					gotLines = append(gotLines, line)
				case <-ctx.Done():
					return
				}

				select {
				case sig := <-signals:
					_, ok := sig.(msgs.LogLinesAvailable)
					require.True(t, ok, "expected LogLinesAvailable signal")
				case <-ctx.Done():
					return
				}
			}

			err := <-errCh

			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.expectedLines, gotLines)
		})
	}
}

func TestReadFramesContextCancel(t *testing.T) {
	t.Parallel()

	t.Run("context cancelled before any read", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		lines := make(chan string, 1)
		signals := make(chan tea.Msg, 1)

		frame := makeFrame(1, "hello\n")
		r := io.NopCloser(strings.NewReader(string(frame)))

		errCh := make(chan error, 1)
		go func() {
			errCh <- logs.ReadFrames(ctx, r, lines, signals, "")
		}()

		require.ErrorIs(t, <-errCh, context.Canceled)
	})

	t.Run("context cancelled during header read", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		lines := make(chan string, 1)
		signals := make(chan tea.Msg, 1)

		// Reader provides partial header, then blocks until cancelled.
		partialHeader := makeFrame(1, "hello\n")[:4]
		r := &blockThenEOFReader{ctx: ctx, data: partialHeader}

		errCh := make(chan error, 1)
		go func() {
			errCh <- logs.ReadFrames(ctx, r, lines, signals, "")
		}()

		cancel()

		require.ErrorIs(t, <-errCh, context.Canceled)
	})

	t.Run("context cancelled during payload read", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		lines := make(chan string, 1)
		signals := make(chan tea.Msg, 1)

		fullFrame := makeFrame(1, "hello\n")
		// Provide header but stop short of full payload.
		headerAndPartial := fullFrame[:10]
		r := &blockThenEOFReader{ctx: ctx, data: headerAndPartial}

		errCh := make(chan error, 1)
		go func() {
			errCh <- logs.ReadFrames(ctx, r, lines, signals, "")
		}()

		cancel()

		require.ErrorIs(t, <-errCh, context.Canceled)
	})

	t.Run("context cancelled during channel send", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Unbuffered lines channel — ReadFrames will block on send.
		lines := make(chan string)
		signals := make(chan tea.Msg, 1)

		frame := makeFrame(1, "hello\n")
		r := io.NopCloser(strings.NewReader(string(frame)))

		errCh := make(chan error, 1)
		go func() {
			errCh <- logs.ReadFrames(ctx, r, lines, signals, "")
		}()

		// Cancel while ReadFrames is blocked on lines <- "hello".
		cancel()

		require.ErrorIs(t, <-errCh, context.Canceled)
	})
}
