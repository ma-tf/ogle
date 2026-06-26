// Package clipboard abstracts the system clipboard behind an interface so
// production code can write to the clipboard and tests can use a mock.
//
//mockery:generate: true
package clipboard

import (
	"fmt"

	atotto "github.com/atotto/clipboard"
)

// Clipboard writes text to the system clipboard.
type Clipboard interface {
	WriteAll(text string) error
}

type systemClipboard struct{}

// WriteAll writes the given text to the system clipboard via the atotto
// library. Only the plain-text clipboard is supported.
func (systemClipboard) WriteAll(text string) error {
	if err := atotto.WriteAll(text); err != nil {
		return fmt.Errorf("clipboard write: %w", err)
	}

	return nil
}

// New returns a production Clipboard that writes to the real system clipboard.
func New() Clipboard {
	return systemClipboard{}
}
