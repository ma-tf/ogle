# ADR-0010: Unit test conventions

**Status:** Accepted

## Context

The codebase had zero test coverage. A prior attempt to add tests
(`test-2026-05-10-add-unit-tests.md`) lacked sufficient design detail and
could not be implemented cleanly. A design interview was conducted to resolve
the outstanding decisions before implementation began.

The following conventions apply to all unit tests in this codebase. The parser
service (`internal/services/parser`) is the first application of these
conventions.

Options considered per decision are noted inline.

## Decision

**Test package style:** `package foo_test` (black-box) throughout. Internal
whitebox tests are not used except for `export_test.go` files, which are the
standard Go pattern for exposing private fields to black-box tests in the same
package. Currently one such file exists (`internal/app/export_test.go`).
Rationale: forces tests to interact only through the public API, preventing
tests from encoding implementation details.

**Assertions:** `testify/assert` and `testify/require` throughout. `require`
for preconditions and single-path assertions; `assert` for independent
multi-field checks. No raw `if err != nil { t.Fatal(...) }` patterns.

**Sentinel error assertions:** store the expected sentinel as `expectedError
error` on the test struct; assert with `require.ErrorIs`. Never use
`wantErr bool` — it cannot distinguish between wrong sentinels.

**Struct equality:** assert on the full returned struct with `require.Equal`.
Do not assert on named fields selectively — partial assertions hide regressions
in unexamined fields.

**Data-driven tests:** each test function owns a local `testCase` struct
(named `testCase`, scoped to the function — two functions in one file may both
define `testCase` without conflict). Fields are grouped and commented as
`// arrange` and `// assert`. Terminology uses `expected` throughout, not
`want`.

**Inline fixtures:** small input data (e.g. YAML strings) is inlined as a
field on the test struct. No `testdata/` directory. Rationale: keeps the full
test case — input and expected output — readable in one place; fixtures for
this domain are small and not shared across packages.

**Parallelism:** every test function calls `t.Parallel()` at its top, and
every `t.Run` subtest calls `t.Parallel()` as its first statement.

**Per-case setup:** each test case carries a `setup func(tc *testCase)` arrange
field. The test loop calls `tc.setup(&tc)` before writing any files or calling
the subject. The callback is responsible for all per-case environment
preparation: constructing paths, creating subdirectories, and populating any
mutable arrange fields (e.g. `path string`) whose values depend on the temp
directory. This avoids branching logic in the test loop and keeps each case
self-contained.

**Filesystem:** `t.TempDir()` with real files. No filesystem abstraction
layer. File writes are performed in the test loop after `setup` runs, guarded
by whether the relevant input field is non-empty (e.g. `if tc.yaml != ""`).

**Computed fields:** fields whose value cannot be known at table-definition
time are populated inside the `setup` callback, not hardcoded in the table and
not assigned ad-hoc in the test loop.

**Mocks:** generated via `mockery`. Named `MockFoo` in package `mocks`,
constructed via `NewMockFoo(t)` which registers assertion cleanup
automatically. Live in a `mocks/` subdirectory per package (e.g.
`internal/services/parser/mocks/`). Do not edit generated mock files manually.

**Hand-written fakes:** when mockery-generated mocks cannot produce the
required behaviour (e.g. programmable channels, error injection for event
loops), a hand-written fake is acceptable. The fake implements the target
interface with public channel fields for direct injection and a `sync.Mutex`
for thread-safe call recording. A constructor (`newFakeXxx`) initialises
channels. The fake is defined in the test file and is not shared across
packages. See `fakeFileWatcher` in `internal/services/watcher/service_test.go`
for a canonical example.

## Consequences

- All test cases for a given function are visible in one place with no
  indirection to external fixture files.
- Full struct equality will fail loudly if any field regresses, including
  fields not under active test focus.
- The `testCase` naming convention requires developers to read the enclosing
  function name for context — the struct name alone is not self-describing.
- `t.Parallel()` on every subtest maximises parallelism but requires each
  case to operate on its own `t.TempDir()` — shared mutable state across cases
  is not permitted.
- The `setup` callback makes each case self-describing but adds a small amount
  of boilerplate per case compared to a flat struct.
- `t.TempDir()` ties tests to the real filesystem; tests that write files are
  slightly slower than pure in-memory tests, which is acceptable for this
  domain.
- Mockery-generated mocks must be regenerated when interfaces change. The
  generated files must not be edited manually.
