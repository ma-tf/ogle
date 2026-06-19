package carousel

import (
	"charm.land/bubbles/v2/key"
)

//nolint:gochecknoglobals // package-level key bindings are shared across all Model instances
var (
	// KeyShiftTab focuses the previous card slot.
	KeyShiftTab = key.NewBinding(
		key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "focus previous"),
	)
	// KeyTab focuses the next card slot.
	KeyTab = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "focus next"))
	// KeyEnter starts or stops the focused service.
	KeyEnter = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "start/stop"))
	// KeyPgUp navigates to the previous page of cards.
	KeyPgUp = key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "prev page"))
	// KeyPgDown navigates to the next page of cards.
	KeyPgDown = key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdown", "next page"))
)

// Keymap implements help.KeyMap for the carousel component.
type Keymap struct{}

// ShortHelp returns the carousel's key bindings for the help bar.
func (k Keymap) ShortHelp() []key.Binding {
	return []key.Binding{KeyTab, KeyShiftTab, KeyEnter, KeyPgUp, KeyPgDown}
}

// FullHelp returns nil; the carousel has no full help view.
func (k Keymap) FullHelp() [][]key.Binding {
	return nil
}
