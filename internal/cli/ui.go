package cli

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Color Palette
// =============================================================================

var (
	colorCyan   = lipgloss.Color("36")  // Teal - primary actions
	colorGreen  = lipgloss.Color("35")  // Green - success
	colorYellow = lipgloss.Color("220") // Amber - warnings
	colorRed    = lipgloss.Color("167") // Soft red - errors
	colorBlue   = lipgloss.Color("75")  // Light blue - links
	colorWhite  = lipgloss.Color("255") // Bright white - values
	colorGray   = lipgloss.Color("245") // Gray - secondary text
	colorDim    = lipgloss.Color("240") // Dim gray - muted text
)

// =============================================================================
// Public Styles
// =============================================================================

var (
	// StyleTitle for main headings.
	StyleTitle = lipgloss.NewStyle().Bold(true).Foreground(colorCyan)

	// StyleHighlight for emphasized values.
	StyleHighlight = lipgloss.NewStyle().Foreground(colorCyan)

	// StyleLink for URLs.
	StyleLink = lipgloss.NewStyle().Foreground(colorBlue).Underline(true)

	// StyleDim for secondary/muted text.
	StyleDim = lipgloss.NewStyle().Foreground(colorDim)

	// StyleValue for data values.
	StyleValue = lipgloss.NewStyle().Foreground(colorWhite)

	// StyleNumber for numeric values.
	StyleNumber = lipgloss.NewStyle().Foreground(colorCyan)

	// StyleSuccess for success messages.
	StyleSuccess = lipgloss.NewStyle().Foreground(colorGreen)

	// StyleWarning for warning messages.
	StyleWarning = lipgloss.NewStyle().Foreground(colorYellow)
)

// =============================================================================
// Internal Styles
// =============================================================================

var (
	styleIconSuccess = lipgloss.NewStyle().Foreground(colorGreen)
	styleIconError   = lipgloss.NewStyle().Foreground(colorRed)
	styleIconWarning = lipgloss.NewStyle().Foreground(colorYellow)
	styleIconInfo    = lipgloss.NewStyle().Foreground(colorGray)
	styleIconSpinner = lipgloss.NewStyle().Foreground(colorCyan)

	styleCached   = lipgloss.NewStyle().Foreground(colorGreen)
	styleComputed = lipgloss.NewStyle().Foreground(colorGray)

	styleCommand = lipgloss.NewStyle().Foreground(colorBlue)
)

// =============================================================================
// Icons
// =============================================================================

const (
	iconSuccess = "✓"
	iconError   = "✗"
	iconWarning = "!"
	iconInfo    = "›"
	iconArrow   = "→"
	iconCached  = "cached"
	iconFresh   = "fresh"
)

// =============================================================================
// Status Output
// =============================================================================

// printSuccess prints a success message.
func printSuccess(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(styleIconSuccess.Render(iconSuccess) + " " + msg)
}

// printError prints an error message.
func printError(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(styleIconError.Render(iconError) + " " + msg)
}

// printWarning prints a warning message.
func printWarning(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(styleIconWarning.Render(iconWarning) + " " + StyleWarning.Render(msg))
}

// printInfo prints an info/status message.
func printInfo(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(styleIconInfo.Render(iconInfo) + " " + msg)
}

// printDetail prints a detail line (indented).
func printDetail(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println("  " + StyleDim.Render(msg))
}

// =============================================================================
// File Output
// =============================================================================

// printFile prints a file output line.
func printFile(path string) {
	fmt.Println("  " + StyleDim.Render(iconArrow) + " " + StyleValue.Render(path))
}

// =============================================================================
// Key-Value Output
// =============================================================================

// printKeyValue prints a labeled value.
func printKeyValue(key, value string) {
	keyStyle := lipgloss.NewStyle().Foreground(colorGray).Width(12)
	fmt.Println(keyStyle.Render(key) + " " + StyleValue.Render(value))
}

// =============================================================================
// Stats Display
// =============================================================================

// printStats prints graph statistics on a single line.
func printStats(nodeCount, edgeCount int, cached bool) {
	var parts []string
	if nodeCount > 0 {
		parts = append(parts, fmt.Sprintf("%d nodes", nodeCount))
	}
	if edgeCount > 0 {
		parts = append(parts, fmt.Sprintf("%d edges", edgeCount))
	}

	status := iconFresh
	statusStyle := styleComputed
	if cached {
		status = iconCached
		statusStyle = styleCached
	}
	parts = append(parts, statusStyle.Render(status))

	line := "  "
	for i, part := range parts {
		if i > 0 {
			line += StyleDim.Render(" · ")
		}
		line += StyleDim.Render(part)
	}
	fmt.Println(line)
}

// =============================================================================
// Commands & Next Steps
// =============================================================================

// printNextStep prints a suggested next command.
func printNextStep(description, cmd string) {
	fmt.Println(StyleDim.Render(description+":") + " " + styleCommand.Render(cmd))
}

// =============================================================================
// Utilities
// =============================================================================

// printInline prints a dim message without a trailing newline.
func printInline(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Print(StyleDim.Render(msg))
}

// printNewline prints an empty line.
func printNewline() {
	fmt.Println()
}
