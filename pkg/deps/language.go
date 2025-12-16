package deps

import (
	"fmt"
	"time"
)

type Language struct {
	Name            string
	DefaultRegistry string
	RegistryAliases map[string]string
	ManifestTypes   []string
	ManifestAliases map[string]string
	NewResolver     func(ttl time.Duration) (Resolver, error)
	NewManifest     func(name string, res Resolver) ManifestParser
	ManifestParsers func(res Resolver) []ManifestParser
}

func (l *Language) Registry(name string) (Resolver, error) {
	name = l.alias(l.RegistryAliases, name)
	if name != l.DefaultRegistry {
		return nil, fmt.Errorf("unknown registry %q (available: %s)", name, l.DefaultRegistry)
	}
	return l.NewResolver(DefaultCacheTTL)
}

func (l *Language) Resolver() (Resolver, error) {
	return l.NewResolver(DefaultCacheTTL)
}

func (l *Language) Manifest(name string, res Resolver) (ManifestParser, bool) {
	if l.NewManifest == nil {
		return nil, false
	}
	p := l.NewManifest(l.alias(l.ManifestAliases, name), res)
	return p, p != nil
}

func (l *Language) HasManifests() bool {
	return l.NewManifest != nil
}

func (l *Language) alias(m map[string]string, name string) string {
	if v, ok := m[name]; ok {
		return v
	}
	return name
}
