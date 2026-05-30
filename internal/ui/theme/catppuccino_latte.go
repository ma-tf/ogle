package theme

import "charm.land/lipgloss/v2"

// Catppuccin Latte palette — https://catppuccin.com/palette
//
//nolint:unused,gochecknoglobals // package-level colour definitions for Catppuccino Latte theme
var (
	latteRosewater = lipgloss.Color("#dc8a78")
	latteFlamingo  = lipgloss.Color("#dd7878")
	lattePink      = lipgloss.Color("#ea76cb")
	latteMauve     = lipgloss.Color("#8839ef")
	latteRed       = lipgloss.Color("#d20f39")
	latteMaroon    = lipgloss.Color("#e64553")
	lattePeach     = lipgloss.Color("#fe640b")
	latteYellow    = lipgloss.Color("#df8e1d")
	latteGreen     = lipgloss.Color("#40a02b")
	latteTeal      = lipgloss.Color("#179299")
	latteSky       = lipgloss.Color("#04a5e5")
	latteSapphire  = lipgloss.Color("#209fb5")
	latteBlue      = lipgloss.Color("#1e66f5")
	latteLavender  = lipgloss.Color("#7287fd")
	latteText      = lipgloss.Color("#4c4f69")
	latteSubtext1  = lipgloss.Color("#5c5f77")
	latteSubtext0  = lipgloss.Color("#6c6f85")
	latteOverlay2  = lipgloss.Color("#7c7f93")
	latteOverlay1  = lipgloss.Color("#8c8fa1")
	latteOverlay0  = lipgloss.Color("#9ca0b0")
	latteSurface2  = lipgloss.Color("#acb0be")
	latteSurface1  = lipgloss.Color("#bcc0cc")
	latteSurface0  = lipgloss.Color("#ccd0da")
	latteBase      = lipgloss.Color("#eff1f5")
	latteMantle    = lipgloss.Color("#e6e9ef")
	latteCrust     = lipgloss.Color("#dce0e8")
	latteWhite     = lipgloss.Color("#ffffff")
)

// CatppuccinoLatte returns a theme based on the Catppuccin Latte palette.
//
//nolint:dupl // theme initialisers are structurally identical by design
func CatppuccinoLatte() *Theme {
	return &Theme{
		BorderFocused: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(latteLavender),
		BorderBlurred: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(latteOverlay0),
		ServiceListTitle:               lipgloss.NewStyle().Bold(true).Foreground(latteSubtext1),
		HelpKey:                        lipgloss.NewStyle().Foreground(latteText),
		HelpDesc:                       lipgloss.NewStyle().Foreground(latteSubtext0),
		HelpSep:                        lipgloss.NewStyle().Foreground(latteOverlay1),
		HelpBackground:                 latteCrust,
		ServiceListBackground:          latteCrust,
		HoverBackground:                latteSurface0,
		SelectedBackground:             latteMantle,
		Text:                           latteText,
		Subtext:                        latteSubtext0,
		StateRunning:                   latteGreen,
		StateExited:                    latteRed,
		StatePaused:                    latteYellow,
		StateTransient:                 lattePeach,
		StateMuted:                     latteOverlay1,
		ActionError:                    latteRed,
		StatusInfo:                     latteSubtext1,
		StatusBarBackground:            latteSurface0,
		TopbarBackground:               latteCrust,
		TopbarBrandText:                latteWhite,
		TopbarBrandBackground:          latteBlue,
		TopbarContextText:              latteSubtext0,
		TopbarStatusText:               latteWhite,
		TopbarDisconnectedBackground:   latteRed,
		TopbarRetryBackground:          lattePeach,
		TopbarWrapBackground:           latteYellow,
		TopbarTruncBackground:          latteYellow,
		CarouselFocused:                latteSubtext0,
		CarouselBlurred:                latteOverlay0,
		CarouselBackground:             latteCrust,
		CarouselNavBackground:          latteCrust,
		CarouselHover:                  latteText,
		CarouselEmpty:                  latteSurface0,
		CardHoverBackground:            latteMauve,
		LogPaneBackground:              latteCrust,
		BodyBackground:                 latteBase,
		AboutBackground:                latteMantle,
		AboutTitleColour:               latteMauve,
		AboutArtColour:                 latteOverlay0,
		AboutTextColour:                latteText,
		AboutLinkColour:                latteBlue,
		AboutHintColour:                latteOverlay0,
		AccordionLabel:                 latteSubtext0,
		AccordionValue:                 latteText,
		AccordionBackground:            latteCrust,
		AccordionHeaderBackground:      latteMantle,
		AccordionHeaderHoverBackground: latteBase,
	}
}
