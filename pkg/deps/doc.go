// Package deps provides dependency resolution from package registries and
// manifest files.
//
// # Overview
//
// Stacktower can fetch dependency data from multiple sources:
//
//   - Package registries (PyPI, npm, crates.io, RubyGems, Packagist, Maven, Go Proxy)
//   - Manifest files (requirements.txt, package.json, Cargo.toml, etc.)
//
// This package provides the core abstractions and concurrent resolver that
// powers the `stacktower parse` command.
//
// # Architecture
//
// The dependency resolution system has three layers:
//
//  1. Integrations ([integrations]): Low-level HTTP clients for each registry API
//  2. Language definitions (this package): Registry/manifest mappings
//  3. CLI ([internal/cli]): User-facing commands and output
//
// # Resolving Dependencies
//
// Use [Language.Resolve] to fetch a complete dependency tree:
//
//	lang := python.Language
//	resolver, _ := lang.NewResolver(24 * time.Hour)
//	g, _ := resolver.Resolve(ctx, "fastapi", deps.Options{
//	    MaxDepth: 10,
//	    MaxNodes: 1000,
//	})
//
// The resolver:
//
//  1. Fetches the root package from the registry
//  2. Recursively fetches dependencies (with configurable depth/node limits)
//  3. Builds a [dag.DAG] with package metadata
//  4. Optionally enriches with GitHub metadata (stars, maintainers)
//
// # Options
//
// [Options] controls resolution behavior:
//
//   - MaxDepth: Maximum dependency depth (default 50)
//   - MaxNodes: Maximum packages to fetch (default 5000)
//   - CacheTTL: How long to cache HTTP responses (default 24h)
//   - Refresh: Bypass cache for fresh data
//   - MetadataProviders: Enrichment sources (GitHub, GitLab)
//   - Logger: Progress callback
//
// # Package Data
//
// Each resolved package becomes a [Package] with:
//
//   - Name, Version: Package identity
//   - Dependencies: Direct dependency names
//   - Description, License, Author: Registry metadata
//   - Repository, HomePage: Source URLs
//   - Downloads: Popularity metric (where available)
//
// The [Package.Metadata] method converts this to a map suitable for
// node metadata in the DAG.
//
// # Manifest Parsing
//
// For local projects, parse manifest files directly:
//
//	parser := python.PoetryParser{}
//	result, _ := parser.Parse("poetry.lock", opts)
//	g := result.Graph
//
// Manifest parsers implement [ManifestParser] and may provide:
//
//   - Direct dependencies only (requirements.txt)
//   - Full transitive closure (poetry.lock, Cargo.lock)
//
// # Metadata Enrichment
//
// [MetadataProvider] implementations add data from external sources:
//
//	providers := []deps.MetadataProvider{
//	    metadata.NewGitHubProvider(token, ttl),
//	}
//	opts := deps.Options{MetadataProviders: providers}
//
// The GitHub provider adds: repo_stars, repo_owner, repo_maintainers,
// repo_last_commit, repo_archived—all used by Nebraska ranking and
// brittle detection.
//
// # Supported Languages
//
// Each language has a subpackage with its [Language] definition:
//
//   - [python]: PyPI, poetry.lock, requirements.txt, pyproject.toml
//   - [rust]: crates.io, Cargo.toml
//   - [javascript]: npm, package.json
//   - [ruby]: RubyGems, Gemfile
//   - [php]: Packagist, composer.json
//   - [java]: Maven Central, pom.xml
//   - [golang]: Go Module Proxy, go.mod
//
// [integrations]: github.com/matzehuels/stacktower/pkg/integrations
// [internal/cli]: github.com/matzehuels/stacktower/internal/cli
// [dag.DAG]: github.com/matzehuels/stacktower/pkg/dag.DAG
// [python]: github.com/matzehuels/stacktower/pkg/deps/python
// [rust]: github.com/matzehuels/stacktower/pkg/deps/rust
// [javascript]: github.com/matzehuels/stacktower/pkg/deps/javascript
// [ruby]: github.com/matzehuels/stacktower/pkg/deps/ruby
// [php]: github.com/matzehuels/stacktower/pkg/deps/php
// [java]: github.com/matzehuels/stacktower/pkg/deps/java
// [golang]: github.com/matzehuels/stacktower/pkg/deps/golang
package deps
