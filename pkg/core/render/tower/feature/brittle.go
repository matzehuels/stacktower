package feature

import (
	"time"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/deps/metadata"
)

const (
	abandonedThreshold = 2 * 365 * 24 * time.Hour
	staleThreshold     = 1 * 365 * 24 * time.Hour
	lowStarCount       = 100
	minMaintainerCount = 2
)

// IsBrittle returns true if a node represents a package that is potentially
// unmaintained or risky to depend on. It checks for archived repositories,
// long periods of inactivity, and low maintainer counts.
func IsBrittle(n *dag.Node) bool {
	if n == nil || n.Meta == nil {
		return false
	}
	if archived, _ := n.Meta[metadata.RepoArchived].(bool); archived {
		return true
	}

	maintainers := CountMaintainers(n.Meta[metadata.RepoMaintainers])
	stars, _ := n.Meta[metadata.RepoStars].(int)

	// Check last commit date for staleness/abandonment
	lastCommit := ParseDate(n.Meta[metadata.RepoLastCommit])
	if !lastCommit.IsZero() {
		age := time.Since(lastCommit)
		if age > abandonedThreshold {
			return true
		}
		// If recently updated (< 1 year), not brittle regardless of maintainers/stars
		if age <= staleThreshold {
			return false
		}
		// Stale (1-2 years): brittle if low maintainers or low stars
		return maintainers == 1 || stars < lowStarCount || maintainers <= minMaintainerCount
	}

	// No last commit date available - fall back to maintainers/stars check
	// This ensures consistency with the graph serialization logic
	hasFewMaintainers := maintainers > 0 && maintainers <= minMaintainerCount
	hasLowStars := stars > 0 && stars < lowStarCount
	return hasFewMaintainers || hasLowStars
}

func ParseDate(v any) time.Time {
	s, ok := v.(string)
	if !ok || s == "" {
		return time.Time{}
	}
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func CountMaintainers(v any) int {
	switch v := v.(type) {
	case []string:
		return len(v)
	case []any:
		return len(v)
	}
	return 0
}

func AsInt(v any) int {
	switch v := v.(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}
