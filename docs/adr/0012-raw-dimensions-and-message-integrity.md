# ADR-0012: Components receive raw terminal dimensions; no parent pre-calculation or message reconstruction

**Status:** Accepted

> **Note (implementation status):** `layout.FrameHeight` (aliased as `ChromeHeight` in the ADR body) was extracted as a
> shared constant at `internal/ui/layout/layout.go`, removing the duplicated `frameHeight` locals from `dashboard`,
> `startup`, and `watching`. Components now receive raw terminal dimensions and derive their own content size internally.
> `servicehost.ServiceName()` and `statusbar.Height()` have been removed. The remaining items (no message
> reconstruction, pre-resize overflow guard) are also implemented in the current codebase.

## Context

The ogle component tree uses Bubble Tea's `tea.Model` convention: every component implements `Init()`,
`Update(tea.Msg)`, and `View()`. Terminal dimensions arrive as `tea.WindowSizeMsg{Width, Height}` from the runtime and
propagate through the tree. In practice, parent components have been pre-calculating usable dimensions (subtracting
chrome height, splitting percentages) and either (a) passing pre-chewed sizes to child constructors, or (b)
reconstructing a synthetic `tea.WindowSizeMsg` with modified `Height` before forwarding. Two related smells appear:

1. **Helper methods exposing internals** — `servicehost.ServiceName()` and `statusbar.Height()` expose internal state so
parents can route messages and adjust layout.
2. **Message reconstruction in the chain** — `startup.go` builds `tea.WindowSizeMsg{Width: msg.Width, Height: msg.Height

- frameHeight}` instead of passing the original message through. `dashboard.go` and `watching.go` each duplicate
`frameHeight` and subtract it on receipt.

1. **Pre-resize overflow** — Before the first `tea.WindowSizeMsg` arrives, `app.View()` composes children at zero
dimensions, causing content to overflow over the helpbar.

The result is a shallow seam: every child must know what its parent subtracted, and parents must query children for
height or name to compose the frame. The interface of each module is nearly as complex as its implementation.

Options considered:

1. **Status quo** — parents pre-calculate dimensions, pass them to constructors, and reconstruct messages. Rejected:
leaks layout policy across seams; duplication of `frameHeight`, `listRatio`, and `listMinTermWidth`; children cannot be
tested without reverse-engineering parent arithmetic.
2. **Raw dimensions, internal calculation** — `tea.WindowSizeMsg` propagates unchanged. Every component stores raw `w`
and `h` and derives its own content size internally. Parents never query children for height or routing state;
`ServiceSelected` messages drive selection, not `ServiceName()`. Accepted.
3. **Shared layout value object** — a `layout.Dashboard` struct centralises split ratios and chrome constants. Rejected:
adds a new dependency; the duplication of `listRatio` is small and local; the deeper problem is message reconstruction,
not constant duplication.

## Decision

- `tea.WindowSizeMsg` is never reconstructed, wrapped, or modified in the middle of the message chain. It propagates
everywhere with its original `Width` and `Height`.
- Constructors receive **raw terminal dimensions**, never parent-pre-calculated ones. Each component derives its own
content size from raw `w` and `h` at construction time, avoiding a zero-size frame before the first resize message
arrives.
- On subsequent `tea.WindowSizeMsg`, each component re-derives its content size from the new raw dimensions.
- Parent components do not query children for height or internal state to adjust layout. `statusbar` renders zero-height
content when inactive; `app.View()` measures every chrome element dynamically and derives the body area from exact
heights. `servicehost` decides internally whether to consume input and whether to render, driven by `ServiceSelected`
messages.
- `app.View()` returns an empty view when terminal dimensions are zero (before the first resize), preventing pre-resize
overflow.

## Consequences

- `layout.ChromeHeight` (3 lines: topbar + helpbar) is a shared layout policy constant for the always-present chrome. It
is removed from `dashboard`, `startup`, and `watching` as duplicated local constants. Components that make height-based
layout decisions (e.g. `dashboard`, `logpane`) import the shared value and derive their own usable area internally,
guarding with `max(0, ...)` to avoid negative dimensions on small terminals.
- `listRatio` and `listMinTermWidth` may still be duplicated between `carousel` and `logpane` because both independently
derive their allocation from raw `w`. This is accepted as the lesser evil versus parent pre-calculation.
- `servicehost.ServiceName()` and `statusbar.Height()` are removed. The interface of a component is `Init/Update/View`
only.
- Tests improve: any model can be exercised with `tea.WindowSizeMsg{Width: 80, Height: 24}` and asserted to store
exactly 80×24, with no need to reverse-engineer what a parent subtracted.
- View tests assert on rendered content (empty string vs active message) rather than querying `Height()`.
