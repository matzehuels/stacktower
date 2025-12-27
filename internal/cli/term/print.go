package term

import "fmt"

// =============================================================================
// Icons
// =============================================================================

const (
	IconSuccess = "✓"
	IconError   = "✗"
	IconWarning = "!"
	IconInfo    = "›"
	IconArrow   = "→"
	IconDot     = "•"
	IconFile    = "→"
	IconCached  = "cached"
	IconFresh   = "fresh"
)

// =============================================================================
// Status Output
// =============================================================================

// PrintSuccess prints a success message.
func PrintSuccess(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(styleIconSuccess.Render(IconSuccess) + " " + msg)
}

// PrintError prints an error message.
func PrintError(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(styleIconError.Render(IconError) + " " + msg)
}

// PrintWarning prints a warning message.
func PrintWarning(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(styleIconWarning.Render(IconWarning) + " " + StyleWarning.Render(msg))
}

// PrintInfo prints an info/status message.
func PrintInfo(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(styleIconInfo.Render(IconInfo) + " " + msg)
}

// PrintDetail prints a detail line (indented).
func PrintDetail(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println("  " + StyleDim.Render(msg))
}

// =============================================================================
// Key-Value Output
// =============================================================================

// PrintKeyValue prints a labeled value.
func PrintKeyValue(key, value string) {
	fmt.Println(styleKeyLabel.Render(key) + " " + StyleValue.Render(value))
}

// PrintInlineStats prints stats on a single line.
func PrintInlineStats(label string, pairs ...string) {
	line := styleStat.Render(label)
	for i := 0; i < len(pairs); i += 2 {
		if i > 0 {
			line += styleStat.Render(" · ")
		} else {
			line += " "
		}
		line += styleStat.Render(pairs[i]+": ") + styleStatValue.Render(pairs[i+1])
	}
	fmt.Println(line)
}

// =============================================================================
// File Output
// =============================================================================

// PrintFileSaved prints a file save confirmation.
func PrintFileSaved(path string) {
	fmt.Println(styleIconSuccess.Render(IconSuccess) + " " + StyleDim.Render("Saved") + " " + StyleHighlight.Render(path))
}

// PrintFileList prints a list of output files.
func PrintFileList(paths []string) {
	for _, path := range paths {
		fmt.Println("  " + StyleDim.Render(IconArrow) + " " + StyleValue.Render(path))
	}
}

// =============================================================================
// Commands & Next Steps
// =============================================================================

// PrintCommand prints a command suggestion.
func PrintCommand(cmd string) {
	fmt.Println("  " + styleCommand.Render(cmd))
}

// PrintNextStep prints a suggested next command.
func PrintNextStep(description, cmd string) {
	fmt.Println(StyleDim.Render(description+":") + " " + styleCommand.Render(cmd))
}

// =============================================================================
// Utilities
// =============================================================================

// PrintInline prints a dim message without a trailing newline.
// Use for status messages that will be followed by more output on the same line.
func PrintInline(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Print(StyleDim.Render(msg))
}

// PrintNewline prints an empty line.
func PrintNewline() {
	fmt.Println()
}

// PrintDivider prints a subtle divider.
func PrintDivider() {
	fmt.Println(StyleDim.Render("─────────────────────────────────────"))
}
