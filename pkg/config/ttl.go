package config

import "time"

// Cache TTLs - single source of truth
const (
	// GraphTTL is how long resolved dependency graphs are cached.
	// Longer TTL because dependency trees rarely change.
	GraphTTL = 7 * 24 * time.Hour // 7 days

	// LayoutTTL is how long computed layouts are cached.
	// Longer than graph because layout depends on graph hash.
	LayoutTTL = 30 * 24 * time.Hour // 30 days

	// RenderTTL is how long rendered artifacts (SVG/PNG/PDF) are cached.
	// Longest because renders are deterministic given layout hash.
	RenderTTL = 90 * 24 * time.Hour // 90 days

	// HTTPCacheTTL is the default TTL for HTTP response caching.
	HTTPCacheTTL = 24 * time.Hour

	// SessionTTL is the default session duration.
	SessionTTL = 24 * time.Hour

	// OAuthStateTTL is the default OAuth state token duration.
	OAuthStateTTL = 10 * time.Minute
)
