package cache

// ScopedKeyer wraps a Keyer with a prefix for multi-tenant isolation.
// This is useful in the cloud platform where different users or contexts
// need separate cache namespaces.
//
// Example usage:
//
//	// User-specific keys for private repos
//	userKeyer := NewScopedKeyer(NewDefaultKeyer(), "user:abc123:")
//
//	// Global keys for public packages
//	globalKeyer := NewDefaultKeyer()
type ScopedKeyer struct {
	inner  Keyer
	prefix string
}

// NewScopedKeyer creates a keyer with a prefix.
// The prefix is prepended to all generated keys.
func NewScopedKeyer(inner Keyer, prefix string) Keyer {
	if inner == nil {
		inner = NewDefaultKeyer()
	}
	return &ScopedKeyer{
		inner:  inner,
		prefix: prefix,
	}
}

// HTTPKey generates a prefixed key for HTTP response caching.
func (k *ScopedKeyer) HTTPKey(namespace, key string) string {
	return k.prefix + k.inner.HTTPKey(namespace, key)
}

// GraphKey generates a prefixed key for dependency graph caching.
func (k *ScopedKeyer) GraphKey(language, pkg string, opts GraphKeyOpts) string {
	return k.prefix + k.inner.GraphKey(language, pkg, opts)
}

// LayoutKey generates a prefixed key for layout caching.
func (k *ScopedKeyer) LayoutKey(graphHash string, opts LayoutKeyOpts) string {
	return k.prefix + k.inner.LayoutKey(graphHash, opts)
}

// ArtifactKey generates a prefixed key for artifact caching.
func (k *ScopedKeyer) ArtifactKey(layoutHash string, opts ArtifactKeyOpts) string {
	return k.prefix + k.inner.ArtifactKey(layoutHash, opts)
}
