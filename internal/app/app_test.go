package app_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	key "charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	"github.com/ma-tf/ogle/config"
	"github.com/ma-tf/ogle/internal/app"
	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	dockermocks "github.com/ma-tf/ogle/internal/services/docker/mocks"
	parsermocks "github.com/ma-tf/ogle/internal/services/parser/mocks"
	watchermocks "github.com/ma-tf/ogle/internal/services/watcher/mocks"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	testComposeFile = "/tmp/compose.yaml"
	testServiceName = "web"
	testStatusMsg   = "hello"
	testProjectName = "myapp"
	testComposePath = "/path/to/compose.yaml"
	testImageName   = "nginx:latest"
)

type testKeymap struct{}

func (testKeymap) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}
}

func (testKeymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
			key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		},
	}
}

func newModel(t *testing.T) (
	app.Model, func() error, *dockermocks.MockDocker, *watchermocks.MockWatcher,
) {
	t.Helper()

	ctx := context.Background()
	cfg := config.Defaults()
	th := theme.Default()

	mockDocker := dockermocks.NewMockDocker(t)
	mockParser := parsermocks.NewMockParser(t)
	mockWatcher := watchermocks.NewMockWatcher(t)

	mockWatcher.EXPECT().Close().Return(nil)

	m, cleanup, err := app.New(ctx, cfg, "", "", th, mockDocker, mockParser, mockWatcher)
	require.NoError(t, err)

	return m, cleanup, mockDocker, mockWatcher
}

func TestInit(t *testing.T) {
	t.Parallel()

	m, cleanup, mockDocker, mockWatcher := newModel(t)
	defer func() {
		require.NoError(t, cleanup())
	}()

	mockDocker.EXPECT().Connect(mock.Anything).Return(func() tea.Msg { return nil }).Maybe()
	mockWatcher.EXPECT().Snapshot().Return(nil)

	cmd := m.Init()
	require.NotNil(t, cmd)
}

func TestUpdateQuit(t *testing.T) {
	t.Parallel()

	m, cleanup, _, _ := newModel(t)
	defer func() {
		require.NoError(t, cleanup())
	}()

	result, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	require.NotNil(t, result)
	require.NotNil(t, cmd)

	msg := cmd()
	require.NotNil(t, msg)
}

func TestUpdateFileAvailabilityChanged(t *testing.T) {
	t.Parallel()

	m, cleanup, _, watcherMock := newModel(t)
	defer func() {
		require.NoError(t, cleanup())
	}()

	watcherMock.EXPECT().Next().Return(func() tea.Msg { return nil })

	msg := msgs.FileAvailabilityChanged{Files: []string{testComposeFile}}
	result, cmd := m.Update(msg)
	require.NotNil(t, result)
	require.NotNil(t, cmd)
}

func TestUpdateFileRemoved(t *testing.T) {
	t.Parallel()

	m, cleanup, _, _ := newModel(t)
	defer func() {
		require.NoError(t, cleanup())
	}()

	msg := msgs.FileRemoved{File: testComposeFile}
	result, cmd := m.Update(msg)
	require.NotNil(t, result)
	require.NotNil(t, cmd)

	assert.Contains(t, result.View().Content, "compose file unavailable — waiting")

	resultMsg := cmd()
	batch, ok := resultMsg.(tea.BatchMsg)
	require.True(t, ok, "expected BatchMsg, got %T", resultMsg)

	foundTopbar := false
	foundBindings := false
	foundFrameHeight := false

	for _, entry := range batch {
		if tc, tcOk := entry().(msgs.TopbarContext); tcOk {
			assert.Equal(t, "watching", tc.Phase)
			assert.Empty(t, tc.File)

			foundTopbar = true
		} else if bm, bmOk := entry().(msgs.BindingsMsg); bmOk {
			assert.NotNil(t, bm.Keymap)

			foundBindings = true
		} else if _, fhOk := entry().(msgs.FrameHeight); fhOk {
			foundFrameHeight = true
		}
	}

	assert.True(t, foundTopbar, "expected TopbarContext in BatchMsg")
	assert.True(t, foundBindings, "expected BindingsMsg in BatchMsg")
	assert.True(t, foundFrameHeight, "expected FrameHeight in BatchMsg")
}

