package theme

import "charm.land/lipgloss/v2"

// Catppuccin Frappé palette — https://catppuccin.com/palette
//
//nolint:unused,gochecknoglobals // package-level colour definitions for Catppuccino Frappé theme
var (
	frappeRosewater = lipgloss.Color("#f2d5cf")
	frappeFlamingo  = lipgloss.Color("#eebebe")
	frappePink      = lipgloss.Color("#f4b8e4")
	frappeMauve     = lipgloss.Color("#ca9ee6")
	frappeRed       = lipgloss.Color("#e78284")
	frappeMaroon    = lipgloss.Color("#ea999c")
	frappePeach     = lipgloss.Color("#ef9f76")
	frappeYellow    = lipgloss.Color("#e5c890")
	frappeGreen     = lipgloss.Color("#a6d189")
	frappeTeal      = lipgloss.Color("#81c8be")
	frappeSky       = lipgloss.Color("#99d1db")
	frappeSapphire  = lipgloss.Color("#85c1dc")
	frappeBlue      = lipgloss.Color("#8caaee")
	frappeLavender  = lipgloss.Color("#babbf1")
	frappeText      = lipgloss.Color("#c6d0f5")
	frappeSubtext1  = lipgloss.Color("#b5bfe2")
	frappeSubtext0  = lipgloss.Color("#a5adce")
	frappeOverlay2  = lipgloss.Color("#949cbb")
	frappeOverlay1  = lipgloss.Color("#838ba7")
	frappeOverlay0  = lipgloss.Color("#737994")
	frappeSurface2  = lipgloss.Color("#626880")
	frappeSurface1  = lipgloss.Color("#51576d")
	frappeSurface0  = lipgloss.Color("#414559")
	frappeBase      = lipgloss.Color("#303446")
	frappeMantle    = lipgloss.Color("#292c3c")
	frappeCrust     = lipgloss.Color("#232634")
	frappeWhite     = lipgloss.Color("#ffffff")
)

// CatppuccinoFrappe returns a theme based on the Catppuccin Frappé palette.
//
//nolint:dupl // theme initialisers are structurally identical by design
func CatppuccinoFrappe() *Theme {
	return &Theme{
		BorderFocused: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(frappeLavender),
		BorderBlurred: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(frappeOverlay0),
		ServiceListTitle:               lipgloss.NewStyle().Bold(true).Foreground(frappeSubtext1),
		HelpKey:                        lipgloss.NewStyle().Foreground(frappeText),
		HelpDesc:                       lipgloss.NewStyle().Foreground(frappeSubtext0),
		HelpSep:                        lipgloss.NewStyle().Foreground(frappeOverlay1),
		HelpBackground:                 frappeCrust,
		ServiceListBackground:          frappeCrust,
		HoverBackground:                frappeSurface0,
		SelectedBackground:             frappeMantle,
		Text:                           frappeText,
		Subtext:                        frappeSubtext0,
		StateRunning:                   frappeGreen,
		StateExited:                    frappeRed,
		StatePaused:                    frappeYellow,
		StateTransient:                 frappePeach,
		StateMuted:                     frappeOverlay1,
		ActionError:                    frappeRed,
		StatusInfo:                     frappeSubtext1,
		StatusBarBackground:            frappeSurface0,
		TopbarBackground:               frappeCrust,
		TopbarBrandText:                frappeWhite,
		TopbarBrandBackground:          frappeBlue,
		TopbarContextText:              frappeSubtext0,
		TopbarStatusText:               frappeWhite,
		TopbarDisconnectedBackground:   frappeRed,
		TopbarRetryBackground:          frappePeach,
		TopbarWrapBackground:           frappeYellow,
		TopbarTruncBackground:          frappeYellow,
		CarouselFocused:                frappeSubtext0,
		CarouselBlurred:                frappeOverlay0,
		CarouselBackground:             frappeCrust,
		CarouselNavBackground:          frappeCrust,
		CarouselHover:                  frappeText,
		CarouselEmpty:                  frappeSurface0,
		CardHoverBackground:            frappeSurface0,
		LogPaneBackground:              frappeCrust,
		BodyBackground:                 frappeBase,
		AboutBackground:                frappeMantle,
		AboutTitleColour:               frappeMauve,
		AboutArtColour:                 frappeOverlay0,
		AboutTextColour:                frappeText,
		AboutLinkColour:                frappeBlue,
		AboutHintColour:                frappeOverlay0,
		AccordionLabel:                 frappeSubtext0,
		AccordionValue:                 frappeText,
		AccordionBackground:            frappeCrust,
		AccordionHeaderBackground:      frappeMantle,
		AccordionHeaderHoverBackground: frappeBase,
	}
}
