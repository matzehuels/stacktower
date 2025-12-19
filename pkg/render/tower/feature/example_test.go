package feature_test

import (
	"fmt"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/render/tower/feature"
)

func ExampleRankNebraska() {
	// Create a dependency graph with GitHub metadata
	g := dag.New(nil)

	// Root application (row 0)
	_ = g.AddNode(dag.Node{ID: "myapp", Row: 0})

	// Mid-level dependencies (row 1)
	_ = g.AddNode(dag.Node{
		ID:  "framework",
		Row: 1,
		Meta: dag.Metadata{
			"repo_owner":       "alice",
			"repo_maintainers": []string{"alice", "bob"},
			"repo_url":         "https://github.com/alice/framework",
		},
	})

	// Foundation packages (row 2) - deeper = more critical
	_ = g.AddNode(dag.Node{
		ID:  "http-client",
		Row: 2,
		Meta: dag.Metadata{
			"repo_owner":       "charlie",
			"repo_maintainers": []string{"charlie"},
			"repo_url":         "https://github.com/charlie/http-client",
		},
	})
	_ = g.AddNode(dag.Node{
		ID:  "json-parser",
		Row: 2,
		Meta: dag.Metadata{
			"repo_owner":       "alice",
			"repo_maintainers": []string{"alice", "dave"},
			"repo_url":         "https://github.com/alice/json-parser",
		},
	})

	// Add edges
	_ = g.AddEdge(dag.Edge{From: "myapp", To: "framework"})
	_ = g.AddEdge(dag.Edge{From: "framework", To: "http-client"})
	_ = g.AddEdge(dag.Edge{From: "framework", To: "json-parser"})

	// Rank the top 3 maintainers
	rankings := feature.RankNebraska(g, 3)

	for _, r := range rankings {
		fmt.Printf("%s (score: %.1f)\n", r.Maintainer, r.Score)
		for _, pkg := range r.Packages {
			fmt.Printf("  - %s (%s, depth %d)\n", pkg.Package, pkg.Role, pkg.Depth)
		}
	}
	// Output:
	// charlie (score: 6.0)
	//   - http-client (owner, depth 2)
	// alice (score: 4.5)
	//   - json-parser (owner, depth 2)
	//   - framework (owner, depth 1)
	// dave (score: 1.5)
	//   - json-parser (lead, depth 2)
}

func ExampleRankNebraska_sharedResponsibility() {
	// Demonstrate how scores are divided among maintainers
	g := dag.New(nil)

	_ = g.AddNode(dag.Node{ID: "app", Row: 0})
	_ = g.AddNode(dag.Node{
		ID:  "critical-lib",
		Row: 1,
		Meta: dag.Metadata{
			"repo_owner": "alice",
			// Three maintainers share responsibility
			"repo_maintainers": []string{"alice", "bob", "charlie"},
		},
	})
	_ = g.AddEdge(dag.Edge{From: "app", To: "critical-lib"})

	rankings := feature.RankNebraska(g, 3)

	fmt.Println("Shared responsibility divides the score:")
	for _, r := range rankings {
		fmt.Printf("%s: %.2f\n", r.Maintainer, r.Score)
	}
	// Output:
	// Shared responsibility divides the score:
	// alice: 1.00
	// bob: 0.50
	// charlie: 0.33
}

func ExampleIsBrittle() {
	// Check if a package is potentially unmaintained
	node := &dag.Node{
		ID:  "old-package",
		Row: 1,
		Meta: dag.Metadata{
			"repo_archived":    true,
			"repo_last_commit": "2020-01-01",
		},
	}

	if feature.IsBrittle(node) {
		fmt.Println("Package is brittle (archived or unmaintained)")
	}
	// Output:
	// Package is brittle (archived or unmaintained)
}

func ExampleIsBrittle_staleWithFewMaintainers() {
	// Package with no recent activity and single maintainer
	node := &dag.Node{
		ID:  "stale-package",
		Row: 1,
		Meta: dag.Metadata{
			"repo_archived":    false,
			"repo_last_commit": "2022-01-01", // Over 1 year old
			"repo_maintainers": []string{"solo-dev"},
			"repo_stars":       50, // Low star count
		},
	}

	if feature.IsBrittle(node) {
		fmt.Println("Package may be at risk")
	}
	// Output:
	// Package may be at risk
}

func ExampleIsBrittle_healthy() {
	// Well-maintained package with recent activity
	node := &dag.Node{
		ID:  "active-package",
		Row: 1,
		Meta: dag.Metadata{
			"repo_archived":    false,
			"repo_last_commit": "2024-12-01", // Recent
			"repo_maintainers": []string{"dev1", "dev2", "dev3"},
			"repo_stars":       1000,
		},
	}

	if !feature.IsBrittle(node) {
		fmt.Println("Package appears healthy")
	}
	// Output:
	// Package appears healthy
}
