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

### Export surface

- Model structs expose only `New()`, `Init()`, `Update()`, `View()`. No other exported methods on the model struct.

### Package boundary

- Black-box only: Tests must use `package foo_test`. No white-box accessors on production types, no exported for-testing functions on production types. Dependencies controllable via constructor injection only.
- Exception: `export_test.go` files that declare `package foo` (same-package) are accepted for exposing private fields/methods to black-box tests in the same package's `_test.go` files. This is the standard Go pattern for white-box accessors in test-only files, used sparingly (currently only `internal/app/export_test.go`).
