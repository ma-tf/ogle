package watcher_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/scanner/mocks"
	"github.com/ma-tf/ogle/internal/services/watcher"
)

const (
	knownFilename = "compose.yml"
	knownFilePath = "/dir/compose.yml"
)

// fakeFileWatcher implements watcher.FileWatcher with programmable channels
// and error injection for deterministic testing of the event loop.
type fakeFileWatcher struct {
	eventsCh chan fsnotify.Event
	errorsCh chan error

	mu       sync.Mutex
	addCalls []string
	addErr   error
	closeErr error
	closed   bool
}

func newFakeFileWatcher() *fakeFileWatcher {
	return &fakeFileWatcher{
		eventsCh: make(chan fsnotify.Event),
		errorsCh: make(chan error),
	}
}

func (f *fakeFileWatcher) Events() chan fsnotify.Event { return f.eventsCh }
func (f *fakeFileWatcher) Errors() chan error          { return f.errorsCh }

func (f *fakeFileWatcher) Add(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.addCalls = append(f.addCalls, name)

	return f.addErr
}

func (f *fakeFileWatcher) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.closed = true

	return f.closeErr
}

func TestNewWithFileWatcher_AddCalled(t *testing.T) {
	t.Parallel()

	fw := newFakeFileWatcher()
	sc := mocks.NewMockScanner(t)
	sc.EXPECT().KnownFilenames().Return([]string{knownFilename}).Maybe()

	dir := t.TempDir()
	w, err := watcher.NewWithFileWatcher(dir, "", sc, fw)
	require.NoError(t, err)

	defer w.Close()

	fw.mu.Lock()
	assert.Equal(t, []string{dir}, fw.addCalls)
	fw.mu.Unlock()
}

func TestRun_EventCreate(t *testing.T) {
	t.Parallel()

	fw := newFakeFileWatcher()
	sc := mocks.NewMockScanner(t)
	sc.EXPECT().KnownFilenames().Return([]string{knownFilename}).Maybe()
	sc.EXPECT().ScanAll(mock.Anything).Return([]string{knownFilePath}).Once()

	w, err := watcher.NewWithFileWatcher("/dir", "", sc, fw)
	require.NoError(t, err)

	w.Start(func(tea.Msg) {})

	defer w.Close()

	fw.eventsCh <- fsnotify.Event{Name: knownFilePath, Op: fsnotify.Create}

	result := w.Next()()
	require.NotNil(t, result)
	msg, ok := result.(msgs.FileAvailabilityChanged)
	require.True(t, ok)
	require.Equal(t, []string{knownFilePath}, msg.Files)
}

func TestRun_EventWrite(t *testing.T) {
	t.Parallel()

	fw := newFakeFileWatcher()
	sc := mocks.NewMockScanner(t)
	sc.EXPECT().KnownFilenames().Return([]string{knownFilename}).Maybe()
	sc.EXPECT().ScanAll(mock.Anything).Return([]string{knownFilePath}).Once()

	w, err := watcher.NewWithFileWatcher("/dir", "", sc, fw)
	require.NoError(t, err)

	w.Start(func(tea.Msg) {})

	defer w.Close()

	fw.eventsCh <- fsnotify.Event{Name: knownFilePath, Op: fsnotify.Write}

	result := w.Next()()
	require.NotNil(t, result)
	_, ok := result.(msgs.FileAvailabilityChanged)
	require.True(t, ok)
}

func TestRun_EventRemove(t *testing.T) {
	t.Parallel()

	fw := newFakeFileWatcher()
	sc := mocks.NewMockScanner(t)
	sc.EXPECT().KnownFilenames().Return([]string{knownFilename}).Maybe()
	sc.EXPECT().ScanAll(mock.Anything).Return(nil).Once()

	w, err := watcher.NewWithFileWatcher("/dir", "", sc, fw)
	require.NoError(t, err)

	w.Start(func(tea.Msg) {})

	defer w.Close()

	fw.eventsCh <- fsnotify.Event{Name: knownFilePath, Op: fsnotify.Remove}

	result := w.Next()()
	require.NotNil(t, result)
	msg, ok := result.(msgs.FileAvailabilityChanged)
	require.True(t, ok)
	require.Empty(t, msg.Files)
}

func TestRun_MultipleEvents(t *testing.T) {
	t.Parallel()

	fw := newFakeFileWatcher()
	sc := mocks.NewMockScanner(t)
	sc.EXPECT().KnownFilenames().Return([]string{knownFilename}).Maybe()
	sc.EXPECT().ScanAll(mock.Anything).Return([]string{knownFilePath}).Times(2)

	w, err := watcher.NewWithFileWatcher("/dir", "", sc, fw)
	require.NoError(t, err)

	w.Start(func(tea.Msg) {})

	defer w.Close()

	fw.eventsCh <- fsnotify.Event{Name: knownFilePath, Op: fsnotify.Create}

	msg1 := w.Next()()
	require.NotNil(t, msg1)
	_, ok := msg1.(msgs.FileAvailabilityChanged)
	require.True(t, ok)

	fw.eventsCh <- fsnotify.Event{Name: knownFilePath, Op: fsnotify.Create}

	msg2 := w.Next()()
	require.NotNil(t, msg2)
	_, ok = msg2.(msgs.FileAvailabilityChanged)
	require.True(t, ok)
}

