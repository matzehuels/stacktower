// Package feature provides analysis features for tower visualizations.
//
// # Overview
//
// Beyond basic rendering, Stacktower can analyze dependency graphs to surface
// insights about the software ecosystem. This package provides:
//
//   - Nebraska ranking: Identify critical maintainers (inspired by XKCD #2347)
//   - Brittle detection: Flag potentially unmaintained dependencies
//
// # Nebraska Ranking
//
// The [RankNebraska] function identifies the maintainers whose packages are
// most critical to a project's foundation—the "Nebraska guys" from the famous
// XKCD comic about that one random person maintaining infrastructure everyone
// depends on.
//
// The scoring algorithm considers:
//
//   - Depth: Packages deeper in the tower (more dependencies above) score higher.
//     These foundational packages have larger "blast radius" if abandoned.
//
//   - Role weight: Owners get 3x weight, leads get 1.5x, regular maintainers
//     get 1x. This reflects the difference in bus factor impact.
//
//   - Shared credit: When multiple maintainers share a package, the depth
//     score is divided among them, reflecting distributed responsibility.
//
// Usage:
//
//	rankings := feature.RankNebraska(g, 10)  // Top 10 maintainers
//	for _, r := range rankings {
//	    fmt.Printf("%s (score: %.1f)\n", r.Maintainer, r.Score)
//	    for _, pkg := range r.Packages {
//	        fmt.Printf("  - %s (%s, depth %d)\n", pkg.Package, pkg.Role, pkg.Depth)
//	    }
//	}
//
// # Roles
//
// The package recognizes three maintainer roles:
//
//   - [RoleOwner]: The repository owner (highest weight)
//   - [RoleLead]: First listed maintainer who isn't owner (medium weight)
//   - [RoleMaintainer]: All other maintainers (standard weight)
//
// Role information comes from node metadata (repo_owner, repo_maintainers)
// populated by the GitHub enrichment during dependency parsing.
//
// # Brittle Detection
//
// The [IsBrittle] function identifies packages that may be at risk:
//
//   - Archived repositories
//   - No commits in over 2 years
//   - Single maintainer with no recent activity
//
// Brittle packages are highlighted in visualizations to draw attention to
// potential maintenance risks in the dependency tree.
//
// # Visualization Integration
//
// These features integrate with the rendering pipeline:
//
//	rankings := feature.RankNebraska(g, 5)
//	svg := sink.RenderSVG(layout,
//	    sink.WithNebraska(rankings),  // Adds ranking panel
//	    sink.WithPopups(),            // Shows brittle warnings in popups
//	)
//
// The SVG renderer adds interactive highlighting—hovering over a maintainer
// in the Nebraska panel highlights all their packages in the tower.
package feature
