package theme_test

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"

	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	testColorRed   = "#ff0000"
	testColorGreen = "#00ff00"
	testColorBlue  = "#0000ff"
)

func TestApplyOverrides(t *testing.T) {
	t.Parallel()

	original := theme.Default()
	base := theme.Default()

	overrides := theme.UserThemeFile{
		TextColour:           testColorRed,
		StateRunningColour:   testColorGreen,
		HelpBackgroundColour: testColorBlue,
	}

	result := theme.ApplyOverrides(base, overrides)

	assert.Equal(t, lipgloss.Color(testColorRed), result.Text)
	assert.Equal(t, lipgloss.Color(testColorGreen), result.StateRunning)
	assert.Equal(t, lipgloss.Color(testColorBlue), result.HelpBackground)

	assert.Equal(t, original.StateExited, result.StateExited)
	assert.Equal(t, original.StatePaused, result.StatePaused)
	assert.Equal(t, original.Subtext, result.Subtext)
}

func TestApplyOverridesEmptyFieldsDoNotOverride(t *testing.T) {
	t.Parallel()

	base := theme.Default()

	overrides := theme.UserThemeFile{
		TextColour: testColorRed,
	}

	result := theme.ApplyOverrides(base, overrides)

	assert.Equal(t, lipgloss.Color(testColorRed), result.Text)
	assert.Equal(t, base.StateRunning, result.StateRunning)
}