func TestRun_ErrorReceived(t *testing.T) {
	t.Parallel()

	fw := newFakeFileWatcher()
	sc := mocks.NewMockScanner(t)
	sc.EXPECT().KnownFilenames().Return([]string{knownFilename}).Maybe()
	sc.EXPECT().ScanAll(mock.Anything).Return([]string{knownFilePath}).Once()

	var gotErr string

	w, err := watcher.NewWithFileWatcher("/dir", "", sc, fw)
	require.NoError(t, err)

	w.Start(func(msg tea.Msg) {
		if em, ok := msg.(msgs.DisplayError); ok {
			gotErr = em.Err
		}
	})

	defer w.Close()

	fw.errorsCh <- assert.AnError

	fw.eventsCh <- fsnotify.Event{Name: knownFilePath, Op: fsnotify.Create}

	result := w.Next()()
	require.NotNil(t, result)
	_, ok := result.(msgs.FileAvailabilityChanged)
	require.True(t, ok)

	require.Eventually(t, func() bool {
		return gotErr != ""
	}, time.Second, 10*time.Millisecond, "expected DisplayError to be delivered")
}

func TestRun_FilteredEvents(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		event fsnotify.Event
	}

	cases := []testCase{
		{
			name:  "Chmod",
			event: fsnotify.Event{Name: knownFilePath, Op: fsnotify.Chmod},
		},
		{
			name:  "UnknownFile",
			event: fsnotify.Event{Name: "/dir/unknown.txt", Op: fsnotify.Create},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fw := newFakeFileWatcher()
			sc := mocks.NewMockScanner(t)
			sc.EXPECT().KnownFilenames().Return([]string{knownFilename}).Maybe()
			sc.EXPECT().ScanAll(mock.Anything).Return([]string{}).Once()

			w, err := watcher.NewWithFileWatcher("/dir", "", sc, fw)
			require.NoError(t, err)

			w.Start(func(tea.Msg) {})

			defer w.Close()

			fw.eventsCh <- tc.event

			fw.eventsCh <- fsnotify.Event{Name: knownFilePath, Op: fsnotify.Create}

			result := w.Next()()
			require.NotNil(t, result)
			_, ok := result.(msgs.FileAvailabilityChanged)
			require.True(t, ok)
		})
	}
}

func TestRun_CloseTerminates(t *testing.T) {
	t.Parallel()

	fw := newFakeFileWatcher()
	sc := mocks.NewMockScanner(t)
	sc.EXPECT().KnownFilenames().Return([]string{knownFilename}).Maybe()

	w, err := watcher.NewWithFileWatcher("/dir", "", sc, fw)
	require.NoError(t, err)

	require.NoError(t, w.Close())

	fw.mu.Lock()
	assert.True(t, fw.closed)
	fw.mu.Unlock()

	result := w.Next()()
	require.Nil(t, result)
}

func TestSnapshot(t *testing.T) {
	t.Parallel()

	fw := newFakeFileWatcher()
	sc := mocks.NewMockScanner(t)
	sc.EXPECT().KnownFilenames().Return([]string{knownFilename}).Maybe()
	sc.EXPECT().ScanAll("/dir").Return([]string{knownFilePath})

	w, err := watcher.NewWithFileWatcher("/dir", "", sc, fw)
	require.NoError(t, err)

	defer w.Close()

	result := w.Snapshot()()
	require.NotNil(t, result)
	msg, ok := result.(msgs.FileAvailabilityChanged)
	require.True(t, ok)
	require.Equal(t, []string{knownFilePath}, msg.Files)
}

func TestSnapshot_WithExtraFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	extraFile := filepath.Join(dir, "custom.yml")
	require.NoError(t, os.WriteFile(extraFile, []byte("services:\n  test:\n"), 0o600))

	fw := newFakeFileWatcher()
	sc := mocks.NewMockScanner(t)
	sc.EXPECT().KnownFilenames().Return([]string{knownFilename}).Maybe()
	sc.EXPECT().ScanAll(dir).Return(nil)

	w, err := watcher.NewWithFileWatcher(dir, extraFile, sc, fw)
	require.NoError(t, err)

	defer w.Close()

	result := w.Snapshot()()
	require.NotNil(t, result)
	msg, ok := result.(msgs.FileAvailabilityChanged)
	require.True(t, ok)
	require.Equal(t, []string{extraFile}, msg.Files)
}

func TestDir(t *testing.T) {
	t.Parallel()

	fw := newFakeFileWatcher()
	sc := mocks.NewMockScanner(t)
	sc.EXPECT().KnownFilenames().Return([]string{knownFilename}).Maybe()

	w, err := watcher.NewWithFileWatcher("/mydir", "", sc, fw)
	require.NoError(t, err)

	defer w.Close()

	assert.Equal(t, "/mydir", w.Dir())
}

