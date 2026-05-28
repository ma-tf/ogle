package startup

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
)

//nolint:gochecknoglobals // package-level key bindings are shared across all Model instances
var (
	keyUp     = key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up"))
	keyDown   = key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down"))
	keySelect = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select"))
	keyQuit   = key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit"))
	keyHelp   = key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help"))
	keyAbout  = key.NewBinding(key.WithKeys("f1"), key.WithHelp("f1", "about"))
)

type startupKeymap struct{}

func (k startupKeymap) PinnedHelp() []key.Binding {
	return []key.Binding{keyHelp, keyQuit}
}

func (k startupKeymap) ShortHelp() []key.Binding {
	return []key.Binding{keyUp, keyDown, keySelect}
}

func (k startupKeymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{keyUp, keyDown, keySelect, keyHelp, keyQuit, keyAbout}}
}

var _ help.KeyMap = startupKeymap{}
