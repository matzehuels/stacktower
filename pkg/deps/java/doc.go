// Package java provides dependency resolution for Maven/Java packages.
//
// # Overview
//
// This package implements [deps.Language] for Java, supporting:
//
//   - Maven Central registry resolution via [maven] client
//   - pom.xml manifest parsing
//
// # Registry Resolution
//
// Use [Language.Resolver] to fetch dependencies from Maven Central:
//
//	resolver, _ := java.Language.Resolver()
//	g, _ := resolver.Resolve(ctx, "com.google.guava:guava", deps.Options{MaxDepth: 10})
//
// Package names use Maven coordinates: "groupId:artifactId".
//
// # Manifest Parsing
//
// Parse pom.xml files:
//
//	parser, _ := java.Language.Manifest("pom", nil)
//	result, _ := parser.Parse("pom.xml", deps.Options{})
//
// The parser extracts dependencies from <dependencies> elements,
// excluding test and provided scopes.
//
// [maven]: github.com/matzehuels/stacktower/pkg/integrations/maven
// [deps.Language]: github.com/matzehuels/stacktower/pkg/deps.Language
package java
