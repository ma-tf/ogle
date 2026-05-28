// Package layout defines shared layout policy constants for the ogle TUI.
// These are cross-cutting constants used by both the app chrome and phase
// components to agree on how much terminal space is consumed by chrome.
package layout

// FrameHeight is the fallback default for the number of terminal lines
// consumed by the app-level chrome (topbar + bottom bar). Components use
// this value until they receive a msgs.FrameHeight message with the actual
// current chrome height.
const FrameHeight = 3
