package term

import "github.com/charmbracelet/lipgloss"

// =============================================================================
// Color Palette - Warm, cohesive terminal colors
// =============================================================================

var (
	// Core colors
	colorCyan   = lipgloss.Color("36")  // Teal - primary actions
	colorGreen  = lipgloss.Color("35")  // Green - success
	colorYellow = lipgloss.Color("220") // Amber - warnings
	colorRed    = lipgloss.Color("167") // Soft red - errors
	colorBlue   = lipgloss.Color("75")  // Light blue - links

	// Neutral colors
	colorWhite = lipgloss.Color("255") // Bright white - values
	colorGray  = lipgloss.Color("245") // Gray - secondary text
	colorDim   = lipgloss.Color("240") // Dim gray - muted text
)

// =============================================================================
// Text Styles
// =============================================================================

var (
	// StyleTitle for main headings
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCyan)

	// StyleHeading for section headers
	StyleHeading = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite)

	// StyleHighlight for emphasized values (no background)
	StyleHighlight = lipgloss.NewStyle().
			Foreground(colorCyan)

	// StyleLink for URLs
	StyleLink = lipgloss.NewStyle().
			Foreground(colorBlue).
			Underline(true)

	// StyleDim for secondary/muted text
	StyleDim = lipgloss.NewStyle().
			Foreground(colorDim)

	// StyleLabel for form labels
	StyleLabel = lipgloss.NewStyle().
			Foreground(colorGray)

	// StyleValue for data values
	StyleValue = lipgloss.NewStyle().
			Foreground(colorWhite)

	// StyleNumber for numeric values
	StyleNumber = lipgloss.NewStyle().
			Foreground(colorCyan)
)

// =============================================================================
// Status Styles
// =============================================================================

var (
	// StyleSuccess for success messages
	StyleSuccess = lipgloss.NewStyle().
			Foreground(colorGreen)

	// StyleWarning for warning messages
	StyleWarning = lipgloss.NewStyle().
			Foreground(colorYellow)

	// StyleError for error messages
	StyleError = lipgloss.NewStyle().
			Foreground(colorRed)
)

// =============================================================================
// Container Styles
// =============================================================================

var (
	// StyleBox for bordered content boxes
	StyleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorGray).
			Padding(0, 1)

	// StyleBoxSuccess for success boxes
	StyleBoxSuccess = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorGreen).
			Padding(0, 1)
)

// =============================================================================
// Internal Styles
// =============================================================================

var (
	// Icon styles
	styleIconSuccess = lipgloss.NewStyle().Foreground(colorGreen)
	styleIconError   = lipgloss.NewStyle().Foreground(colorRed)
	styleIconWarning = lipgloss.NewStyle().Foreground(colorYellow)
	styleIconInfo    = lipgloss.NewStyle().Foreground(colorGray)
	styleIconSpinner = lipgloss.NewStyle().Foreground(colorCyan)

	// Key-value styles
	styleKey = lipgloss.NewStyle().
			Foreground(colorGray)

	styleKeyLabel = lipgloss.NewStyle().
			Foreground(colorGray).
			Width(12)

	// Stats line style
	styleStat = lipgloss.NewStyle().
			Foreground(colorGray)

	styleStatValue = lipgloss.NewStyle().
			Foreground(colorCyan)

	// Command style (no background)
	styleCommand = lipgloss.NewStyle().
			Foreground(colorBlue)

	// Cache status (no background)
	styleCached = lipgloss.NewStyle().
			Foreground(colorGreen)

	styleComputed = lipgloss.NewStyle().
			Foreground(colorGray)
)