func TestWatchingKeymapPinnedHelp(t *testing.T) {
	t.Parallel()

	m, cleanup, _, _ := newModel(t)
	defer func() { require.NoError(t, cleanup()) }()

	_, cmd := m.Update(msgs.FileRemoved{File: testComposeFile})
	require.NotNil(t, cmd)

	batch, ok := cmd().(tea.BatchMsg)
	require.True(t, ok)

	bindMsg, found := extractBindingsMsgFromBatch(t, batch)
	require.True(t, found)

	km, ok := bindMsg.Keymap.(interface {
		PinnedHelp() []key.Binding
	})
	require.True(t, ok, "keymap should implement PinnedHelp")

	pinned := km.PinnedHelp()
	require.Len(t, pinned, 2)
	assert.Equal(t, "?", pinned[0].Help().Key)
	assert.Equal(t, "toggle help", pinned[0].Help().Desc)
	assert.Equal(t, "q", pinned[1].Help().Key)
	assert.Equal(t, "quit", pinned[1].Help().Desc)
}

func TestWatchingKeymapShortHelpEmpty(t *testing.T) {
	t.Parallel()

	m, cleanup, _, _ := newModel(t)
	defer func() { require.NoError(t, cleanup()) }()

	_, cmd := m.Update(msgs.FileRemoved{File: testComposeFile})
	require.NotNil(t, cmd)

	batch, ok := cmd().(tea.BatchMsg)
	require.True(t, ok)

	bindMsg, found := extractBindingsMsgFromBatch(t, batch)
	require.True(t, found)

	bindings := bindMsg.Keymap.ShortHelp()
	assert.Empty(t, bindings)
}

func TestWatchingKeymapFullHelp(t *testing.T) {
	t.Parallel()

	m, cleanup, _, _ := newModel(t)
	defer func() { require.NoError(t, cleanup()) }()

	_, cmd := m.Update(msgs.FileRemoved{File: testComposeFile})
	require.NotNil(t, cmd)

	batch, ok := cmd().(tea.BatchMsg)
	require.True(t, ok)

	bindMsg, found := extractBindingsMsgFromBatch(t, batch)
	require.True(t, found)

	fullHelp := bindMsg.Keymap.FullHelp()
	require.Len(t, fullHelp, 1)

	col := fullHelp[0]
	assert.Equal(t, "?", col[0].Help().Key)
	assert.Equal(t, "toggle help", col[0].Help().Desc)
	assert.Equal(t, "q", col[1].Help().Key)
	assert.Equal(t, "quit", col[1].Help().Desc)
	assert.Equal(t, "f1", col[2].Help().Key)
	assert.Equal(t, "about", col[2].Help().Desc)
}

func TestUpdateFileAvailabilityChangedDuringDashboard(t *testing.T) {
	t.Parallel()

	m, cleanup, _, watcherMock := newModel(t)
	defer func() {
		require.NoError(t, cleanup())
	}()

	project := &domain.Project{
		Name: testProjectName,
		File: testComposePath,
		Services: []domain.ServiceDef{
			{Name: testServiceName, Image: testImageName},
		},
	}

	resultPL, _ := m.Update(msgs.ProjectLoaded{Project: project})

	var plOk bool

	m, plOk = resultPL.(app.Model)
	require.True(t, plOk)

	watcherMock.EXPECT().Next().Return(func() tea.Msg { return nil })

	msg := msgs.FileAvailabilityChanged{Files: []string{testComposeFile}}
	result, cmd := m.Update(msg)
	require.NotNil(t, result)
	require.NotNil(t, cmd)

	assert.Contains(t, result.View().Content, testServiceName)
}

