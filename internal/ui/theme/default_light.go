package theme

import "charm.land/lipgloss/v2"

// DefaultLight returns a light-themed variation of the Default theme using the
// same ANSI base palette.
//
//nolint:dupl // theme initialisers are structurally identical by design
func DefaultLight() *Theme {
	return &Theme{
		BorderFocused: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(defaultMagenta),
		BorderBlurred: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(defaultDarkGrey),
		ServiceListTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(defaultDarkGrey),
		HelpKey:                        lipgloss.NewStyle().Foreground(defaultBlack),
		HelpDesc:                       lipgloss.NewStyle().Foreground(defaultDarkGrey),
		HelpSep:                        lipgloss.NewStyle().Foreground(defaultBrightBlack),
		HelpBackground:                 defaultWhite,
		ServiceListBackground:          defaultWhite,
		HoverBackground:                defaultBrightBlack,
		SelectedBackground:             defaultMagenta,
		Text:                           defaultBlack,
		Subtext:                        defaultDarkGrey,
		StateRunning:                   defaultGreen,
		StateExited:                    defaultBrightRed,
		StatePaused:                    defaultYellow,
		StateTransient:                 defaultBrightYellow,
		StateMuted:                     defaultBrightBlack,
		ActionError:                    defaultBrightRed,
		StatusInfo:                     defaultWhite,
		StatusBarBackground:            defaultBrightBlack,
		TopbarBackground:               defaultWhite,
		TopbarBrandText:                defaultWhite,
		TopbarBrandBackground:          defaultBlue,
		TopbarContextText:              defaultBrightBlack,
		TopbarStatusText:               defaultWhite,
		TopbarDisconnectedBackground:   defaultRed,
		TopbarRetryBackground:          defaultBrightYellow,
		TopbarWrapBackground:           defaultGreen,
		TopbarTruncBackground:          defaultYellow,
		CarouselFocused:                defaultBlack,
		CarouselBlurred:                defaultBrightBlack,
		CarouselBackground:             defaultWhite,
		CarouselNavBackground:          defaultWhite,
		CarouselHover:                  defaultBlack,
		CarouselEmpty:                  defaultBrightBlack,
		CardHoverBackground:            defaultBrightBlack,
		LogPaneBackground:              defaultWhite,
		BodyBackground:                 defaultWhite,
		AboutBackground:                defaultBrightBlack,
		AboutTitleColour:               defaultMagenta,
		AboutArtColour:                 defaultDarkGrey,
		AboutTextColour:                defaultBlack,
		AboutLinkColour:                defaultBlue,
		AboutHintColour:                defaultDarkGrey,
		AccordionLabel:                 defaultDarkGrey,
		AccordionValue:                 defaultBlack,
		AccordionBackground:            defaultWhite,
		AccordionHeaderBackground:      defaultBrightBlack,
		AccordionHeaderHoverBackground: defaultBrightBlack,
	}
}
