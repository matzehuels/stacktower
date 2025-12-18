package feature

import (
	"time"

	"github.com/matzehuels/stacktower/pkg/dag"
)

const (
	abandonedThreshold = 2 * 365 * 24 * time.Hour
	staleThreshold     = 1 * 365 * 24 * time.Hour
	lowStarCount       = 100
	minMaintainerCount = 2
)

func IsBrittle(n *dag.Node) bool {
	if n == nil || n.Meta == nil {
		return false
	}
	if archived, _ := n.Meta["repo_archived"].(bool); archived {
		return true
	}

	lastCommit := ParseDate(n.Meta["repo_last_commit"])
	if lastCommit.IsZero() {
		return false
	}

	age := time.Since(lastCommit)
	if age > abandonedThreshold {
		return true
	}
	if age <= staleThreshold {
		return false
	}

	maintainers := CountMaintainers(n.Meta["repo_maintainers"])
	stars, _ := n.Meta["repo_stars"].(int)
	return maintainers == 1 || stars < lowStarCount || maintainers <= minMaintainerCount
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
