// Package theme defines the Theme type, built-in themes, and user theme loading.
// lipgloss must not be imported outside the UI layer; this package is the
// single source of all style definitions.
package theme

import (
	"errors"
	"fmt"
	"image/color"
	"os"
	"path/filepath"

	"charm.land/lipgloss/v2"
	"go.yaml.in/yaml/v3"

	"github.com/ma-tf/ogle/internal/domain"
)

// ErrUnknownTheme is returned by Load when the name does not match any
// built-in theme and no user file exists for it.
var ErrUnknownTheme = errors.New("unknown theme")

// ColourForState maps a Service State to the theme colour for that state.
func (t *Theme) ColourForState(s domain.ServiceState) color.Color {
	switch s {
	case domain.ServiceStateRunning:
		return t.StateRunning
	case domain.ServiceStateExited, domain.ServiceStateDead:
		return t.StateExited
	case domain.ServiceStatePaused:
		return t.StatePaused
	case domain.ServiceStateRestarting:
		return t.StateTransient
	case domain.ServiceStateNotCreated, domain.ServiceStateUnknown:
		return t.StateMuted
	default:
		return t.StateMuted
	}
}

// Theme holds the complete set of themeable style values for the UI layer.
// BorderFocused and BorderBlurred pre-compose lipgloss.NormalBorder(); call
// sites extend with Width/Height only.
type Theme struct {
	BorderFocused                  lipgloss.Style
	BorderBlurred                  lipgloss.Style
	ServiceListTitle               lipgloss.Style
	HelpKey                        lipgloss.Style // key binding label (e.g. "ctrl+c")
	HelpDesc                       lipgloss.Style // key binding description (e.g. "quit")
	HelpSep                        lipgloss.Style // separator and ellipsis
	HelpBackground                 color.Color    // full-width background fill behind the help bar
	ServiceListBackground          color.Color
	HoverBackground                color.Color
	SelectedBackground             color.Color
	Text                           color.Color // body copy / primary foreground
	Subtext                        color.Color // labels, secondary text
	StateRunning                   color.Color // running
	StateExited                    color.Color // exited / dead
	StatePaused                    color.Color // paused
	StateTransient                 color.Color // restarting, action in-flight
	StateMuted                     color.Color // not created, unknown, nil runtime
	ActionError                    color.Color // error suffix text
	StatusInfo                     color.Color // info-level status bar text
	StatusBarBackground            color.Color // status bar background tint
	TopbarBackground               color.Color // top bar background tint
	TopbarBrandText                color.Color // "ogle" brand text foreground
	TopbarBrandBackground          color.Color // "ogle" brand text background
	TopbarContextText              color.Color // context/phase text foreground
	TopbarStatusText               color.Color // daemon status text foreground
	TopbarDisconnectedBackground   color.Color // red — "DISCONNECTED" badge
	TopbarRetryBackground          color.Color // orange — "RECONNECTING" badge
	TopbarWrapBackground           color.Color // green — "WRAP" badge background
	TopbarTruncBackground          color.Color // amber — ">>" truncation badge background
	LogPaneBackground              color.Color // log pane background fill
	CarouselFocused                color.Color
	CarouselBlurred                color.Color
	CarouselBackground             color.Color // background behind the card grid
	CarouselNavBackground          color.Color // background behind the nav bar
	CarouselHover                  color.Color // border/chevron color when hovered (not focused)
	CarouselEmpty                  color.Color // border colour for empty placeholder cards
	CardHoverBackground            color.Color // card background when hovered
	AccordionLabel                 color.Color // accordion label colour (e.g. "Image:")
	AccordionValue                 color.Color // accordion value colour
	AccordionBackground            color.Color // accordion background fill
	AccordionHeaderBackground      color.Color // header bar background (brighter than AccordionBackground)
	AccordionHeaderHoverBackground color.Color // header bar background when hovered
	BodyBackground                 color.Color // background fill behind body content
	AboutBackground                color.Color // about overlay background fill
	AboutTitleColor                color.Color // about overlay title ("ogle") foreground
	AboutArtColor                  color.Color // about overlay ASCII art foreground
	AboutTextColor                 color.Color // about overlay version line foreground
	AboutLinkColor                 color.Color // about overlay URL hyperlink foreground
	AboutHintColor                 color.Color // about overlay close hint foreground
}

