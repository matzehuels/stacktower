package deps

import (
	"fmt"
	"time"
)

// Language defines how to resolve dependencies for a programming language.
// It maps registry names to resolvers and manifest file types to parsers.
type Language struct {
	Name            string                                         // Language identifier (e.g., "python", "rust")
	DefaultRegistry string                                         // Primary registry (e.g., "pypi", "crates")
	RegistryAliases map[string]string                              // Alternative names for registries
	ManifestTypes   []string                                       // Supported manifest types (e.g., "poetry", "cargo")
	ManifestAliases map[string]string                              // Filename to type mappings
	NewResolver     func(ttl time.Duration) (Resolver, error)      // Factory for registry resolver
	NewManifest     func(name string, res Resolver) ManifestParser // Factory for manifest parsers
	ManifestParsers func(res Resolver) []ManifestParser            // All available manifest parsers
}

// Registry returns a Resolver for the named registry, resolving aliases.
func (l *Language) Registry(name string) (Resolver, error) {
	name = l.alias(l.RegistryAliases, name)
	if name != l.DefaultRegistry {
		return nil, fmt.Errorf("unknown registry %q (available: %s)", name, l.DefaultRegistry)
	}
	return l.NewResolver(DefaultCacheTTL)
}

// Resolver returns the default registry resolver for this language.
func (l *Language) Resolver() (Resolver, error) {
	return l.NewResolver(DefaultCacheTTL)
}

// Manifest returns a parser for the named manifest type, resolving aliases.
// Returns nil, false if the manifest type is not supported.
func (l *Language) Manifest(name string, res Resolver) (ManifestParser, bool) {
	if l.NewManifest == nil {
		return nil, false
	}
	p := l.NewManifest(l.alias(l.ManifestAliases, name), res)
	return p, p != nil
}

// HasManifests reports whether this language supports manifest file parsing.
func (l *Language) HasManifests() bool {
	return l.NewManifest != nil
}

func (l *Language) alias(m map[string]string, name string) string {
	if v, ok := m[name]; ok {
		return v
	}
	return name
}
