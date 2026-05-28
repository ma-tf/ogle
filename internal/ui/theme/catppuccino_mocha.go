package theme

import "charm.land/lipgloss/v2"

// Catppuccin Mocha palette — https://catppuccin.com/palette
//
//nolint:unused,gochecknoglobals // package-level colour definitions for Catppuccino Mocha theme
var (
	mochaRosewater = lipgloss.Color("#f5e0dc")
	mochaFlamingo  = lipgloss.Color("#f2cdcd")
	mochaPink      = lipgloss.Color("#f5c2e7")
	mochaMauve     = lipgloss.Color("#cba6f7")
	mochaRed       = lipgloss.Color("#f38ba8")
	mochaMaroon    = lipgloss.Color("#eba0ac")
	mochaPeach     = lipgloss.Color("#fab387")
	mochaYellow    = lipgloss.Color("#f9e2af")
	mochaGreen     = lipgloss.Color("#a6e3a1")
	mochaTeal      = lipgloss.Color("#94e2d5")
	mochaSky       = lipgloss.Color("#89dceb")
	mochaSapphire  = lipgloss.Color("#74c7ec")
	mochaBlue      = lipgloss.Color("#89b4fa")
	mochaLavender  = lipgloss.Color("#b4befe")
	mochaText      = lipgloss.Color("#cdd6f4")
	mochaSubtext1  = lipgloss.Color("#bac2de")
	mochaSubtext0  = lipgloss.Color("#a6adc8")
	mochaOverlay2  = lipgloss.Color("#9399b2")
	mochaOverlay1  = lipgloss.Color("#7f849c")
	mochaOverlay0  = lipgloss.Color("#6c7086")
	mochaSurface2  = lipgloss.Color("#585b70")
	mochaSurface1  = lipgloss.Color("#45475a")
	mochaSurface0  = lipgloss.Color("#313244")
	mochaBase      = lipgloss.Color("#1e1e2e")
	mochaMantle    = lipgloss.Color("#181825")
	mochaCrust     = lipgloss.Color("#11111b")
	mochaWhite     = lipgloss.Color("#ffffff")
)

// CatppuccinoMocha returns a theme based on the Catppuccin Mocha palette.
//
//nolint:dupl // theme initialisers are structurally identical by design
func CatppuccinoMocha() *Theme {
	return &Theme{
		BorderFocused: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(mochaLavender),
		BorderBlurred: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(mochaOverlay0),
		ServiceListTitle:               lipgloss.NewStyle().Bold(true).Foreground(mochaSubtext1),
		HelpKey:                        lipgloss.NewStyle().Foreground(mochaText),
		HelpDesc:                       lipgloss.NewStyle().Foreground(mochaSubtext0),
		HelpSep:                        lipgloss.NewStyle().Foreground(mochaOverlay1),
		HelpBackground:                 mochaCrust,
		ServiceListBackground:          mochaCrust,
		HoverBackground:                mochaSurface0,
		SelectedBackground:             mochaMantle,
		Text:                           mochaText,
		Subtext:                        mochaSubtext0,
		StateRunning:                   mochaGreen,
		StateExited:                    mochaRed,
		StatePaused:                    mochaYellow,
		StateTransient:                 mochaPeach,
		StateMuted:                     mochaOverlay1,
		ActionError:                    mochaRed,
		StatusInfo:                     mochaSubtext1,
		StatusBarBackground:            mochaSurface0,
		TopbarBackground:               mochaCrust,
		TopbarBrandText:                mochaWhite,
		TopbarBrandBackground:          mochaBlue,
		TopbarContextText:              mochaSubtext0,
		TopbarStatusText:               mochaWhite,
		TopbarDisconnectedBackground:   mochaRed,
		TopbarRetryBackground:          mochaPeach,
		CarouselFocused:                mochaSubtext0,
		CarouselBlurred:                mochaOverlay0,
		CarouselBackground:             mochaCrust,
		CarouselNavBackground:          mochaCrust,
		CarouselHover:                  mochaText,
		CarouselEmpty:                  mochaSurface0,
		CardHoverBackground:            mochaSurface0,
		LogPaneBackground:              mochaCrust,
		BodyBackground:                 mochaBase,
		AboutBackground:                mochaBase,
		AboutTitleColor:                mochaMauve,
		AboutArtColor:                  mochaOverlay0,
		AboutTextColor:                 mochaText,
		AboutLinkColor:                 mochaBlue,
		AboutHintColor:                 mochaOverlay0,
		AccordionLabel:                 mochaSubtext0,
		AccordionValue:                 mochaText,
		AccordionBackground:            mochaCrust,
		AccordionHeaderBackground:      mochaMantle,
		AccordionHeaderHoverBackground: mochaBase,
	}
}