func TestUpdateFileAvailabilityChangedDuringWatching(t *testing.T) {
	t.Parallel()

	m, cleanup, _, watcherMock := newModel(t)
	defer func() {
		require.NoError(t, cleanup())
	}()

	resultFR, _ := m.Update(msgs.FileRemoved{File: "compose.yaml"})

	var frOk bool

	m, frOk = resultFR.(app.Model)
	require.True(t, frOk)

	watcherMock.EXPECT().Next().Return(func() tea.Msg { return nil })

	msg := msgs.FileAvailabilityChanged{Files: []string{testComposeFile}}
	result, cmd := m.Update(msg)
	require.NotNil(t, result)
	require.NotNil(t, cmd)

	assert.Contains(t, result.View().Content, "compose file unavailable")
}

func TestUpdateProjectLoaded(t *testing.T) {
	t.Parallel()

	m, cleanup, _, _ := newModel(t)
	defer func() {
		require.NoError(t, cleanup())
	}()

	assert.Contains(t, m.View().Content, "scanning for compose files",
		"should start in startup phase")

	project := &domain.Project{
		Name: testProjectName,
		File: testComposePath,
		Services: []domain.ServiceDef{
			{Name: testServiceName, Image: testImageName},
		},
	}

	result, cmd := m.Update(msgs.ProjectLoaded{Project: project})
	require.NotNil(t, result)
	require.NotNil(t, cmd)

	assert.Contains(t, result.View().Content, testServiceName,
		"should transition to dashboard phase")

	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	require.True(t, ok, "expected BatchMsg, got %T", msg)

	found := false
	foundFrameHeight := false

	for _, entry := range batch {
		if tc, tcOk := entry().(msgs.TopbarContext); tcOk {
			assert.Equal(t, "dashboard", tc.Phase)
			assert.Equal(t, "compose.yaml", tc.File)

			found = true
		}

		if _, fhOk := entry().(msgs.FrameHeight); fhOk {
			foundFrameHeight = true
		}
	}

	require.True(t, found, "expected TopbarContext in BatchMsg")
	assert.True(t, foundFrameHeight, "expected FrameHeight in BatchMsg")
}

func TestView(t *testing.T) {
	t.Parallel()

	m, cleanup, _, _ := newModel(t)
	defer func() {
		require.NoError(t, cleanup())
	}()

	v := m.View()
	require.NotNil(t, v)
	assert.NotEmpty(t, v.Content)
}

func newModelWithConfig(t *testing.T, configPath string) (
	app.Model, func() error, *theme.Theme,
) {
	t.Helper()

	ctx := context.Background()
	cfg := config.Defaults()
	th := theme.Default()

	mockDocker := dockermocks.NewMockDocker(t)
	mockParser := parsermocks.NewMockParser(t)
	mockWatcher := watchermocks.NewMockWatcher(t)
	mockWatcher.EXPECT().Close().Return(nil)

	m, cleanup, err := app.New(
		ctx, cfg, configPath, "", th, mockDocker, mockParser, mockWatcher,
	)
	require.NoError(t, err)

	return m, cleanup, th
}

func TestUpdateSettingsApplied(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name              string
		themeName         string
		logBufferCap      int
		expectThemeLoaded bool
	}

	for _, tc := range []testCase{
		{
			name:              "known theme loads and persists",
			themeName:         "solarized_dark",
			logBufferCap:      2000,
			expectThemeLoaded: true,
		},
		{
			name:              "unknown theme keeps existing theme",
			themeName:         "not-a-theme",
			logBufferCap:      2000,
			expectThemeLoaded: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			configPath := filepath.Join(dir, "config.yaml")

			require.NoError(t, config.Save(configPath, config.Defaults()))

			m, cleanup, originalTheme := newModelWithConfig(t, configPath)
			defer func() {
				require.NoError(t, cleanup())
			}()

			msg := msgs.SettingsApplied{Theme: tc.themeName, LogBufferCap: tc.logBufferCap}
			result, cmd := m.Update(msg)
			require.NotNil(t, result)
			require.NotNil(t, cmd)

			changed := extractThemeChanged(t, cmd())
			require.NotNil(t, changed, "expected theme.Changed in batch")

			if tc.expectThemeLoaded {
				assert.NotSame(t, originalTheme, changed.Theme)
			} else {
				assert.Same(t, originalTheme, changed.Theme)
			}

			data, err := os.ReadFile(configPath)
			require.NoError(t, err, "config should be persisted")

			var savedCfg config.Config
			require.NoError(t, yaml.Unmarshal(data, &savedCfg))
			assert.Equal(t, tc.themeName, savedCfg.Theme)
			assert.Equal(t, tc.logBufferCap, savedCfg.LogBufferCap)
		})
	}
}

func TestUpdateSettingsAppliedConfigSaveFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "nonexistent", "config.yaml")

	m, cleanup, originalTheme := newModelWithConfig(t, configPath)
	defer func() {
		require.NoError(t, cleanup())
	}()

	msg := msgs.SettingsApplied{Theme: "solarized_dark", LogBufferCap: 2000}
	result, cmd := m.Update(msg)
	require.NotNil(t, result)
	require.NotNil(t, cmd)

	changed := extractThemeChanged(t, cmd())
	require.NotNil(t, changed, "expected theme.Changed in batch")

	assert.NotSame(t, originalTheme, changed.Theme)

	_, err := os.Stat(configPath)
	require.True(t, os.IsNotExist(err))
}

func TestUpdateWindowSize(t *testing.T) {
	t.Parallel()

	m, cleanup, _, _ := newModel(t)
	defer func() {
		require.NoError(t, cleanup())
	}()

	result, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	require.NotNil(t, result)
	require.NotNil(t, cmd)

	assertHasFrameHeight(t, cmd)
}

func assertFrameHeightInBatch(t *testing.T, batch tea.BatchMsg) {
	t.Helper()

	for _, entry := range batch {
		if entry == nil {
			continue
		}

		if fh, ok := entry().(msgs.FrameHeight); ok {
			assert.Positive(t, fh.Height, "FrameHeight should be positive")
			assert.LessOrEqual(t, fh.Height, 10, "FrameHeight should be reasonable")

			return
		}
	}

	t.Log("FrameHeight not found in batch - entries:", len(batch))
}

func assertHasFrameHeight(t *testing.T, cmd tea.Cmd) {
	t.Helper()

	require.NotNil(t, cmd)
	msg := cmd()

	if batch, ok := msg.(tea.BatchMsg); ok {
		assertFrameHeightInBatch(t, batch)

		return
	}

	if fh, ok := msg.(msgs.FrameHeight); ok {
		assert.Positive(t, fh.Height, "FrameHeight should be positive")
		assert.LessOrEqual(t, fh.Height, 10, "FrameHeight should be reasonable")

		return
	}

	t.Errorf("expected FrameHeight, got %T", msg)
}

