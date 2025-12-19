// Package ordering provides algorithms for determining the left-to-right
// arrangement of nodes within each row of a layered graph.
//
// # The Ordering Problem
//
// In tower visualizations, blocks must physically rest on what they depend on.
// This requires eliminating edge crossings—if edges cross, the corresponding
// blocks cannot support each other. Finding an ordering with zero crossings
// (or minimum crossings when zero is impossible) is NP-hard.
//
// This package provides multiple algorithms with different tradeoffs:
//
//   - [Barycentric]: Fast heuristic, O(n log n) per pass
//   - [OptimalSearch]: Exact algorithm with branch-and-bound pruning
//
// # Barycentric Heuristic
//
// The [Barycentric] orderer implements the classic Sugiyama barycenter method
// with weighted median refinement. It positions each node near the average
// position of its neighbors, then iteratively improves through alternating
// top-down and bottom-up sweeps.
//
// The algorithm:
//
//  1. Initialize the top row (alphabetically or by structure)
//  2. For each subsequent row, sort nodes by their parents' average positions
//  3. Apply transpose passes to swap adjacent nodes that reduce crossings
//  4. Alternate sweep direction (top-down, bottom-up) for several passes
//  5. Return the best ordering found
//
// This runs in milliseconds even for large graphs, but provides no optimality
// guarantee. It's used both standalone and as the initial bound for optimal
// search.
//
// # Optimal Search
//
// [OptimalSearch] uses branch-and-bound with PQ-tree pruning to find the
// true minimum-crossing ordering. The key innovations:
//
//   - PQ-tree constraints: Children of the same parent should be adjacent;
//     subdivider chains must stay together. This prunes invalid orderings.
//
//   - Barycentric initialization: The heuristic provides a tight initial
//     bound, enabling aggressive pruning from the start.
//
//   - Incremental crossing count: Crossings are computed layer-by-layer
//     using Fenwick trees, with early termination when the bound is exceeded.
//
//   - Parallel search: Multiple starting permutations are explored
//     concurrently using goroutines.
//
// For most real-world dependency graphs (dozens to low hundreds of nodes),
// optimal search finds the true minimum in seconds. A configurable timeout
// ensures graceful fallback to the best-found solution.
//
// # Usage
//
// The [Orderer] interface allows algorithms to be used interchangeably:
//
//	var orderer ordering.Orderer = ordering.Barycentric{Passes: 24}
//	orders := orderer.OrderRows(g)  // map[row][]nodeID
//
// For optimal search with progress reporting:
//
//	orderer := ordering.OptimalSearch{
//	    Timeout: 30 * time.Second,
//	    Progress: func(explored, pruned, best int) {
//	        fmt.Printf("Explored %d, pruned %d, best=%d\n", explored, pruned, best)
//	    },
//	}
//	orders := orderer.OrderRows(g)
//
// # Quality Presets
//
// The [Quality] type provides preset configurations:
//
//   - [QualityFast]: 100ms timeout, suitable for interactive use
//   - [QualityBalanced]: 5s timeout, good for most graphs
//   - [QualityOptimal]: 60s timeout, for publication-quality output
//
// # Algorithm Selection
//
// Use barycentric for:
//   - Large graphs (hundreds of nodes)
//   - Interactive/preview rendering
//   - When "good enough" suffices
//
// Use optimal search for:
//   - Publication or showcase output
//   - Smaller graphs where crossing-free is achievable
//   - When visual quality is critical
package ordering
