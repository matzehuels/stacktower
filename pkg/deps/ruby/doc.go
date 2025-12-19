// Package ruby provides dependency resolution for Ruby gems.
//
// # Overview
//
// This package implements [deps.Language] for Ruby, supporting:
//
//   - RubyGems.org registry resolution via [rubygems] client
//   - Gemfile parsing
//
// # Registry Resolution
//
// Use [Language.Resolver] to fetch dependencies from RubyGems:
//
//	resolver, _ := ruby.Language.Resolver()
//	g, _ := resolver.Resolve(ctx, "rails", deps.Options{MaxDepth: 10})
//
// # Manifest Parsing
//
// Parse Gemfile:
//
//	parser, _ := ruby.Language.Manifest("gem", nil)
//	result, _ := parser.Parse("Gemfile", deps.Options{})
//
// Note: Gemfile parsing extracts gem names from `gem "name"` declarations.
// Transitive dependencies are resolved via RubyGems.
//
// [rubygems]: github.com/matzehuels/stacktower/pkg/integrations/rubygems
// [deps.Language]: github.com/matzehuels/stacktower/pkg/deps.Language
package ruby
