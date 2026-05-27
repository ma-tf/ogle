// Package watcher monitors a directory for changes to known compose filenames
// and delivers snapshot messages to the Bubble Tea runtime via tea.Cmd.
package watcher

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/fsnotify/fsnotify"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/scanner"
)

// ErrCreateWatcher is returned when watcher initialisation fails.
var ErrCreateWatcher = errors.New("create watcher")

// Watcher monitors a directory for filesystem changes and delivers
// msgs.FileAvailabilityChanged snapshots to the Bubble Tea runtime.
//
//mockery:generate: true
type Watcher interface {
	// Dir returns the directory being monitored.
	Dir() string
	// Start begins the background event loop. sendMsg routes non-domain
	// messages (e.g. errors) into the Bubble Tea event loop. Must be
	// called once after the tea.Program is created.
	Start(sendMsg func(tea.Msg))
	// Next returns a tea.Cmd that blocks until the next snapshot is ready.
	// Re-call after each FileAvailabilityChanged to continue listening.
	Next() tea.Cmd
	// Snapshot returns a tea.Cmd that delivers the current filesystem state
	// as a msgs.FileAvailabilityChanged without waiting for a change event.
	Snapshot() tea.Cmd
	// Close stops the background goroutine and releases resources.
	Close() error
}

// FileWatcher abstracts a filesystem watcher so the event loop is testable
// without a real filesystem. The production implementation wraps
// *fsnotify.Watcher.
type FileWatcher interface {
	Events() chan fsnotify.Event
	Errors() chan error
	Add(name string) error
	Close() error
}

// realFileWatcher wraps *fsnotify.Watcher to implement FileWatcher.
type realFileWatcher struct {
	w *fsnotify.Watcher
}

func (r *realFileWatcher) Events() chan fsnotify.Event { return r.w.Events }
func (r *realFileWatcher) Errors() chan error          { return r.w.Errors }

func (r *realFileWatcher) Add(name string) error {
	if err := r.w.Add(name); err != nil {
		return fmt.Errorf("real file watcher add: %w", err)
	}

	return nil
}

func (r *realFileWatcher) Close() error {
	if err := r.w.Close(); err != nil {
		return fmt.Errorf("real file watcher close: %w", err)
	}

	return nil
}

// Service monitors a directory for filesystem events on known compose
// filenames and delivers msgs.FileAvailabilityChanged snapshots to the
// Bubble Tea runtime. extraFile is an additional basename to track beyond
// the scanner's known filenames (e.g. a manually specified project file).
// When empty, only the scanner's known filenames are tracked.
type Service struct {
	fw        FileWatcher
	dir       string
	scanner   scanner.Scanner
	sendMsg   func(tea.Msg)
	extraFile string
	events    chan tea.Msg
	done      chan struct{}
	once      sync.Once
	startOnce sync.Once
}

// New creates a Watcher that monitors dir. extraFile is an additional
// basename to track (e.g. a manually specified project file). Pass "" to
// only track the scanner's known filenames. On failure, nil is returned
// alongside the error. Call Start to begin the background event loop.
func New(dir string, extraFile string, sc scanner.Scanner) (Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("%w: fsnotify: %w", ErrCreateWatcher, err)
	}

	return NewWithFileWatcher(dir, extraFile, sc, &realFileWatcher{w: fw})
}

// NewWithFileWatcher creates a Watcher with the given FileWatcher
// implementation. Intended for testing with a fake FileWatcher; production
// code should use New instead. Call Start to begin the background event loop.
func NewWithFileWatcher(
	dir string,
	extraFile string,
	sc scanner.Scanner,
	fw FileWatcher,
) (Watcher, error) {
	if addErr := fw.Add(dir); addErr != nil {
		closeErr := fw.Close()

		return nil, fmt.Errorf("%w: add watch: %w", ErrCreateWatcher, errors.Join(addErr, closeErr))
	}

	w := &Service{
		fw:      fw,
		dir:     dir,
		scanner: sc,
		extraFile: func() string {
			if extraFile == "" {
				return ""
			}

			return filepath.Base(extraFile)
		}(),
		events:    make(chan tea.Msg, 1),
		done:      make(chan struct{}),
		once:      sync.Once{},
		sendMsg:   nil,
		startOnce: sync.Once{},
	}

	return w, nil
}

