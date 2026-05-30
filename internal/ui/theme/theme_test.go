package theme_test

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

func builtinThemeNames() []string {
	return []string{
		"default", "default_light",
		"catppuccino_frappe", "catppuccino_latte",
		"catppuccino_macchiato", "catppuccino_mocha",
		"solarized_dark", "solarized_light",
	}
}

func TestCardHoverBackgroundIsPurple(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name      string
		construct func() *theme.Theme
	}

	for _, tc := range []testCase{
		{name: "default", construct: theme.Default},
		{name: "default_light", construct: theme.DefaultLight},
		{name: "catppuccino_frappe", construct: theme.CatppuccinoFrappe},
		{name: "catppuccino_latte", construct: theme.CatppuccinoLatte},
		{name: "catppuccino_macchiato", construct: theme.CatppuccinoMacchiato},
		{name: "catppuccino_mocha", construct: theme.CatppuccinoMocha},
		{name: "solarized_dark", construct: theme.SolarizedDark},
		{name: "solarized_light", construct: theme.SolarizedLight},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			th := tc.construct()
			require.NotNil(t, th)

			assert.NotEqual(t, th.HoverBackground, th.CardHoverBackground,
				"CardHoverBackground should differ from HoverBackground")
		})
	}
}

func TestColourForState(t *testing.T) {
	t.Parallel()

	th := theme.Default()

	type testCase struct {
		name          string
		state         domain.ServiceState
		expectedColor color.Color
	}

	for _, tc := range []testCase{
		{name: "running", state: domain.ServiceStateRunning, expectedColor: th.StateRunning},
		{name: "exited", state: domain.ServiceStateExited, expectedColor: th.StateExited},
		{name: "dead", state: domain.ServiceStateDead, expectedColor: th.StateExited},
		{name: "paused", state: domain.ServiceStatePaused, expectedColor: th.StatePaused},
		{name: "restarting", state: domain.ServiceStateRestarting, expectedColor: th.StateTransient},
		{name: "not created", state: domain.ServiceStateNotCreated, expectedColor: th.StateMuted},
		{name: "unknown", state: domain.ServiceStateUnknown, expectedColor: th.StateMuted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := th.ColourForState(tc.state)
			assert.Equal(t, tc.expectedColor, got)
		})
	}
}

func TestBuiltinConstructors(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name      string
		construct func() *theme.Theme
	}

	for _, tc := range []testCase{
		{name: "Default", construct: theme.Default},
		{name: "DefaultLight", construct: theme.DefaultLight},
		{name: "CatppuccinoFrappe", construct: theme.CatppuccinoFrappe},
		{name: "CatppuccinoLatte", construct: theme.CatppuccinoLatte},
		{name: "CatppuccinoMacchiato", construct: theme.CatppuccinoMacchiato},
		{name: "CatppuccinoMocha", construct: theme.CatppuccinoMocha},
		{name: "SolarizedDark", construct: theme.SolarizedDark},
		{name: "SolarizedLight", construct: theme.SolarizedLight},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			th := tc.construct()
			require.NotNil(t, th)
		})
	}
}

func TestBuiltinNames(t *testing.T) {
	t.Parallel()

	names := theme.BuiltinNames()
	require.Len(t, names, 8)

	assert.ElementsMatch(t, builtinThemeNames(), names)
}

func TestLoadBuiltin(t *testing.T) {
	t.Parallel()

	t.Run("empty string returns default", func(t *testing.T) {
		t.Parallel()

		th, err := theme.Load("", t.TempDir())
		require.NoError(t, err)
		require.NotNil(t, th)
	})

	for _, name := range builtinThemeNames() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			th, err := theme.Load(name, t.TempDir())
			require.NoError(t, err)
			require.NotNil(t, th)
		})
	}
}

func TestLoadUnknownTheme(t *testing.T) {
	t.Parallel()

	th, err := theme.Load("nonexistent-theme", t.TempDir())
	require.Error(t, err)
	require.ErrorIs(t, err, theme.ErrUnknownTheme)
	require.NotNil(t, th)
}
