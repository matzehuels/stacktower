package cli

import "github.com/charmbracelet/lipgloss"

// Color palette - consistent across all CLI commands
var (
	// Primary colors
	ColorPrimary   = lipgloss.Color("62")  // Purple - brand color
	ColorSecondary = lipgloss.Color("39")  // Cyan - links, highlights
	ColorAccent    = lipgloss.Color("212") // Pink - important values

	// Status colors
	ColorSuccess = lipgloss.Color("42")  // Green
	ColorWarning = lipgloss.Color("214") // Orange
	ColorError   = lipgloss.Color("196") // Red

	// Neutral colors
	ColorDim    = lipgloss.Color("243") // Gray - secondary text
	ColorMuted  = lipgloss.Color("250") // Light gray
	ColorBright = lipgloss.Color("15")  // White
)

// Shared styles
var (
	// Text styles
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	StyleHeading = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBright)

	StyleHighlight = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent)

	StyleLink = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Underline(true)

	StyleDim = lipgloss.NewStyle().
			Foreground(ColorDim)

	StyleLabel = lipgloss.NewStyle().
			Foreground(ColorDim).
			Width(8)

	// Status styles
	StyleSuccess = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSuccess)

	StyleWarning = lipgloss.NewStyle().
			Foreground(ColorWarning)

	StyleError = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorError)

	// Container styles
	StyleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)

	StyleBoxSubtle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDim).
			Padding(0, 1)
)