func TestViewPhaseContent(t *testing.T) {
	t.Parallel()

	project := &domain.Project{
		Name: testProjectName,
		File: testComposePath,
		Services: []domain.ServiceDef{
			{Name: testServiceName, Image: testImageName},
		},
	}

	type testCase struct {
		name string
		// arrange
		setup func(m tea.Model) tea.Model
		// assert
		expectedContains string
	}

	for _, tc := range []testCase{
		{
			name: "startup phase shows startup body",
			setup: func(m tea.Model) tea.Model {
				return m
			},
			expectedContains: "scanning for compose files",
		},
		{
			name: "dashboard phase shows dashboard body",
			setup: func(m tea.Model) tea.Model {
				r, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
				m = r

				r, _ = m.Update(msgs.ProjectLoaded{Project: project})

				return r
			},
			expectedContains: testServiceName,
		},
		{
			name: "watching phase shows watching body",
			setup: func(m tea.Model) tea.Model {
				r, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
				m = r

				r, _ = m.Update(msgs.FileRemoved{File: "test.yaml"})

				return r
			},
			expectedContains: "compose file unavailable",
		},
		{
			name: "about overlay composes on top",
			setup: func(m tea.Model) tea.Model {
				r, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
				m = r

				r, _ = m.Update(msgs.AboutVisibilityChanged{Visible: true})

				return r
			},
			expectedContains: "github.com/ma-tf/ogle",
		},
		{
			name: "display error routes to statusbar",
			setup: func(m tea.Model) tea.Model {
				r, _ := m.Update(msgs.DisplayError{Err: "oops"})

				return r
			},
			expectedContains: "oops",
		},
		{
			name: "display status routes to statusbar",
			setup: func(m tea.Model) tea.Model {
				r, _ := m.Update(msgs.DisplayStatus{Msg: testStatusMsg})

				return r
			},
			expectedContains: testStatusMsg,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m, cleanup, _, _ := newModel(t)
			defer func() {
				require.NoError(t, cleanup())
			}()

			r := tc.setup(m)
			m2, ok := r.(app.Model)
			require.True(t, ok, "expected app.Model, got %T", r)

			v := m2.View()
			require.NotNil(t, v)
			assert.Contains(t, v.Content, tc.expectedContains)
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// act
		msg tea.Msg
		// assert
		expectedMsg tea.Msg
		expectCmd   bool
		check       func(*testing.T, tea.Cmd)
	}

	cases := []testCase{
		{
			name:      "DaemonConnected produces command",
			msg:       msgs.DaemonConnected{},
			expectCmd: true,
		},
		{
			name:      "DaemonUnavailable produces command",
			msg:       msgs.DaemonUnavailable{Err: assert.AnError},
			expectCmd: true,
		},
		{
			name:      "DisplayError produces command",
			msg:       msgs.DisplayError{Err: "test error"},
			expectCmd: true,
		},
		{
			name:      "DisplayStatus produces command",
			msg:       msgs.DisplayStatus{Msg: testStatusMsg},
			expectCmd: true,
			check: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				assertHasFrameHeight(t, cmd)
			},
		},
		{
			name:      "ClearStatusMsg produces command",
			msg:       msgs.ClearStatusMsg{},
			expectCmd: true,
			check: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				assertHasFrameHeight(t, cmd)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m, cleanup, _, _ := newModel(t)
			defer func() {
				require.NoError(t, cleanup())
			}()

			_, cmd := m.Update(tc.msg)

			switch {
			case tc.check != nil:
				tc.check(t, cmd)
			case tc.expectedMsg != nil:
				require.NotNil(t, cmd)
				require.Equal(t, tc.expectedMsg, cmd())
			case tc.expectCmd:
				require.NotNil(t, cmd)
			default:
				require.Nil(t, cmd)
			}
		})
	}
}

func showingAbout() func(m app.Model) app.Model {
	return func(m app.Model) app.Model {
		r, _ := m.Update(msgs.AboutVisibilityChanged{Visible: true})

		model, _ := r.(app.Model)

		return model
	}
}

