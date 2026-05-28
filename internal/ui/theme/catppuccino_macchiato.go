package theme

import "charm.land/lipgloss/v2"

// Catppuccin Macchiato palette — https://catppuccin.com/palette
//
//nolint:unused,gochecknoglobals // package-level colour definitions for Catppuccino Macchiato theme
var (
	macchiatoRosewater = lipgloss.Color("#f4dbd6")
	macchiatoFlamingo  = lipgloss.Color("#f0c6c6")
	macchiatoPink      = lipgloss.Color("#f5bde6")
	macchiatoMauve     = lipgloss.Color("#c6a0f6")
	macchiatoRed       = lipgloss.Color("#ed8796")
	macchiatoMaroon    = lipgloss.Color("#ee99a0")
	macchiatoPeach     = lipgloss.Color("#f5a97f")
	macchiatoYellow    = lipgloss.Color("#eed49f")
	macchiatoGreen     = lipgloss.Color("#a6da95")
	macchiatoTeal      = lipgloss.Color("#8bd5ca")
	macchiatoSky       = lipgloss.Color("#91d7e3")
	macchiatoSapphire  = lipgloss.Color("#7dc4e4")
	macchiatoBlue      = lipgloss.Color("#8aadf4")
	macchiatoLavender  = lipgloss.Color("#b7bdf8")
	macchiatoText      = lipgloss.Color("#cad3f5")
	macchiatoSubtext1  = lipgloss.Color("#b8c0e0")
	macchiatoSubtext0  = lipgloss.Color("#a5adcb")
	macchiatoOverlay2  = lipgloss.Color("#939ab7")
	macchiatoOverlay1  = lipgloss.Color("#8087a2")
	macchiatoOverlay0  = lipgloss.Color("#6e738d")
	macchiatoSurface2  = lipgloss.Color("#5b6078")
	macchiatoSurface1  = lipgloss.Color("#494d64")
	macchiatoSurface0  = lipgloss.Color("#363a4f")
	macchiatoBase      = lipgloss.Color("#24273a")
	macchiatoMantle    = lipgloss.Color("#1e2030")
	macchiatoCrust     = lipgloss.Color("#181926")
	macchiatoWhite     = lipgloss.Color("#ffffff")
)

// CatppuccinoMacchiato returns a theme based on the Catppuccin Macchiato palette.
//
//nolint:dupl // theme initialisers are structurally identical by design
func CatppuccinoMacchiato() *Theme {
	return &Theme{
		BorderFocused: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(macchiatoLavender),
		BorderBlurred: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(macchiatoOverlay0),
		ServiceListTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(macchiatoSubtext1),
		HelpKey:                        lipgloss.NewStyle().Foreground(macchiatoText),
		HelpDesc:                       lipgloss.NewStyle().Foreground(macchiatoSubtext0),
		HelpSep:                        lipgloss.NewStyle().Foreground(macchiatoOverlay1),
		HelpBackground:                 macchiatoCrust,
		ServiceListBackground:          macchiatoCrust,
		HoverBackground:                macchiatoSurface0,
		SelectedBackground:             macchiatoMantle,
		Text:                           macchiatoText,
		Subtext:                        macchiatoSubtext0,
		StateRunning:                   macchiatoGreen,
		StateExited:                    macchiatoRed,
		StatePaused:                    macchiatoYellow,
		StateTransient:                 macchiatoPeach,
		StateMuted:                     macchiatoOverlay1,
		ActionError:                    macchiatoRed,
		StatusInfo:                     macchiatoSubtext1,
		StatusBarBackground:            macchiatoSurface0,
		TopbarBackground:               macchiatoCrust,
		TopbarBrandText:                macchiatoWhite,
		TopbarBrandBackground:          macchiatoBlue,
		TopbarContextText:              macchiatoSubtext0,
		TopbarStatusText:               macchiatoWhite,
		TopbarDisconnectedBackground:   macchiatoRed,
		TopbarRetryBackground:          macchiatoPeach,
		CarouselFocused:                macchiatoSubtext0,
		CarouselBlurred:                macchiatoOverlay0,
		CarouselBackground:             macchiatoCrust,
		CarouselNavBackground:          macchiatoCrust,
		CarouselHover:                  macchiatoText,
		CarouselEmpty:                  macchiatoSurface0,
		CardHoverBackground:            macchiatoSurface0,
		LogPaneBackground:              macchiatoCrust,
		BodyBackground:                 macchiatoBase,
		AboutBackground:                macchiatoBase,
		AboutTitleColor:                macchiatoMauve,
		AboutArtColor:                  macchiatoOverlay0,
		AboutTextColor:                 macchiatoText,
		AboutLinkColor:                 macchiatoBlue,
		AboutHintColor:                 macchiatoOverlay0,
		AccordionLabel:                 macchiatoSubtext0,
		AccordionValue:                 macchiatoText,
		AccordionBackground:            macchiatoCrust,
		AccordionHeaderBackground:      macchiatoMantle,
		AccordionHeaderHoverBackground: macchiatoBase,
	}
}
