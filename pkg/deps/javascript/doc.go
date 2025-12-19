// Package javascript provides dependency resolution for npm packages.
//
// # Overview
//
// This package implements [deps.Language] for JavaScript/Node.js, supporting:
//
//   - npm registry resolution via [npm] client
//   - package.json manifest parsing
//
// # Registry Resolution
//
// Use [Language.Resolver] to fetch dependencies from npm:
//
//	resolver, _ := javascript.Language.Resolver()
//	g, _ := resolver.Resolve(ctx, "express", deps.Options{MaxDepth: 10})
//
// The resolver fetches the "dependencies" field from each package, excluding
// devDependencies, peerDependencies, and optionalDependencies.
//
// # Manifest Parsing
//
// Parse package.json files:
//
//	parser, _ := javascript.Language.Manifest("npm", nil)
//	result, _ := parser.Parse("package.json", deps.Options{})
//
// Note: package.json contains direct dependencies only. The resolver fetches
// transitive dependencies from npm.
//
// [npm]: github.com/matzehuels/stacktower/pkg/integrations/npm
// [deps.Language]: github.com/matzehuels/stacktower/pkg/deps.Language
package javascript
