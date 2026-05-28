package theme

import "charm.land/lipgloss/v2"

// SolarizedLight returns a light-background theme based on the Solarized palette.
//
//nolint:dupl // theme initialisers are structurally identical by design
func SolarizedLight() *Theme {
	return &Theme{
		BorderFocused: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(solarizedBlue),
		BorderBlurred: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(solarizedBase1),
		ServiceListTitle:               lipgloss.NewStyle().Bold(true).Foreground(solarizedBase00),
		HelpKey:                        lipgloss.NewStyle().Foreground(solarizedBase00),
		HelpDesc:                       lipgloss.NewStyle().Foreground(solarizedBase1),
		HelpSep:                        lipgloss.NewStyle().Foreground(solarizedBase1),
		HelpBackground:                 solarizedBase2,
		ServiceListBackground:          solarizedBase3,
		HoverBackground:                solarizedBase2,
		SelectedBackground:             solarizedBase2,
		Text:                           solarizedBase00,
		Subtext:                        solarizedBase1,
		StateRunning:                   solarizedGreen,
		StateExited:                    solarizedRed,
		StatePaused:                    solarizedYellow,
		StateTransient:                 solarizedOrange,
		StateMuted:                     solarizedBase1,
		ActionError:                    solarizedRed,
		StatusInfo:                     solarizedBase01,
		StatusBarBackground:            solarizedBase2,
		TopbarBackground:               solarizedBase2,
		TopbarBrandText:                solarizedWhite,
		TopbarBrandBackground:          solarizedBlue,
		TopbarContextText:              solarizedBase00,
		TopbarStatusText:               solarizedWhite,
		TopbarDisconnectedBackground:   solarizedRed,
		TopbarRetryBackground:          solarizedOrange,
		TopbarWrapBackground:           solarizedGreen,
		TopbarTruncBackground:          solarizedYellow,
		CarouselFocused:                solarizedBase01,
		CarouselBlurred:                solarizedBase1,
		CarouselBackground:             solarizedBase3,
		CarouselNavBackground:          solarizedBase3,
		CarouselHover:                  solarizedBase00,
		CarouselEmpty:                  solarizedBase2,
		CardHoverBackground:            solarizedBase2,
		LogPaneBackground:              solarizedBase3,
		BodyBackground:                 solarizedBase3,
		AboutBackground:                solarizedBase2,
		AboutTitleColor:                solarizedBlue,
		AboutArtColor:                  solarizedBase1,
		AboutTextColor:                 solarizedBase00,
		AboutLinkColor:                 solarizedCyan,
		AboutHintColor:                 solarizedBase1,
		AccordionLabel:                 solarizedBase1,
		AccordionValue:                 solarizedBase00,
		AccordionBackground:            solarizedBase3,
		AccordionHeaderBackground:      solarizedBase2,
		AccordionHeaderHoverBackground: solarizedBase2,
	}
}
