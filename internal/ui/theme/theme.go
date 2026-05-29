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
	AboutTitleColour               color.Color // about overlay title ("ogle") foreground
	AboutArtColour                 color.Color // about overlay ASCII art foreground
	AboutTextColour                color.Color // about overlay version line foreground
	AboutLinkColour                color.Color // about overlay URL hyperlink foreground
	AboutHintColour                color.Color // about overlay close hint foreground
}

// UserThemeFile is the YAML schema for a user-defined theme override file.
type UserThemeFile struct {
	Base                                 string `yaml:"base"`
	BorderFocusedColour                  string `yaml:"borderFocusedColour"`
	BorderBlurredColour                  string `yaml:"borderBlurredColour"`
	ServiceListTitleColour               string `yaml:"serviceListTitleColour"`
	HelpKeyColour                        string `yaml:"helpKeyColour"`
	HelpDescColour                       string `yaml:"helpDescColour"`
	HelpSepColour                        string `yaml:"helpSepColour"`
	HelpBackgroundColour                 string `yaml:"helpBackgroundColour"`
	ServiceListBackgroundColour          string `yaml:"serviceListBackgroundColour"`
	HoverBackgroundColour                string `yaml:"hoverBackgroundColour"`
	SelectedBackgroundColour             string `yaml:"selectedBackgroundColour"`
	TextColour                           string `yaml:"textColour"`
	SubtextColour                        string `yaml:"subtextColour"`
	StateRunningColour                   string `yaml:"stateRunningColour"`
	StateExitedColour                    string `yaml:"stateExitedColour"`
	StatePausedColour                    string `yaml:"statePausedColour"`
	StateTransientColour                 string `yaml:"stateTransientColour"`
	StateMutedColour                     string `yaml:"stateMutedColour"`
	ActionErrorColour                    string `yaml:"actionErrorColour"`
	StatusInfoColour                     string `yaml:"statusInfoColour"`
	StatusBarBackgroundColour            string `yaml:"statusBarBackgroundColour"`
	TopbarBackgroundColour               string `yaml:"topbarBackgroundColour"`
	TopbarBrandTextColour                string `yaml:"topbarBrandTextColour"`
	TopbarBrandBackgroundColour          string `yaml:"topbarBrandBackgroundColour"`
	TopbarContextTextColour              string `yaml:"topbarContextTextColour"`
	TopbarStatusTextColour               string `yaml:"topbarStatusTextColour"`
	TopbarDisconnectedBackgroundColour   string `yaml:"topbarDisconnectedBackgroundColour"`
	TopbarRetryBackgroundColour          string `yaml:"topbarRetryBackgroundColour"`
	TopbarWrapBackgroundColour           string `yaml:"topbarWrapBackgroundColour"`
	TopbarTruncBackgroundColour          string `yaml:"topbarTruncBackgroundColour"`
	CarouselFocusedColour                string `yaml:"carouselFocusedColour"`
	CarouselBlurredColour                string `yaml:"carouselBlurredColour"`
	CarouselBackgroundColour             string `yaml:"carouselBackgroundColour"`
	CarouselNavBackgroundColour          string `yaml:"carouselNavBackgroundColour"`
	CarouselHoverColour                  string `yaml:"carouselHoverColour"`
	CarouselEmptyColour                  string `yaml:"carouselEmptyColour"`
	CardHoverBackgroundColour            string `yaml:"cardHoverBackgroundColour"`
	LogPaneBackgroundColour              string `yaml:"logPaneBackgroundColour"`
	AccordionLabelColour                 string `yaml:"accordionLabelColour"`
	AccordionValueColour                 string `yaml:"accordionValueColour"`
	AccordionBackgroundColour            string `yaml:"accordionBackgroundColour"`
	AccordionHeaderBackgroundColour      string `yaml:"accordionHeaderBackgroundColour"`
	AccordionHeaderHoverBackgroundColour string `yaml:"accordionHeaderHoverBackgroundColour"`
	BodyBackgroundColour                 string `yaml:"bodyBackgroundColour"`
	AboutBackgroundColour                string `yaml:"aboutBackgroundColour"`
	AboutTitleColour                     string `yaml:"aboutTitleColour"`
	AboutArtColour                       string `yaml:"aboutArtColour"`
	AboutTextColour                      string `yaml:"aboutTextColour"`
	AboutLinkColour                      string `yaml:"aboutLinkColour"`
	AboutHintColour                      string `yaml:"aboutHintColour"`
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
	if f.BorderFocusedColour != "" {
		result.BorderFocused = result.BorderFocused.BorderForeground(
			lipgloss.Color(f.BorderFocusedColour),
		)
	}

	if f.BorderBlurredColour != "" {
		result.BorderBlurred = result.BorderBlurred.BorderForeground(
			lipgloss.Color(f.BorderBlurredColour),
		)
	}

	if f.ServiceListTitleColour != "" {
		result.ServiceListTitle = result.ServiceListTitle.Foreground(
			lipgloss.Color(f.ServiceListTitleColour),
		)
	}

	if f.HelpKeyColour != "" {
		result.HelpKey = result.HelpKey.Foreground(lipgloss.Color(f.HelpKeyColour))
	}

	if f.HelpDescColour != "" {
		result.HelpDesc = result.HelpDesc.Foreground(lipgloss.Color(f.HelpDescColour))
	}

	if f.HelpSepColour != "" {
		result.HelpSep = result.HelpSep.Foreground(lipgloss.Color(f.HelpSepColour))
	}
}

