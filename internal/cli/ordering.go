package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/log"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/render/tower/ordering"
)

// optimalOrderer wraps ordering.OptimalSearch with progress logging and debug output.
// It logs initial solutions, improvements, periodic status updates (every 10 seconds),
// and warnings when the search encounters combinatorial bottlenecks.
//
// The orderer is not safe for concurrent use; it maintains internal state for logging.
type optimalOrderer struct {
	ordering.OptimalSearch
	prog                     *progress
	logger                   *log.Logger
	lastExplored, lastPruned int
	lastBest                 int
	start, lastLog           time.Time
}

// newOptimalOrderer creates an optimal orderer with a timeout and logger from ctx.
// The timeoutSec parameter controls how long the search runs before returning the best solution found.
// Longer timeouts may find better orderings (fewer edge crossings) at the cost of increased runtime.
//
// The orderer logs progress updates including initial solutions, improvements, and periodic heartbeats.
func newOptimalOrderer(ctx context.Context, timeoutSec int) ordering.Orderer {
	logger := loggerFromContext(ctx)
	o := &optimalOrderer{
		prog:     newProgress(logger),
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
// It logs the initial solution, improvements when bestScore decreases, and periodic heartbeats
// every 10 seconds to show the search is still running.
//
// Parameters:
//   - explored: number of partial solutions examined
//   - pruned: number of branches eliminated via bounds
//   - bestScore: current best edge crossing count (lower is better)
func (o *optimalOrderer) onProgress(explored, pruned, bestScore int) {
	o.lastExplored, o.lastPruned = explored, pruned
	if bestScore < 0 || (explored == 0 && pruned == 0) {
		return
	}

	switch {
	case o.lastBest < 0:
		o.logger.Infof("Initial: %d crossings (explored: %d, pruned: %d)", bestScore, explored, pruned)
		o.lastLog = time.Now()
	case bestScore < o.lastBest:
		o.logger.Infof("Improved: %d crossings (↓%d)", bestScore, o.lastBest-bestScore)
		o.lastLog = time.Now()
	default:
		if time.Since(o.lastLog) >= 10*time.Second {
			elapsed := time.Since(o.start).Truncate(time.Second)
			timeout := o.Timeout.Seconds()
			o.logger.Infof("Searching... %v/%.0fs elapsed, %d crossings (pruned: %d)", elapsed, timeout, bestScore, pruned)
			o.lastLog = time.Now()
		}
	}
	o.lastBest = bestScore
}

// onDebug is called when the search completes to report diagnostic information.
// It logs the search space size, maximum depth reached, and identifies bottleneck rows
// with >100 candidate orderings that may have caused incomplete search.
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

// OrderRows implements ordering.Orderer by delegating to OptimalSearch and logging the final result.
// It counts edge crossings in the returned ordering and warns if crossings remain.
//
// If crossings > 0, users are advised to increase --ordering-timeout for better results.
func (o *optimalOrderer) OrderRows(g *dag.DAG) map[int][]string {
	result := o.OptimalSearch.OrderRows(g)
	crossings := dag.CountCrossings(g, result)
	o.prog.done(fmt.Sprintf("Layout complete: %d crossings", crossings))

	if crossings >= 0 {
		o.logger.Infof("Best: %d crossings (explored: %d, pruned: %d)", crossings, o.lastExplored, o.lastPruned)
	}
	if crossings > 0 {
		o.logger.Warn("Layout has edge crossings; try increasing the timeout (--ordering-timeout)")
	}
	return result
}