// UserThemeFile is the YAML schema for a user-defined theme override file.
type UserThemeFile struct {
	Base                                string `yaml:"base"`
	BorderFocusedColor                  string `yaml:"borderFocusedColor"`
	BorderBlurredColor                  string `yaml:"borderBlurredColor"`
	ServiceListTitleColor               string `yaml:"serviceListTitleColor"`
	HelpKeyColor                        string `yaml:"helpKeyColor"`
	HelpDescColor                       string `yaml:"helpDescColor"`
	HelpSepColor                        string `yaml:"helpSepColor"`
	HelpBackgroundColor                 string `yaml:"helpBackgroundColor"`
	ServiceListBackgroundColor          string `yaml:"serviceListBackgroundColor"`
	HoverBackgroundColor                string `yaml:"hoverBackgroundColor"`
	SelectedBackgroundColor             string `yaml:"selectedBackgroundColor"`
	TextColor                           string `yaml:"textColor"`
	SubtextColor                        string `yaml:"subtextColor"`
	StateRunningColor                   string `yaml:"stateRunningColor"`
	StateExitedColor                    string `yaml:"stateExitedColor"`
	StatePausedColor                    string `yaml:"statePausedColor"`
	StateTransientColor                 string `yaml:"stateTransientColor"`
	StateMutedColor                     string `yaml:"stateMutedColor"`
	ActionErrorColor                    string `yaml:"actionErrorColor"`
	StatusInfoColor                     string `yaml:"statusInfoColor"`
	StatusBarBackgroundColor            string `yaml:"statusBarBackgroundColor"`
	TopbarBackgroundColor               string `yaml:"topbarBackgroundColor"`
	TopbarBrandTextColor                string `yaml:"topbarBrandTextColor"`
	TopbarBrandBackgroundColor          string `yaml:"topbarBrandBackgroundColor"`
	TopbarContextTextColor              string `yaml:"topbarContextTextColor"`
	TopbarStatusTextColor               string `yaml:"topbarStatusTextColor"`
	TopbarDisconnectedBackgroundColor   string `yaml:"topbarDisconnectedBackgroundColor"`
	TopbarRetryBackgroundColor          string `yaml:"topbarRetryBackgroundColor"`
	TopbarWrapBackgroundColor           string `yaml:"topbarWrapBackgroundColor"`
	TopbarTruncBackgroundColor          string `yaml:"topbarTruncBackgroundColor"`
	CarouselFocusedColor                string `yaml:"carouselFocusedColor"`
	CarouselBlurredColor                string `yaml:"carouselBlurredColor"`
	CarouselBackgroundColor             string `yaml:"carouselBackgroundColor"`
	CarouselNavBackgroundColor          string `yaml:"carouselNavBackgroundColor"`
	CarouselHoverColor                  string `yaml:"carouselHoverColor"`
	CarouselEmptyColor                  string `yaml:"carouselEmptyColor"`
	CardHoverBackgroundColor            string `yaml:"cardHoverBackgroundColor"`
	LogPaneBackgroundColor              string `yaml:"logPaneBackgroundColor"`
	AccordionLabelColor                 string `yaml:"accordionLabelColor"`
	AccordionValueColor                 string `yaml:"accordionValueColor"`
	AccordionBackgroundColor            string `yaml:"accordionBackgroundColor"`
	AccordionHeaderBackgroundColor      string `yaml:"accordionHeaderBackgroundColor"`
	AccordionHeaderHoverBackgroundColor string `yaml:"accordionHeaderHoverBackgroundColor"`
	BodyBackgroundColor                 string `yaml:"bodyBackgroundColor"`
	AboutBackgroundColor                string `yaml:"aboutBackgroundColor"`
	AboutTitleColor                     string `yaml:"aboutTitleColor"`
	AboutArtColor                       string `yaml:"aboutArtColor"`
	AboutTextColor                      string `yaml:"aboutTextColor"`
	AboutLinkColor                      string `yaml:"aboutLinkColor"`
	AboutHintColor                      string `yaml:"aboutHintColor"`
}

// Load resolves a theme by name. configDir is the directory containing
// config.yaml (typically ~/.ogle).
//
// Resolution order:
//  1. configDir/themes/<name>.yaml — user-defined theme file
//  2. Built-in theme with the given name
//
// On any resolution failure Load returns Default() and a descriptive error.
// Callers should log the error at Warn level and continue.
func Load(name, configDir string) (*Theme, error) {
	path := filepath.Join(configDir, "themes", name+".yaml")

	data, err := os.ReadFile(path)
	if err == nil {
		var f UserThemeFile
		if yamlErr := yaml.Unmarshal(data, &f); yamlErr != nil {
			return Default(), fmt.Errorf("parse theme file %q: %w", path, yamlErr)
		}

		base := builtinByName(f.Base)
		if base == nil {
			base = Default()
		}

		return ApplyOverrides(base, f), nil
	}

	t := builtinByName(name)
	if t != nil {
		return t, nil
	}

	return Default(), fmt.Errorf("%q: %w", name, ErrUnknownTheme)
}

func builtinByName(name string) *Theme {
	switch name {
	case "default", "":
		return Default()
	case "default_light":
		return DefaultLight()
	case "catppuccino_frappe":
		return CatppuccinoFrappe()
	case "catppuccino_latte":
		return CatppuccinoLatte()
	case "catppuccino_macchiato":
		return CatppuccinoMacchiato()
	case "catppuccino_mocha":
		return CatppuccinoMocha()
	case "solarized_dark":
		return SolarizedDark()
	case "solarized_light":
		return SolarizedLight()
	default:
		return nil
	}
}

// ApplyOverrides applies user-defined override values from f to a copy of t.
func ApplyOverrides(t *Theme, f UserThemeFile) *Theme {
	result := *t

	applyStyleOverrides(&result, f)
	applyColorOverrides(&result, f)

	return &result
}

