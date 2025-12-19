// Package pkg provides the core libraries for Stacktower dependency visualization.
//
// # Overview
//
// Stacktower transforms dependency trees into visual tower diagrams where packages
// rest on what they depend on—inspired by XKCD #2347 ("Dependency"). The pkg
// directory contains reusable Go libraries organized into four main areas:
//
//  1. Dependency Resolution ([deps], [integrations], [httputil])
//  2. Graph Data Structures ([dag])
//  3. Visualization Rendering ([render])
//  4. Data Import/Export ([io])
//
// # Architecture
//
// The typical data flow through Stacktower:
//
//	Package Registry/Manifest
//	         ↓
//	    [deps] package (resolve dependencies)
//	         ↓
//	    [dag] package (graph structure + transformations)
//	         ↓
//	    [render] package (layout + visualization)
//	         ↓
//	    SVG/PDF/PNG/JSON output
//
// # Quick Start
//
// Resolve dependencies and render a tower visualization:
//
//	import (
//	    "context"
//	    "github.com/matzehuels/stacktower/pkg/deps/python"
//	    "github.com/matzehuels/stacktower/pkg/dag/transform"
//	    "github.com/matzehuels/stacktower/pkg/render/tower/layout"
//	    "github.com/matzehuels/stacktower/pkg/render/tower/sink"
//	)
//
//	// 1. Resolve dependencies
//	resolver, _ := python.Language.Resolver()
//	g, _ := resolver.Resolve(context.Background(), "fastapi", deps.Options{
//	    MaxDepth: 10,
//	    MaxNodes: 1000,
//	})
//
//	// 2. Transform the graph
//	g = transform.Normalize(g, nil)
//
//	// 3. Compute layout
//	l := layout.Build(g, 1200, 800)
//
//	// 4. Render to SVG
//	svg := sink.RenderSVG(l, g)
//
// # Main Packages
//
// ## Dependency Resolution
//
// [deps] - Core abstractions for dependency resolution. Supports 7 languages
// (Python, Rust, JavaScript, Go, Ruby, PHP, Java) via language-specific
// subpackages. See [deps/python], [deps/rust], etc. for details.
//
// [integrations] - Low-level HTTP clients for package registries (PyPI, npm,
// crates.io, RubyGems, Packagist, Maven, Go proxy). Each subpackage implements
// registry-specific API calls with caching and retry logic.
//
// [httputil] - Shared HTTP utilities for caching and retry logic.
//
// [deps/metadata] - Repository metadata enrichment from GitHub/GitLab (stars,
// maintainers, activity) used for Nebraska ranking and brittle detection.
//
// ## Graph Data Structures
//
// [dag] - Directed acyclic graph optimized for row-based layered layouts.
// Nodes are organized into horizontal rows with edges connecting consecutive
// rows only. Supports regular, subdivider, and auxiliary node types.
//
// [dag/transform] - Graph transformations: transitive reduction, layering,
// edge subdivision, and span overlap resolution. [transform.Normalize] runs
// the complete pipeline.
//
// [dag/perm] - Permutation algorithms including PQ-trees for efficiently
// generating valid orderings with partial ordering constraints.
//
// ## Visualization
//
// [render/tower] - Stacktower's signature tower visualization. The rendering
// pipeline: ordering → layout → transform → sink.
//
//   - [render/tower/ordering]: Minimize edge crossings (barycentric, optimal)
//   - [render/tower/layout]: Compute block positions and dimensions
//   - [render/tower/transform]: Post-layout (merge subdividers, randomize widths)
//   - [render/tower/sink]: Output formats (SVG, PDF, PNG, JSON)
//   - [render/tower/styles]: Visual styles (simple, hand-drawn)
//   - [render/tower/feature]: Analysis (Nebraska ranking, brittle detection)
//
// [render/nodelink] - Traditional directed graph diagrams using Graphviz.
//
// [render] - Top-level utilities for format conversion (SVG to PDF/PNG).
//
// ## Data Import/Export
//
// [io] - Import/export dependency graphs in JSON node-link format.
//
// # Common Workflows
//
// Parse a manifest file:
//
//	parser := python.PoetryLock{}
//	result, _ := parser.Parse("poetry.lock", deps.Options{})
//	g := result.Graph.(*dag.DAG)
//
// Enrich with GitHub metadata:
//
//	provider, _ := metadata.NewGitHub(token, 24*time.Hour)
//	opts := deps.Options{MetadataProviders: []deps.MetadataProvider{provider}}
//	g, _ := resolver.Resolve(ctx, "fastapi", opts)
//
// Render with custom style:
//
//	l := layout.Build(g, 1200, 800)
//	style := handdrawn.New(42)
//	svg := sink.RenderSVG(l, g, sink.WithStyle(style))
//
// Analyze maintainer risk:
//
//	rankings := feature.RankNebraska(g, 10)
//	for _, n := range g.Nodes() {
//	    if feature.IsBrittle(n) {
//	        fmt.Printf("Warning: %s may be unmaintained\n", n.ID)
//	    }
//	}
//
// # Testing
//
// Run tests:
//
//	go test ./pkg/...                    # All tests
//	go test ./pkg/dag/...                # Specific package
//	go test -run Example                 # Examples only
//	go test -tags integration ./pkg/...  # Include integration tests
//
// [deps]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/deps
// [integrations]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/integrations
// [httputil]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/httputil
// [dag]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/dag
// [dag/transform]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/dag/transform
// [dag/perm]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/dag/perm
// [render]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/render
// [render/tower]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/render/tower
// [render/tower/ordering]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/render/tower/ordering
// [render/tower/layout]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/render/tower/layout
// [render/tower/transform]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/render/tower/transform
// [render/tower/sink]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/render/tower/sink
// [render/tower/styles]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/render/tower/styles
// [render/tower/feature]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/render/tower/feature
// [render/nodelink]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/render/nodelink
// [io]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/io
//
// [render/tower/styles/handdrawn]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/render/tower/styles/handdrawn
package pkg
