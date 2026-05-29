package theme

import "charm.land/lipgloss/v2"

// Solarized palette — https://ethanschoonover.com/solarized/
//
//nolint:unused,gochecknoglobals // package-level colour definitions for Solarized Dark theme
var (
	solarizedBase03  = lipgloss.Color("#002b36")
	solarizedBase02  = lipgloss.Color("#073642")
	solarizedBase01  = lipgloss.Color("#586e75")
	solarizedBase00  = lipgloss.Color("#657b83")
	solarizedBase0   = lipgloss.Color("#839496")
	solarizedBase1   = lipgloss.Color("#93a1a1")
	solarizedBase2   = lipgloss.Color("#eee8d5")
	solarizedBase3   = lipgloss.Color("#fdf6e3")
	solarizedYellow  = lipgloss.Color("#b58900")
	solarizedOrange  = lipgloss.Color("#cb4b16")
	solarizedRed     = lipgloss.Color("#dc322f")
	solarizedMagenta = lipgloss.Color("#d33682")
	solarizedViolet  = lipgloss.Color("#6c71c4")
	solarizedBlue    = lipgloss.Color("#268bd2")
	solarizedCyan    = lipgloss.Color("#2aa198")
	solarizedGreen   = lipgloss.Color("#859900")
	solarizedWhite   = lipgloss.Color("#ffffff")
)

// SolarizedDark returns a dark-background theme based on the Solarized palette.
//
//nolint:dupl // theme initialisers are structurally identical by design
func SolarizedDark() *Theme {
	return &Theme{
		BorderFocused: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(solarizedBlue),
		BorderBlurred: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(solarizedBase01),
		ServiceListTitle:               lipgloss.NewStyle().Bold(true).Foreground(solarizedBase0),
		HelpKey:                        lipgloss.NewStyle().Foreground(solarizedBase0),
		HelpDesc:                       lipgloss.NewStyle().Foreground(solarizedBase01),
		HelpSep:                        lipgloss.NewStyle().Foreground(solarizedBase01),
		HelpBackground:                 solarizedBase02,
		ServiceListBackground:          solarizedBase03,
		HoverBackground:                solarizedBase02,
		SelectedBackground:             solarizedBase02,
		Text:                           solarizedBase0,
		Subtext:                        solarizedBase01,
		StateRunning:                   solarizedGreen,
		StateExited:                    solarizedRed,
		StatePaused:                    solarizedYellow,
		StateTransient:                 solarizedOrange,
		StateMuted:                     solarizedBase01,
		ActionError:                    solarizedRed,
		StatusInfo:                     solarizedBase1,
		StatusBarBackground:            solarizedBase01,
		TopbarBackground:               solarizedBase02,
		TopbarBrandText:                solarizedWhite,
		TopbarBrandBackground:          solarizedBlue,
		TopbarContextText:              solarizedBase0,
		TopbarStatusText:               solarizedWhite,
		TopbarDisconnectedBackground:   solarizedRed,
		TopbarRetryBackground:          solarizedOrange,
		TopbarWrapBackground:           solarizedYellow,
		TopbarTruncBackground:          solarizedYellow,
		CarouselFocused:                solarizedBase1,
		CarouselBlurred:                solarizedBase01,
		CarouselBackground:             solarizedBase03,
		CarouselNavBackground:          solarizedBase03,
		CarouselHover:                  solarizedBase0,
		CarouselEmpty:                  solarizedBase02,
		CardHoverBackground:            solarizedBase02,
		LogPaneBackground:              solarizedBase03,
		BodyBackground:                 solarizedBase03,
		AboutBackground:                solarizedBase02,
		AboutTitleColour:               solarizedBlue,
		AboutArtColour:                 solarizedBase01,
		AboutTextColour:                solarizedBase0,
		AboutLinkColour:                solarizedCyan,
		AboutHintColour:                solarizedBase01,
		AccordionLabel:                 solarizedBase01,
		AccordionValue:                 solarizedBase0,
		AccordionBackground:            solarizedBase03,
		AccordionHeaderBackground:      solarizedBase02,
		AccordionHeaderHoverBackground: solarizedBase02,
	}
}
