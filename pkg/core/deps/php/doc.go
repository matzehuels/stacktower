// Package php provides dependency resolution for PHP Composer packages.
//
// # Overview
//
// This package implements [deps.Language] for PHP, supporting:
//
//   - Packagist registry resolution via [packagist] client
//   - composer.json manifest parsing
//
// # Registry Resolution
//
// Use [Language.Resolver] to fetch dependencies from Packagist:
//
//	resolver, _ := php.Language.Resolver()
//	g, _ := resolver.Resolve(ctx, "symfony/console", deps.Options{MaxDepth: 10})
//
// # Manifest Parsing
//
// Parse composer.json files:
//
//	parser, _ := php.Language.Manifest("composer", nil)
//	result, _ := parser.Parse("composer.json", deps.Options{})
//
// Note: composer.json contains direct dependencies in "require". The
// resolver fetches transitive dependencies from Packagist.
//
// [packagist]: github.com/matzehuels/stacktower/pkg/integrations/packagist
// [deps.Language]: github.com/matzehuels/stacktower/pkg/core/deps.Language
package php
