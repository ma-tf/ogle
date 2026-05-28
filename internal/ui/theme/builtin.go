package theme

import "charm.land/lipgloss/v2"

// BuiltinNames returns the names of all built-in themes in display order.
func BuiltinNames() []string {
	return []string{
		"default",
		"default_light",
		"catppuccino_frappe",
		"catppuccino_latte",
		"catppuccino_macchiato",
		"catppuccino_mocha",
		"solarized_dark",
		"solarized_light",
	}
}

// ANSI 16-colour palette for the Default theme
//
//nolint:unused,gochecknoglobals // package-level colour definitions for Default theme
var (
	defaultBlack         = lipgloss.Color("#0B0B0B")
	defaultRed           = lipgloss.Color("#cc0000")
	defaultGreen         = lipgloss.Color("#87d700")
	defaultYellow        = lipgloss.Color("#ffd75f")
	defaultBlue          = lipgloss.Color("#5f87ff")
	defaultMagenta       = lipgloss.Color("#af87ff")
	defaultCyan          = lipgloss.Color("#5fafaf")
	defaultWhite         = lipgloss.Color("#e4e4e4")
	defaultBrightBlack   = lipgloss.Color("#585858")
	defaultBrightRed     = lipgloss.Color("#ff5f5f")
	defaultBrightGreen   = lipgloss.Color("#a8ff60")
	defaultBrightYellow  = lipgloss.Color("#ffaf5f")
	defaultBrightBlue    = lipgloss.Color("#5fd7ff")
	defaultBrightMagenta = lipgloss.Color("#ff87d7")
	defaultBrightCyan    = lipgloss.Color("#5fafd7")
	defaultBrightWhite   = lipgloss.Color("#ffffff")
	defaultDarkGrey      = lipgloss.Color("#333333")
)

// Default returns the default built-in theme.
//
//nolint:dupl // theme initialisers are structurally identical by design
func Default() *Theme {
	return &Theme{
		BorderFocused: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(defaultMagenta),
		BorderBlurred: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(defaultBrightBlack),
		ServiceListTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(defaultBrightBlack),
		HelpKey:                        lipgloss.NewStyle().Foreground(defaultWhite),
		HelpDesc:                       lipgloss.NewStyle().Foreground(defaultBrightBlack),
		HelpSep:                        lipgloss.NewStyle().Foreground(defaultBrightBlack),
		HelpBackground:                 defaultBlack,
		ServiceListBackground:          defaultBlack,
		HoverBackground:                defaultBlack,
		SelectedBackground:             defaultDarkGrey,
		Text:                           defaultWhite,
		Subtext:                        defaultBrightBlack,
		StateRunning:                   defaultGreen,
		StateExited:                    defaultBrightRed,
		StatePaused:                    defaultYellow,
		StateTransient:                 defaultBrightYellow,
		StateMuted:                     defaultBrightBlack,
		ActionError:                    defaultBrightRed,
		StatusInfo:                     defaultWhite,
		StatusBarBackground:            defaultDarkGrey,
		TopbarBackground:               defaultBlack,
		TopbarBrandText:                defaultBrightWhite,
		TopbarBrandBackground:          defaultBlue,
		TopbarContextText:              defaultBrightBlack,
		TopbarStatusText:               defaultBrightWhite,
		TopbarDisconnectedBackground:   defaultRed,
		TopbarRetryBackground:          defaultBrightYellow,
		TopbarWrapBackground:           defaultGreen,
		TopbarTruncBackground:          defaultYellow,
		CarouselFocused:                defaultWhite,
		CarouselBlurred:                defaultBrightBlack,
		CarouselBackground:             defaultBlack,
		CarouselNavBackground:          defaultBlack,
		CarouselHover:                  defaultBrightWhite,
		CarouselEmpty:                  defaultDarkGrey,
		CardHoverBackground:            defaultDarkGrey,
		LogPaneBackground:              defaultBlack,
		BodyBackground:                 defaultBlack,
		AboutBackground:                defaultDarkGrey,
		AboutTitleColor:                defaultMagenta,
		AboutArtColor:                  defaultBrightBlack,
		AboutTextColor:                 defaultWhite,
		AboutLinkColor:                 defaultBlue,
		AboutHintColor:                 defaultBrightBlack,
		AccordionLabel:                 defaultWhite,
		AccordionValue:                 defaultWhite,
		AccordionBackground:            defaultBlack,
		AccordionHeaderBackground:      defaultDarkGrey,
		AccordionHeaderHoverBackground: defaultBrightBlack,
	}
}