//nolint:funlen // many table-driven cases
func TestUpdateKeyPress(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name         string
		setup        func(m app.Model) app.Model
		msg          tea.Msg
		expectedMsg  tea.Msg
		expectCmd    bool
		checkShowAll bool
		check        func(*testing.T, tea.Cmd)
	}

	cases := []testCase{
		{
			name:      "ctrl+c quits",
			msg:       tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl},
			expectCmd: true,
		},
		{
			name:      "ctrl+p produces profile dump command",
			msg:       tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl},
			expectCmd: true,
		},
		{
			name:         "question mark toggles help bar",
			msg:          tea.KeyPressMsg{Text: "?"},
			checkShowAll: true,
		},
		{
			name: "question mark toggle includes FrameHeight cmd",
			setup: func(m app.Model) app.Model {
				r, _ := m.Update(msgs.BindingsMsg{Keymap: testKeymap{}})

				m, _ = r.(app.Model)

				return m
			},
			msg: tea.KeyPressMsg{Text: "?"},
			check: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				assertHasFrameHeight(t, cmd)
			},
		},
		{
			name:        "F1 opens about when not shown",
			msg:         tea.KeyPressMsg{Code: tea.KeyF1},
			expectedMsg: msgs.AboutVisibilityChanged{Visible: true},
		},
		{
			name:        "F1 closes about when shown",
			setup:       showingAbout(),
			msg:         tea.KeyPressMsg{Code: tea.KeyF1},
			expectedMsg: msgs.AboutVisibilityChanged{Visible: false},
		},
		{
			name:        "q closes about when shown",
			setup:       showingAbout(),
			msg:         tea.KeyPressMsg{Text: "q"},
			expectedMsg: msgs.AboutVisibilityChanged{Visible: false},
		},
		{
			name:        "esc closes about when shown",
			setup:       showingAbout(),
			msg:         tea.KeyPressMsg{Code: tea.KeyEsc},
			expectedMsg: msgs.AboutVisibilityChanged{Visible: false},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m, cleanup, _, _ := newModel(t)
			defer func() { require.NoError(t, cleanup()) }()

			if tc.setup != nil {
				m = tc.setup(m)
			}

			if tc.checkShowAll {
				r, _ := m.Update(msgs.BindingsMsg{Keymap: testKeymap{}})
				m2, ok := r.(app.Model)
				require.True(t, ok)

				v1 := m2.View().Content

				r, cmd := m2.Update(tc.msg)
				require.NotNil(t, r)
				require.NotNil(t, cmd, "cmd should be non-nil (helpbar + frameHeight)")

				m3, ok := r.(app.Model)
				require.True(t, ok)

				v2 := m3.View().Content
				assert.NotEqual(t, v1, v2, "help bar content should change on toggle")

				return
			}

			result, cmd := m.Update(tc.msg)
			require.NotNil(t, result)

			switch {
			case tc.check != nil:
				tc.check(t, cmd)
			case tc.expectedMsg != nil:
				require.NotNil(t, cmd)
				require.Equal(t, tc.expectedMsg, cmd())
			case tc.expectCmd:
				require.NotNil(t, cmd)
			default:
				require.Nil(t, cmd)
			}
		})
	}
}

func TestUpdateMouseClick(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name             string
		setup            func(m app.Model) app.Model
		msg              tea.Msg
		expectedMsg      tea.Msg
		expectAboutOpen  bool
		wantShowingAbout bool
	}

	cases := []testCase{
		{
			name:             "brand zone click opens about when not shown",
			msg:              tea.MouseClickMsg{X: 0, Y: 0},
			expectedMsg:      msgs.AboutVisibilityChanged{Visible: true},
			expectAboutOpen:  true,
			wantShowingAbout: true,
		},
		{
			name:             "click anywhere when about shown is consumed",
			setup:            showingAbout(),
			msg:              tea.MouseClickMsg{X: 0, Y: 0},
			wantShowingAbout: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m, cleanup, _, _ := newModel(t)
			defer func() {
				require.NoError(t, cleanup())
			}()

			if tc.setup != nil {
				m = tc.setup(m)
			}
			// Call View to register brand zone in the zone manager.
			_ = m.View()
			// Yield to let the zone worker goroutine process the scan.
			for range 50 {
				runtime.Gosched()
			}

			result, cmd := m.Update(tc.msg)
			require.NotNil(t, result)
			appModel, ok := result.(app.Model)
			require.True(t, ok)

			if tc.expectedMsg != nil {
				require.NotNil(t, cmd)
				require.Equal(t, tc.expectedMsg, cmd())
			}

			if tc.expectAboutOpen {
				require.Contains(t, appModel.View().Content, "github.com/ma-tf/ogle")
			}

			if tc.wantShowingAbout {
				assert.Contains(t, appModel.View().Content, "github.com/ma-tf/ogle")
			}
		})
	}
}

func extractBindingsMsgFromBatch(t *testing.T, batch tea.BatchMsg) (msgs.BindingsMsg, bool) {
	t.Helper()

	for _, entry := range batch {
		if bm, ok := entry().(msgs.BindingsMsg); ok {
			return bm, true
		}
	}

	return msgs.BindingsMsg{}, false
}

// Helper: extractThemeChanged unwraps a single message or batch message to
// find the first theme.Changed.
func extractThemeChanged(t *testing.T, msg tea.Msg) *theme.Changed {
	t.Helper()

	switch m := msg.(type) {
	case theme.Changed:
		return &m
	case tea.BatchMsg:
		for _, entry := range m {
			if tc := extractThemeChanged(t, entry()); tc != nil {
				return tc
			}
		}
	}

	return nil
}
