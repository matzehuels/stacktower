// Package rust provides dependency resolution for Rust crates.
//
// # Overview
//
// This package implements [deps.Language] for Rust, supporting:
//
//   - crates.io registry resolution via [crates] client
//   - Cargo.toml manifest parsing
//
// # Registry Resolution
//
// Use [Language.Resolver] to fetch dependencies from crates.io:
//
//	resolver, _ := rust.Language.Resolver()
//	g, _ := resolver.Resolve(ctx, "serde", deps.Options{MaxDepth: 10})
//
// # Manifest Parsing
//
// Parse Cargo.toml files:
//
//	parser, _ := rust.Language.Manifest("cargo", nil)
//	result, _ := parser.Parse("Cargo.toml", deps.Options{})
//
// Note: Cargo.toml contains direct dependencies only. The resolver fetches
// transitive dependencies from crates.io.
//
// [crates]: github.com/matzehuels/stacktower/pkg/integrations/crates
// [deps.Language]: github.com/matzehuels/stacktower/pkg/deps.Language
package rust