func TestNext_ReturnsNilAfterClose(t *testing.T) {
	t.Parallel()

	fw := newFakeFileWatcher()
	sc := mocks.NewMockScanner(t)
	sc.EXPECT().KnownFilenames().Return([]string{knownFilename}).Maybe()

	w, err := watcher.NewWithFileWatcher("/dir", "", sc, fw)
	require.NoError(t, err)

	require.NoError(t, w.Close())

	require.Nil(t, w.Next()())
}

func TestClose_Idempotent(t *testing.T) {
	t.Parallel()

	fw := newFakeFileWatcher()
	sc := mocks.NewMockScanner(t)
	sc.EXPECT().KnownFilenames().Return([]string{knownFilename}).Maybe()

	w, err := watcher.NewWithFileWatcher("/dir", "", sc, fw)
	require.NoError(t, err)

	require.NoError(t, w.Close())
	require.NoError(t, w.Close())

	fw.mu.Lock()
	assert.True(t, fw.closed)
	fw.mu.Unlock()
}

func TestNewWithFileWatcher_AddError(t *testing.T) {
	t.Parallel()

	fw := newFakeFileWatcher()
	fw.addErr = assert.AnError
	sc := mocks.NewMockScanner(t)

	w, err := watcher.NewWithFileWatcher("/dir", "", sc, fw)
	require.ErrorIs(t, err, watcher.ErrCreateWatcher)
	require.Nil(t, w)
	require.True(t, fw.closed)
}

func TestClose_Error(t *testing.T) {
	t.Parallel()

	fw := newFakeFileWatcher()
	fw.closeErr = assert.AnError
	sc := mocks.NewMockScanner(t)
	sc.EXPECT().KnownFilenames().Return([]string{knownFilename}).Maybe()

	w, err := watcher.NewWithFileWatcher("/dir", "", sc, fw)
	require.NoError(t, err)

	err = w.Close()
	require.Error(t, err)
	require.ErrorContains(t, err, "close fsnotify watcher")
}

func TestRun_EventsChannelClosed(t *testing.T) {
	t.Parallel()

	fw := newFakeFileWatcher()
	sc := mocks.NewMockScanner(t)
	sc.EXPECT().KnownFilenames().Return([]string{knownFilename}).Maybe()

	w, err := watcher.NewWithFileWatcher("/dir", "", sc, fw)
	require.NoError(t, err)

	w.Start(func(tea.Msg) {})

	close(fw.eventsCh)

	require.NoError(t, w.Close())
	fw.mu.Lock()
	assert.True(t, fw.closed)
	fw.mu.Unlock()
}

func TestRun_ErrorsChannelClosed(t *testing.T) {
	t.Parallel()

	fw := newFakeFileWatcher()
	sc := mocks.NewMockScanner(t)
	sc.EXPECT().KnownFilenames().Return([]string{knownFilename}).Maybe()

	w, err := watcher.NewWithFileWatcher("/dir", "", sc, fw)
	require.NoError(t, err)

	w.Start(func(tea.Msg) {})

	close(fw.errorsCh)

	require.NoError(t, w.Close())
	fw.mu.Lock()
	assert.True(t, fw.closed)
	fw.mu.Unlock()
}

func TestRun_ExtraFileEvent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	extraFile := filepath.Join(dir, "custom.yml")
	require.NoError(t, os.WriteFile(extraFile, []byte("services:\n  test:\n"), 0o600))

	fw := newFakeFileWatcher()
	sc := mocks.NewMockScanner(t)
	sc.EXPECT().KnownFilenames().Return([]string{knownFilename}).Maybe()
	sc.EXPECT().ScanAll(dir).Return(nil).Once()

	w, err := watcher.NewWithFileWatcher(dir, extraFile, sc, fw)
	require.NoError(t, err)

	w.Start(func(tea.Msg) {})

	defer w.Close()

	fw.eventsCh <- fsnotify.Event{Name: extraFile, Op: fsnotify.Create}

	result := w.Next()()
	require.NotNil(t, result)
	msg, ok := result.(msgs.FileAvailabilityChanged)
	require.True(t, ok)
	require.Equal(t, []string{extraFile}, msg.Files)
}

func TestRun_ExtraFileEvent_AbsentFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	extraFile := filepath.Join(dir, "custom.yml")

	fw := newFakeFileWatcher()
	sc := mocks.NewMockScanner(t)
	sc.EXPECT().KnownFilenames().Return([]string{knownFilename}).Maybe()
	sc.EXPECT().ScanAll(dir).Return(nil).Once()

	w, err := watcher.NewWithFileWatcher(dir, extraFile, sc, fw)
	require.NoError(t, err)

	w.Start(func(tea.Msg) {})

	defer w.Close()

	fw.eventsCh <- fsnotify.Event{Name: extraFile, Op: fsnotify.Create}

	result := w.Next()()
	require.NotNil(t, result)
	msg, ok := result.(msgs.FileAvailabilityChanged)
	require.True(t, ok)
	require.Empty(t, msg.Files)
}