func applyStyleOverrides(result *Theme, f UserThemeFile) {
	if f.BorderFocusedColor != "" {
		result.BorderFocused = result.BorderFocused.BorderForeground(
			lipgloss.Color(f.BorderFocusedColor),
		)
	}

	if f.BorderBlurredColor != "" {
		result.BorderBlurred = result.BorderBlurred.BorderForeground(
			lipgloss.Color(f.BorderBlurredColor),
		)
	}

	if f.ServiceListTitleColor != "" {
		result.ServiceListTitle = result.ServiceListTitle.Foreground(
			lipgloss.Color(f.ServiceListTitleColor),
		)
	}

	if f.HelpKeyColor != "" {
		result.HelpKey = result.HelpKey.Foreground(lipgloss.Color(f.HelpKeyColor))
	}

	if f.HelpDescColor != "" {
		result.HelpDesc = result.HelpDesc.Foreground(lipgloss.Color(f.HelpDescColor))
	}

	if f.HelpSepColor != "" {
		result.HelpSep = result.HelpSep.Foreground(lipgloss.Color(f.HelpSepColor))
	}
}

type colorOverride struct {
	field string
	dst   *color.Color
}

func applyColorOverrides(result *Theme, f UserThemeFile) {
	overrides := []colorOverride{
		{field: f.HelpBackgroundColor, dst: &result.HelpBackground},
		{field: f.ServiceListBackgroundColor, dst: &result.ServiceListBackground},
		{field: f.HoverBackgroundColor, dst: &result.HoverBackground},
		{field: f.SelectedBackgroundColor, dst: &result.SelectedBackground},
		{field: f.TextColor, dst: &result.Text},
		{field: f.SubtextColor, dst: &result.Subtext},
		{field: f.StateRunningColor, dst: &result.StateRunning},
		{field: f.StateExitedColor, dst: &result.StateExited},
		{field: f.StatePausedColor, dst: &result.StatePaused},
		{field: f.StateTransientColor, dst: &result.StateTransient},
		{field: f.StateMutedColor, dst: &result.StateMuted},
		{field: f.ActionErrorColor, dst: &result.ActionError},
		{field: f.StatusInfoColor, dst: &result.StatusInfo},
		{field: f.StatusBarBackgroundColor, dst: &result.StatusBarBackground},
		{field: f.TopbarBackgroundColor, dst: &result.TopbarBackground},
		{field: f.TopbarBrandTextColor, dst: &result.TopbarBrandText},
		{field: f.TopbarBrandBackgroundColor, dst: &result.TopbarBrandBackground},
		{field: f.TopbarContextTextColor, dst: &result.TopbarContextText},
		{field: f.TopbarStatusTextColor, dst: &result.TopbarStatusText},
		{field: f.TopbarDisconnectedBackgroundColor, dst: &result.TopbarDisconnectedBackground},
		{field: f.TopbarRetryBackgroundColor, dst: &result.TopbarRetryBackground},
		{field: f.TopbarWrapBackgroundColor, dst: &result.TopbarWrapBackground},
		{field: f.TopbarTruncBackgroundColor, dst: &result.TopbarTruncBackground},
		{field: f.CarouselFocusedColor, dst: &result.CarouselFocused},
		{field: f.CarouselBlurredColor, dst: &result.CarouselBlurred},
		{field: f.CarouselBackgroundColor, dst: &result.CarouselBackground},
		{field: f.CarouselNavBackgroundColor, dst: &result.CarouselNavBackground},
		{field: f.CarouselHoverColor, dst: &result.CarouselHover},
		{field: f.CarouselEmptyColor, dst: &result.CarouselEmpty},
		{field: f.CardHoverBackgroundColor, dst: &result.CardHoverBackground},
		{field: f.LogPaneBackgroundColor, dst: &result.LogPaneBackground},
		{field: f.AccordionLabelColor, dst: &result.AccordionLabel},
		{field: f.AccordionValueColor, dst: &result.AccordionValue},
		{field: f.AccordionBackgroundColor, dst: &result.AccordionBackground},
		{field: f.AccordionHeaderBackgroundColor, dst: &result.AccordionHeaderBackground},
		{field: f.AccordionHeaderHoverBackgroundColor, dst: &result.AccordionHeaderHoverBackground},
		{field: f.BodyBackgroundColor, dst: &result.BodyBackground},
		{field: f.AboutBackgroundColor, dst: &result.AboutBackground},
		{field: f.AboutTitleColor, dst: &result.AboutTitleColor},
		{field: f.AboutArtColor, dst: &result.AboutArtColor},
		{field: f.AboutTextColor, dst: &result.AboutTextColor},
		{field: f.AboutLinkColor, dst: &result.AboutLinkColor},
		{field: f.AboutHintColor, dst: &result.AboutHintColor},
	}

	for _, o := range overrides {
		if o.field != "" {
			*o.dst = lipgloss.Color(o.field)
		}
	}
}
