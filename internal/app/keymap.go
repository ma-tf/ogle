package app

import (
	"charm.land/bubbles/v2/key"
)

//nolint:gochecknoglobals // package-level key bindings
var (
	watchingHelpKey  = key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help"))
	watchingQuitKey  = key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit"))
	watchingAboutKey = key.NewBinding(key.WithKeys("f1"), key.WithHelp("f1", "about"))
)

type watchingKeymap struct{}

func (k watchingKeymap) PinnedHelp() []key.Binding {
	return []key.Binding{watchingHelpKey, watchingQuitKey}
}

func (k watchingKeymap) ShortHelp() []key.Binding {
	return nil
}

func (k watchingKeymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{watchingHelpKey, watchingQuitKey, watchingAboutKey}}
}