type colorOverride struct {
	field string
	dst   *color.Color
}

func applyColorOverrides(result *Theme, f UserThemeFile) {
	overrides := []colorOverride{
		{field: f.HelpBackgroundColour, dst: &result.HelpBackground},
		{field: f.ServiceListBackgroundColour, dst: &result.ServiceListBackground},
		{field: f.HoverBackgroundColour, dst: &result.HoverBackground},
		{field: f.SelectedBackgroundColour, dst: &result.SelectedBackground},
		{field: f.TextColour, dst: &result.Text},
		{field: f.SubtextColour, dst: &result.Subtext},
		{field: f.StateRunningColour, dst: &result.StateRunning},
		{field: f.StateExitedColour, dst: &result.StateExited},
		{field: f.StatePausedColour, dst: &result.StatePaused},
		{field: f.StateTransientColour, dst: &result.StateTransient},
		{field: f.StateMutedColour, dst: &result.StateMuted},
		{field: f.ActionErrorColour, dst: &result.ActionError},
		{field: f.StatusInfoColour, dst: &result.StatusInfo},
		{field: f.StatusBarBackgroundColour, dst: &result.StatusBarBackground},
		{field: f.TopbarBackgroundColour, dst: &result.TopbarBackground},
		{field: f.TopbarBrandTextColour, dst: &result.TopbarBrandText},
		{field: f.TopbarBrandBackgroundColour, dst: &result.TopbarBrandBackground},
		{field: f.TopbarContextTextColour, dst: &result.TopbarContextText},
		{field: f.TopbarStatusTextColour, dst: &result.TopbarStatusText},
		{field: f.TopbarDisconnectedBackgroundColour, dst: &result.TopbarDisconnectedBackground},
		{field: f.TopbarRetryBackgroundColour, dst: &result.TopbarRetryBackground},
		{field: f.TopbarWrapBackgroundColour, dst: &result.TopbarWrapBackground},
		{field: f.TopbarTruncBackgroundColour, dst: &result.TopbarTruncBackground},
		{field: f.CarouselFocusedColour, dst: &result.CarouselFocused},
		{field: f.CarouselBlurredColour, dst: &result.CarouselBlurred},
		{field: f.CarouselBackgroundColour, dst: &result.CarouselBackground},
		{field: f.CarouselNavBackgroundColour, dst: &result.CarouselNavBackground},
		{field: f.CarouselHoverColour, dst: &result.CarouselHover},
		{field: f.CarouselEmptyColour, dst: &result.CarouselEmpty},
		{field: f.CardHoverBackgroundColour, dst: &result.CardHoverBackground},
		{field: f.LogPaneBackgroundColour, dst: &result.LogPaneBackground},
		{field: f.AccordionLabelColour, dst: &result.AccordionLabel},
		{field: f.AccordionValueColour, dst: &result.AccordionValue},
		{field: f.AccordionBackgroundColour, dst: &result.AccordionBackground},
		{field: f.AccordionHeaderBackgroundColour, dst: &result.AccordionHeaderBackground},
		{
			field: f.AccordionHeaderHoverBackgroundColour,
			dst:   &result.AccordionHeaderHoverBackground,
		},
		{field: f.BodyBackgroundColour, dst: &result.BodyBackground},
		{field: f.AboutBackgroundColour, dst: &result.AboutBackground},
		{field: f.AboutTitleColour, dst: &result.AboutTitleColour},
		{field: f.AboutArtColour, dst: &result.AboutArtColour},
		{field: f.AboutTextColour, dst: &result.AboutTextColour},
		{field: f.AboutLinkColour, dst: &result.AboutLinkColour},
		{field: f.AboutHintColour, dst: &result.AboutHintColour},
	}

	for _, o := range overrides {
		if o.field != "" {
			*o.dst = lipgloss.Color(o.field)
		}
	}
}
