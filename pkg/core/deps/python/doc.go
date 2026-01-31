// Package python provides dependency resolution for Python packages.
//
// # Overview
//
// This package implements [deps.Language] for Python, supporting:
//
//   - PyPI registry resolution via [pypi] client
//   - poetry.lock manifest parsing (full transitive closure)
//   - requirements.txt parsing (direct dependencies, resolved via PyPI)
//
// # Registry Resolution
//
// Use [Language.Resolver] to fetch dependencies from PyPI:
//
//	resolver, _ := python.Language.Resolver()
//	g, _ := resolver.Resolve(ctx, "fastapi", deps.Options{MaxDepth: 10})
//
// # Manifest Parsing
//
// Parse local manifest files:
//
//	parser, _ := python.Language.Manifest("poetry", nil)
//	result, _ := parser.Parse("poetry.lock", deps.Options{})
//
// Supported manifests:
//
//   - poetry.lock: Full dependency graph with versions (IncludesTransitive: true)
//   - requirements.txt: Direct deps only, resolved via PyPI (IncludesTransitive: false)
//
// # Package Name Normalization
//
// Python package names are normalized following PEP 503: converted to
// lowercase with runs of [_.-] replaced by single hyphens.
//
// [pypi]: github.com/matzehuels/stacktower/pkg/integrations/pypi
// [deps.Language]: github.com/matzehuels/stacktower/pkg/core/deps.Language
package python
