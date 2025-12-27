package term

import "fmt"

// =============================================================================
// Cache Status
// =============================================================================

// FormatCacheStatus returns a styled cache status string.
func FormatCacheStatus(cached bool) string {
	if cached {
		return styleCached.Render(IconCached)
	}
	return styleComputed.Render(IconFresh)
}

// =============================================================================
// Stats Display
// =============================================================================

// PrintStats prints graph statistics on a single line.
func PrintStats(nodeCount, edgeCount int, cached bool) {
	var parts []string
	if nodeCount > 0 {
		parts = append(parts, fmt.Sprintf("%d nodes", nodeCount))
	}
	if edgeCount > 0 {
		parts = append(parts, fmt.Sprintf("%d edges", edgeCount))
	}
	parts = append(parts, FormatCacheStatus(cached))

	line := "  "
	for i, part := range parts {
		if i > 0 {
			line += StyleDim.Render(" · ")
		}
		line += StyleDim.Render(part)
	}
	fmt.Println(line)
}
