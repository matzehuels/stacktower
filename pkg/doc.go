// Package pkg provides the core libraries for Stacktower dependency visualization.
//
// # Overview
//
// Stacktower transforms dependency trees into visual tower diagrams where packages
// rest on what they depend on—inspired by XKCD #2347 ("Dependency"). The pkg
// directory is organized into four main areas:
//
//  1. [core] - Domain logic (dependency resolution, graph structures, rendering)
//  2. [infra] - Infrastructure (caching, storage, queues, sessions)
//  3. [integrations] - External API clients (PyPI, npm, GitHub, etc.)
//  4. [pipeline] - Orchestration (parse → layout → render)
//  5. [io] - Data import/export (JSON node-link format)
//
// # Architecture
//
// The typical data flow through Stacktower:
//
//	Package Registry/Manifest
//	         ↓
//	    [core/deps] package (resolve dependencies)
//	         ↓
//	    [core/dag] package (graph structure + transformations)
//	         ↓
//	    [core/render] package (layout + visualization)
//	         ↓
//	    SVG/PDF/PNG/JSON output
//
// # Quick Start
//
// Resolve dependencies and render a tower visualization:
//
//	import (
//	    "context"
//	    "github.com/matzehuels/stacktower/pkg/core/deps/python"
//	    "github.com/matzehuels/stacktower/pkg/core/dag/transform"
//	    "github.com/matzehuels/stacktower/pkg/core/render/tower/layout"
//	    "github.com/matzehuels/stacktower/pkg/core/render/tower/sink"
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
// ## Core Domain Logic
//
// [core/deps] - Dependency resolution supporting 7 languages (Python, Rust,
// JavaScript, Go, Ruby, PHP, Java). Each language has its own subpackage with
// manifest parsers and registry resolvers.
//
// [core/dag] - Directed acyclic graph optimized for row-based layered layouts.
// Nodes are organized into horizontal rows with edges connecting consecutive
// rows only. Supports regular, subdivider, and auxiliary node types.
//
// [core/render] - Visualization rendering with two output formats: tower
// (stacked blocks) and nodelink (traditional directed graphs).
//
// ## Infrastructure
//
// [infra/cache] - Two-tier cache (Redis + MongoDB) for API server. Stores
// user renders, graph data, and HTTP response caching for registry APIs.
//
// [infra/artifact] - Unified caching backend for both pipeline artifacts
// (graphs, layouts, renders) AND HTTP response caching (PyPI, npm, etc.).
// LocalBackend for CLI, ProdBackend for API/worker.
//
// ## External Integrations
//
// [integrations] - HTTP clients for package registries (PyPI, npm, crates.io,
// RubyGems, Packagist, Maven, Go proxy) and code hosts (GitHub, GitLab).
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
// ## Infrastructure
//
// [pipeline] - Complete visualization pipeline (parse → layout → render) used
// by CLI, API, and worker. Ensures consistent behavior across all entry points.
//
// [artifact] - Unified artifact caching and storage service for the CLI.
// Implements a two-tier caching strategy:
//
//   - Tier 1 (Cache Index): Fast TTL-based lookup mapping hash(inputs) → storage_key
//   - Tier 2 (Storage): Durable artifact storage (storage_key → artifact_bytes)
//
// On cache hit, retrieves artifacts from storage using the cached key. When TTL
// expires or --refresh is set, recomputes and upserts the artifact.
//
// [cache] - Two-tier caching with Redis (fast lookup) and MongoDB (durable storage).
// Defines interfaces for LookupCache and Store backends.
//
// [queue] - Job queue interface with memory and Redis implementations. Supports
// job lifecycle (pending, running, completed, failed, cancelled).
//
// [storage] - Artifact storage interface with memory and GridFS implementations.
// Used to store rendered outputs (SVG, PNG, PDF).
//
// [session] - Session management for authenticated users. Provides memory, Redis,
// and file-based backends for sessions and OAuth state tokens.
//
// [config] - Shared configuration and constants (e.g., TTLs).
//
// [errors] - Shared sentinel errors used across packages.
//
// [hash] - Shared hashing utilities (SHA-256).
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
//	go test ./pkg/core/dag/...           # Specific package
//	go test -run Example                 # Examples only
//	go test -tags integration ./pkg/...  # Include integration tests
//
// [core]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/core
// [core/deps]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/core/deps
// [core/dag]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/core/dag
// [core/render]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/core/render
// [infra]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/infra
// [infra/cache]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/infra/cache
// [infra/artifact]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/infra/artifact
// [integrations]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/integrations
// [dag/transform]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/core/dag/transform
// [dag/perm]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/core/dag/perm
// [render]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/core/render
// [render/tower]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/core/render/tower
// [render/tower/ordering]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/core/render/tower/ordering
// [render/tower/layout]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/core/render/tower/layout
// [render/tower/transform]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/core/render/tower/transform
// [render/tower/sink]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/core/render/tower/sink
// [render/tower/styles]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/core/render/tower/styles
// [render/tower/feature]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/core/render/tower/feature
// [render/nodelink]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/core/render/nodelink
// [io]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/io
// [pipeline]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/pipeline
// [artifact]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/infra/artifact
// [cache]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/infra/cache
// [queue]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/infra/queue
// [session]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/infra/session
//
// [deps]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/core/deps
// [dag]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/core/dag
// [render/tower/styles/handdrawn]: https://pkg.go.dev/github.com/matzehuels/stacktower/pkg/core/render/tower/styles/handdrawn
package pkg
