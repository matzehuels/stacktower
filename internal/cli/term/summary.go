package term

import (
	"fmt"
	"strings"
)

// =============================================================================
// Summary Output
// =============================================================================

// Summary represents a completion summary.
type Summary struct {
	Title  string
	Lines  []string
	Status SummaryStatus
}

// SummaryStatus indicates the summary type.
type SummaryStatus int

const (
	SummarySuccess SummaryStatus = iota
	SummaryError
	SummaryInfo
)

// NewSuccessSummary creates a success summary.
func NewSuccessSummary(title string) *Summary {
	return &Summary{Title: title, Status: SummarySuccess}
}

// NewErrorSummary creates an error summary.
func NewErrorSummary(title string) *Summary {
	return &Summary{Title: title, Status: SummaryError}
}

// AddLine adds a content line.
func (s *Summary) AddLine(format string, args ...any) *Summary {
	s.Lines = append(s.Lines, fmt.Sprintf(format, args...))
	return s
}

// AddKeyValue adds a labeled value line.
func (s *Summary) AddKeyValue(key, value string) *Summary {
	s.Lines = append(s.Lines, styleKey.Render(key+": ")+StyleValue.Render(value))
	return s
}

// AddFile adds a file output line.
func (s *Summary) AddFile(path string) *Summary {
	s.Lines = append(s.Lines, StyleDim.Render(IconArrow)+" "+StyleValue.Render(path))
	return s
}

// Print renders the summary.
func (s *Summary) Print() {
	var b strings.Builder

	// Title line
	icon := IconSuccess
	iconStyle := styleIconSuccess
	if s.Status == SummaryError {
		icon = IconError
		iconStyle = styleIconError
	}

	b.WriteString(iconStyle.Render(icon) + " " + StyleHeading.Render(s.Title))

	// Content lines
	for _, line := range s.Lines {
		b.WriteString("\n  " + line)
	}

	fmt.Println(b.String())
}
