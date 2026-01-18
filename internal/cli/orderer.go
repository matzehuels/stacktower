package cli

import (
	"context"
	"sync"
	"time"

	"github.com/matzehuels/stacktower/internal/cli/term"
	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/ordering"
)

// optimalOrderer wraps ordering.OptimalSearch with progress feedback.
// It shows improvements via UI and keeps debug info for --verbose mode.
type optimalOrderer struct {
	ordering.OptimalSearch
	logger *Logger // Only used for --verbose debug output

	mu                       sync.Mutex
	lastExplored, lastPruned int
	lastBest                 int
	start, lastLog           time.Time
	improved                 bool
}

// newOptimalOrderer creates an optimal orderer with a timeout.
// The timeoutSec parameter controls how long the search runs before returning the best solution found.
// Longer timeouts may find better orderings (fewer edge crossings) at the cost of increased runtime.
func newOptimalOrderer(ctx context.Context, timeoutSec int) ordering.Orderer {
	logger := loggerFromContext(ctx)
	o := &optimalOrderer{
		logger:   logger,
		lastBest: -1,
		start:    time.Now(),
	}

	o.OptimalSearch = ordering.OptimalSearch{
		Timeout:  time.Duration(timeoutSec) * time.Second,
		Progress: o.onProgress,
		Debug:    o.onDebug,
	}
	return o
}

// onProgress is called by the underlying OptimalSearch during the search.
// It tracks improvements silently (the spinner is already showing progress).
func (o *optimalOrderer) onProgress(explored, pruned, bestScore int) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.lastExplored, o.lastPruned = explored, pruned
	if bestScore < 0 || (explored == 0 && pruned == 0) {
		return
	}

	// Track if we improved (for final summary)
	if o.lastBest >= 0 && bestScore < o.lastBest {
		o.improved = true
	}
	o.lastBest = bestScore

	// Debug logging only (visible with --verbose)
	switch {
	case o.lastBest < 0:
		o.logger.Debugf("Initial: %d crossings (explored: %d, pruned: %d)", bestScore, explored, pruned)
		o.lastLog = time.Now()
	case bestScore < o.lastBest:
		o.logger.Debugf("Improved: %d crossings (↓%d)", bestScore, o.lastBest-bestScore)
		o.lastLog = time.Now()
	default:
		if time.Since(o.lastLog) >= 10*time.Second {
			elapsed := time.Since(o.start).Truncate(time.Second)
			timeout := o.Timeout.Seconds()
			o.logger.Debugf("Searching... %v/%.0fs elapsed, %d crossings (pruned: %d)", elapsed, timeout, bestScore, pruned)
			o.lastLog = time.Now()
		}
	}
}

// onDebug is called when the search completes to report diagnostic information.
// This is only visible with --verbose.
func (o *optimalOrderer) onDebug(info ordering.DebugInfo) {
	o.logger.Debugf("Search space: %d rows, max depth reached: %d/%d", info.TotalRows, info.MaxDepth, info.TotalRows)

	bottlenecks := 0
	for _, r := range info.Rows {
		if r.Candidates > 100 {
			o.logger.Debugf("  Row %d: %d nodes, %d candidates", r.Row, r.NodeCount, r.Candidates)
			bottlenecks++
		}
	}

	if info.MaxDepth < info.TotalRows && bottlenecks > 0 {
		o.logger.Debugf("Search incomplete: %d rows have >100 candidates, causing combinatorial explosion", bottlenecks)
	}
}

// OrderRows implements ordering.Orderer by delegating to OptimalSearch.
// It shows a warning via UI if crossings remain.
func (o *optimalOrderer) OrderRows(g *dag.DAG) map[int][]string {
	result := o.OptimalSearch.OrderRows(g)
	crossings := dag.CountCrossings(g, result)

	o.mu.Lock()
	explored, pruned := o.lastExplored, o.lastPruned
	o.mu.Unlock()

	// Debug output (--verbose only)
	o.logger.Debugf("Best: %d crossings (explored: %d, pruned: %d)", crossings, explored, pruned)

	// User-visible warning if there are crossings
	if crossings > 0 {
		term.PrintWarning("Layout has %d edge crossings (try --ordering-timeout to increase search time)", crossings)
	}

	return result
}
