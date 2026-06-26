// Package layout defines shared layout policy constants for the ogle TUI.
// These are cross-cutting constants used by both the app chrome and phase
// components to agree on how much terminal space is consumed by chrome.
package layout

// FrameHeight is the fallback default for the number of terminal lines
// consumed by the app-level chrome (topbar + bottom bar). Components use
// this value until they receive a msgs.FrameHeight message with the actual
// current chrome height.
const FrameHeight = 2

// SidebarWidth is the fixed width in columns of the Service List sidebar.
const SidebarWidth = 40

// SidebarMinTermWidth is the minimum terminal width at which the sidebar
// is shown. When the terminal is narrower the sidebar collapses and only
// the Service Inspector is rendered.
const SidebarMinTermWidth = 80