// Dir returns the directory this Service monitors.
func (w *Service) Dir() string {
	return w.dir
}

// Close stops the background event loop and releases the underlying fsnotify
// watcher. Safe to call more than once; subsequent calls are no-ops.
func (w *Service) Close() error {
	var fwErr error

	w.once.Do(func() {
		close(w.done)
		fwErr = w.fw.Close()
		w.sendMsg = nil
	})

	if fwErr != nil {
		return fmt.Errorf("close fsnotify watcher: %w", fwErr)
	}

	return nil
}

// Next returns a tea.Cmd that blocks until the next availability snapshot is
// ready and returns it as a msgs.FileAvailabilityChanged. After receiving a
// message in Update, call Next again to continue listening. Returns nil if the
// watcher is closed.
func (w *Service) Next() tea.Cmd {
	return func() tea.Msg {
		select {
		case msg := <-w.events:
			return msg
		case <-w.done:
			return nil
		}
	}
}

// scan returns all known compose files and the extra file (if set) that
// currently exist in the monitored directory.
func (w *Service) scan() []string {
	files := w.scanner.ScanAll(w.dir)

	if w.extraFile != "" {
		path := filepath.Join(w.dir, w.extraFile)

		if _, err := os.Stat(path); err == nil && !slices.Contains(files, path) {
			files = append(files, path)
		}
	}

	return files
}

// Snapshot returns a tea.Cmd that delivers the current filesystem state as a
// msgs.FileAvailabilityChanged without waiting for a change event. Extra
// files passed to New are included in the scan.
func (w *Service) Snapshot() tea.Cmd {
	return func() tea.Msg {
		return msgs.FileAvailabilityChanged{Files: w.scan()}
	}
}

// Start begins the background event loop. sendMsg routes non-domain
// messages (e.g. fsnotify errors) into the Bubble Tea event loop. Must be
// called once after the tea.Program is created. Safe to call multiple
// times; subsequent calls are no-ops.
func (w *Service) Start(sendMsg func(tea.Msg)) {
	w.startOnce.Do(func() {
		w.sendMsg = sendMsg
		go w.run()
	})
}

// run is the background goroutine that processes fsnotify events.
func (w *Service) run() {
	for {
		select {
		case <-w.done:
			return

		case event, ok := <-w.fw.Events():
			if !ok {
				return
			}

			snapshot := w.handleEvent(event)
			if snapshot == nil {
				continue
			}

			if !w.sendSnapshot(*snapshot) {
				return
			}

		case err, ok := <-w.fw.Errors():
			if !ok {
				return
			}

			w.sendMsg(msgs.DisplayError{
				Err: fmt.Sprintf("watcher: fsnotify error for %s: %v", w.dir, err),
			})
		}
	}
}

// handleEvent returns a snapshot for the event, or nil if the event should be
// ignored (e.g. Chmod which does not affect file availability).
func (w *Service) handleEvent(event fsnotify.Event) *msgs.FileAvailabilityChanged {
	if event.Op == fsnotify.Chmod {
		return nil
	}

	known := slices.Contains(w.scanner.KnownFilenames(), filepath.Base(event.Name))
	if !known && filepath.Base(event.Name) != w.extraFile {
		return nil
	}

	snapshot := msgs.FileAvailabilityChanged{
		Files: w.scan(),
	}

	return &snapshot
}

// sendSnapshot drains any stale snapshot and sends the current one. Returns
// false if the watcher has been closed during the send.
func (w *Service) sendSnapshot(snapshot msgs.FileAvailabilityChanged) bool {
	select {
	case <-w.events:
	default:
	}

	select {
	case w.events <- snapshot:
		return true
	case <-w.done:
		return false
	}
}
