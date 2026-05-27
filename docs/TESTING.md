# Testing

Conventions established 2026-05-24. Do not re-open without a technical reason.

---

## Service-layer unit tests

Applies to `internal/services/*` and any pure-function subject package.

### Package style

- `package foo_test` (black-box) throughout.

### Assertions

- `testify/require` for preconditions and single-path assertions.
- `testify/assert` for independent multi-field checks.
- No raw `t.Fatal`.

### Data-driven tables

- `testCase` struct scoped to each test function.
- Fields grouped with `// arrange` / `// assert` section comments.
- Assertion fields use `expected` prefix, never `want`.

```go
type testCase struct {
    name string
    // arrange
    input string
    setup func(tc *testCase)

    // assert
    expectedResult string
    expectedError  error
}
```

### Struct equality

- Assert full struct via `require.Equal`.
- Non-deterministic fields (e.g. temp dir paths) patched in `setup`, not hardcoded.

### Error assertions

- `expectedError error` field. Assert with `require.ErrorIs`. Never `wantErr bool`.

```go
if tc.expectedError != nil {
    require.ErrorIs(t, err, tc.expectedError)
    return
}
require.NoError(t, err)
```

### Setup callback

- Signature: `func(tc *testCase)`. Patches fields unknown at table-definition time.
- Must not use `t.Fatal`.

### Fixtures

- Inline only. No `testdata/` directory.

```go
if tc.input != "" {
    require.NoError(t, os.WriteFile(tc.path, []byte(tc.input), 0o600))
}
```

### Parallelism

- `t.Parallel()` on every test function and `t.Run` subtest.
- Each subtest uses its own `t.TempDir()`.

### Mocks

- Generated via [mockery](https://vektra.github.io/mockery/). `MockFoo` in `mocks/` package.
- Never edited manually. Kept even when unused — seam exists for future tests.

### Hand-written fakes

When mockery-generated mocks cannot produce the required behaviour (e.g. programmable channels, error injection for
 event loops), a hand-written fake is acceptable.

- Define the fake as a package-level struct implementing the target interface.
- Expose channels as public fields for direct injection in test cases.
- Use `sync.Mutex` for thread-safe access to recorded call data.
- Provide a constructor (`newFakeXxx`) that initialises channels.
- The test file owns the fake — do not share across packages.

```go
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

func (f *fakeFileWatcher) Close() error {
    f.mu.Lock()
    defer f.mu.Unlock()
    f.closed = true
    return f.closeErr
}
```

For simpler interfaces where the subject under test reads a return value synchronously (rather than receiving events
on a channel), a plain struct with configurable fields is sufficient — no constructor or channels needed:

```go
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
        f.cmd = exec.Command("sh", "-c", "echo '"+f.failStderr+"' >&2 && exit 1")
    } else {
        f.cmd = exec.Command("true")
    }
    return f.cmd
}
```

Choose channel-based fakes for event-loop interfaces (watcher, streamer) and field-based fakes for command-style
interfaces (Commander). See `fakeCommander` in `internal/services/docker/actions_test.go`.

---

## UI model tests

Baseline: service-layer conventions apply. Only deltas documented here.

### State machine driving

- `Init()` or `Update(msg)` → `(model, cmd)`.
- Feed returned `tea.Msg` into next `Update()`.

```go
m, cmd := m.Update(SomeMsg{})
```

### Command assertions

- Call returned `tea.Cmd` to get `tea.Msg`, then assert on the msg.
- Mandatory when cmd type is deterministic. Skip for non-deterministic (e.g. `tea.Tick`).

```go
m, cmd := m.Update(SomeMsg{})
require.NotNil(t, cmd)
result := cmd()
msg, ok := result.(SomeOtherMsg)
require.True(t, ok)
require.Equal(t, expectedField, msg.Field)
```

### TestUpdate

- `msg` is never nil. `View()` never called. Assert only on `expectedMsg tea.Msg`.
- `expectedMsg` nil → `require.Nil(t, cmd)`. Non-nil → `require.Equal(t, tc.expectedMsg, cmd())`.

```go
type testCase struct {
    name string
    // arrange
    input string

    // act
    msg tea.Msg

    // assert
    expectedMsg tea.Msg
}
```

```go
m, cmd := m.Update(tc.msg)

if tc.expectedMsg != nil {
    require.NotNil(t, cmd)
    require.Equal(t, tc.expectedMsg, cmd())
} else {
    require.Nil(t, cmd)
}
```

### TestView

- `Update()` never called in loop body — setup does it.
- `expectedResult string`: `""` → `assert.Empty`, else `assert.Contains`.
- No cmd capture.

```go
type testCase struct {
    name string
    // arrange
    input string
    setup func(m Model) Model

    // assert
    expectedResult string
}
```

```go
if tt.setup != nil {
    m = tt.setup(m)
}

if tt.expectedResult == "" {
    assert.Empty(t, m.View().Content)
} else {
    assert.Contains(t, m.View().Content, tt.expectedResult)
}
```

### Constructor injection

- All dependencies as constructor parameters.
- No model constructs infrastructure internally. `app.go` exempted.

### Constructor options pattern

When a production constructor needs both production defaults and test-time injection, use a functional options
pattern:

```go
type Option func(*Service)

func WithCommander(c Commander) Option {
    return func(s *Service) {
        s.commander = c
    }
}
```

See `svcdocker.WithCommander`, `svcdocker.WithHTTPClient`, `logs.WithHTTPClient` for canonical examples.

### Helper functions for test setup

Repeated test-case orchestration (e.g. constructing the subject with options, running a command, collecting output)
is extracted into a package-level helper to keep table-driven test functions concise:

```go
func runPsTestCase(t *testing.T, tc psCase) {
    t.Helper()
    composeFile := filepath.Join(t.TempDir(), "compose.yaml")
    fc := &psTestCommander{cmd: tc.cmd}
    s := svcdocker.New(svcdocker.WithCommander(fc))
    teaCmd := s.Ps(context.Background(), composeFile, testProject)
    require.NotNil(t, teaCmd)
    msg := teaCmd()
    ...
}
```

The helper calls `t.Helper()` so failure line numbers point to the calling test case. See `runPsTestCase` in
`internal/services/docker/docker_test.go`.

### Extended error assertions

Beyond `require.ErrorIs` with sentinel errors, the codebase also uses:

- `require.Error(t, err)` — when any error is acceptable (e.g. dial errors with unpredictable messages)
- `require.ErrorContains(t, err, substring)` — for stderr content matching (see `actions_test.go`)
- `require.ErrorAs(t, err, &target)` — for typed assertions like `*exec.ExitError` (see `actions_test.go`)

Use `require.ErrorIs` when a sentinel is available. Use `ErrorContains` / `ErrorAs` when the caller wraps dynamic
content (stderr, exit codes) into the error chain.

### Export surface

- Model structs expose `New()`, `Init()`, `Update()`, `View()`.

### Package boundary

- Black-box only: Tests must use `package foo_test`. Dependencies controllable via constructor injection only
 (`WithCommander`, `WithHTTPClient`, `WithReadFile` option pattern). Pure functions and types may be exported for
 black-box testing (e.g. `ParsePsOutput`, `ParseState`, `NormalizePorts`, `ApplyOverrides`, `UserThemeFile`,
 `ErrUnexpectedStatus`). No `export_test.go` files — the codebase has moved away from that pattern.
