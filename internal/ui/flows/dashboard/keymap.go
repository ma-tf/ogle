package dashboard

import (
	"charm.land/bubbles/v2/key"

	"github.com/ma-tf/ogle/internal/ui/components/carousel"
)

//nolint:gochecknoglobals // package-level key bindings are shared across all Model instances
var (
	keyQuit        = key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit"))
	keySettings    = key.NewBinding(key.WithKeys(","), key.WithHelp(",", "settings"))
	keyToggleWrap  = key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "toggle wrap"))
	keyScrollUp    = key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "scroll up"))
	keyScrollDown  = key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "scroll down"))
	keyScrollLeft  = key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "scroll left"))
	keyScrollRight = key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "scroll right"))
	keyRestart     = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "restart"))
	keyRebuild     = key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "rebuild"))
	keyClearLog    = key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clear log"))
	keyHelp        = key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help"))
	keyAbout       = key.NewBinding(key.WithKeys("f1"), key.WithHelp("f1", "about"))
)

type appKeymap struct{}

func (k appKeymap) ShortHelp() []key.Binding {
	return []key.Binding{
		carousel.KeyTab,
		carousel.KeyEnter,
		keyRestart,
		keyRebuild,
		keyToggleWrap,
	}
}

func (k appKeymap) PinnedHelp() []key.Binding {
	return []key.Binding{keyHelp, keyQuit}
}

func (k appKeymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			carousel.KeyTab,
			carousel.KeyEnter,
			keyRestart,
			keyRebuild,
		},
		{
			keyScrollUp,
			keyScrollDown,
			keyScrollLeft,
			keyScrollRight,
		},
		{
			carousel.KeyPgUp,
			carousel.KeyPgDown,
			keyToggleWrap,
			keyClearLog,
			keySettings,
		},
		{
			keyHelp,
			keyQuit,
			keyAbout,
		},
	}
}
