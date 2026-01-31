// Package golang provides dependency resolution for Go modules.
//
// # Overview
//
// This package implements [deps.Language] for Go, supporting:
//
//   - Go Module Proxy resolution via [goproxy] client
//   - go.mod manifest parsing
//
// # Registry Resolution
//
// Use [Language.Resolver] to fetch dependencies from the Go Module Proxy:
//
//	resolver, _ := golang.Language.Resolver()
//	g, _ := resolver.Resolve(ctx, "github.com/spf13/cobra", deps.Options{MaxDepth: 10})
//
// Package names are full module paths (e.g., "github.com/user/repo").
//
// # Manifest Parsing
//
// Parse go.mod files:
//
//	parser, _ := golang.Language.Manifest("gomod", nil)
//	result, _ := parser.Parse("go.mod", deps.Options{})
//
// The parser extracts require directives. Note that go.mod typically
// lists direct dependencies; transitive dependencies are resolved via
// the Go Module Proxy.
//
// [goproxy]: github.com/matzehuels/stacktower/pkg/integrations/goproxy
// [deps.Language]: github.com/matzehuels/stacktower/pkg/core/deps.Language
package golang
