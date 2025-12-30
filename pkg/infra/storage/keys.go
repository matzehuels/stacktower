package storage

import "fmt"

// =============================================================================
// Scope Determination (Single Source of Truth)
// =============================================================================

// DetermineScope returns the appropriate scope for the given inputs.
// This is the SINGLE SOURCE OF TRUTH for scope determination.
//
// Rules:
//   - Manifests (private repo files) → ScopeUser (isolated per user)
//   - Packages (public registry) → ScopeGlobal (shared across all users)
//   - Explicit scope overrides the default
func DetermineScope(explicitScope Scope, hasManifest bool) Scope {
	if explicitScope != "" {
		return explicitScope
	}
	if hasManifest {
		return ScopeUser
	}
	return ScopeGlobal
}

// =============================================================================
// Unified Key Generation
// =============================================================================
//
// This file provides the SINGLE SOURCE OF TRUTH for all cache key generation.
// All code that generates cache keys should use these functions to ensure
// consistency between API, worker, and pipeline components.
//
// Key Schema:
//   {type}:{scope}:{identifiers}:{options_hash}
//
// See doc.go for comprehensive documentation of the keying system.

// Keys provides methods for generating consistent cache keys.
// Use the package-level functions (GraphKey, RenderKey, etc.) for convenience.
var Keys = KeyBuilder{}

// KeyBuilder generates consistent cache keys across the system.
// This is exported to allow for testing and future extensibility.
type KeyBuilder struct{}

// =============================================================================
// Graph Keys
// =============================================================================

// GraphKey generates a cache key for a dependency graph.
// This is the canonical function for graph key generation.
//
// For global packages (ScopeGlobal):
//
//	graph:global:{language}:{package}:{options_hash}
//
// For user manifests (ScopeUser):
//
//	graph:user:{user_id}:{language}:{manifest_hash}:{options_hash}
func (KeyBuilder) GraphKey(scope Scope, userID, language, pkgOrManifest string, opts GraphOptions) string {
	optsHash := OptionsHash(opts)
	if scope == ScopeGlobal {
		return fmt.Sprintf("graph:global:%s:%s:%s", language, pkgOrManifest, optsHash)
	}
	return fmt.Sprintf("graph:user:%s:%s:%s:%s", userID, language, pkgOrManifest, optsHash)
}

// ManifestHash computes a full SHA-256 hash for manifest content.
// Unlike package names (which are short strings), manifests use content hashing.
// We use the FULL 64-char hash for collision resistance with user-controlled input.
func ManifestHash(manifestContent string) string {
	return Hash([]byte(manifestContent)) // Full 64-char SHA-256
}

// GraphKeyFromInputs generates a graph cache key from ParseInputs.
// This provides compatibility with the pipeline's ParseInputs struct.
func (KeyBuilder) GraphKeyFromInputs(scope Scope, userID string, inputs ParseInputs) string {
	// Determine package or manifest identifier
	pkgOrManifest := inputs.Package
	if inputs.ManifestHash != "" {
		// Use FULL hash - manifest content is user-controlled, needs collision resistance
		pkgOrManifest = inputs.ManifestHash
	}

	opts := GraphOptions{
		MaxDepth:  inputs.MaxDepth,
		MaxNodes:  inputs.MaxNodes,
		Normalize: inputs.Normalize,
	}

	return Keys.GraphKey(scope, userID, inputs.Language, pkgOrManifest, opts)
}

// =============================================================================
// Render/History Keys
// =============================================================================

// RenderHistoryKey generates a cache key for user render history lookups.
// This key is used for fast-path cache lookups to find existing renders.
//
// Format: render:user:{user_id}:{language}:{package}:{viz_type}
func (KeyBuilder) RenderHistoryKey(userID, language, pkgOrManifest, vizType string) string {
	return fmt.Sprintf("render:user:%s:%s:%s:%s", userID, language, pkgOrManifest, vizType)
}

// RenderDocumentID generates a MongoDB document ID from render key components.
// This ensures the document ID is deterministic and derived from the same
// inputs as the cache key, preventing mismatches.
//
// Format matches RenderHistoryKey: render:user:{user_id}:{language}:{package}:{viz_type}
// Returns a 24-character hex string (96 bits) for MongoDB compatibility.
//
// Security note: 96 bits provides collision resistance up to ~2^48 documents
// (birthday bound), which far exceeds expected scale. The truncation is safe
// because these IDs are derived from authenticated user inputs, not attacker-controlled.
func (KeyBuilder) RenderDocumentID(userID, language, pkgOrManifest, vizType string) string {
	key := fmt.Sprintf("render:user:%s:%s:%s:%s", userID, language, pkgOrManifest, vizType)
	return Hash([]byte(key))[:24]
}

// =============================================================================
// Content-Addressed Artifact Keys (with optional user scope)
// =============================================================================

// LayoutCacheKey generates a cache key for layout data.
// Layout is cached based on graph content hash and layout options.
// For user-scoped graphs, the layout is also user-scoped to prevent data leakage.
//
// For global (public packages):
//
//	layout:global:{graph_hash}:{options_hash}
//
// For user-scoped (private manifests):
//
//	layout:user:{user_id}:{graph_hash}:{options_hash}
func (KeyBuilder) LayoutCacheKey(scope Scope, userID, graphContentHash string, opts LayoutOptions) string {
	optsHash := OptionsHash(opts)
	if scope == ScopeGlobal {
		return fmt.Sprintf("layout:global:%s:%s", graphContentHash, optsHash)
	}
	return fmt.Sprintf("layout:user:%s:%s:%s", userID, graphContentHash, optsHash)
}

// ArtifactCacheKey generates a cache key for rendered artifacts.
// Artifacts are cached based on layout hash, format, and scope.
// For user-scoped graphs, artifacts are also user-scoped to prevent data leakage.
//
// For global (public packages):
//
//	artifact:global:{layout_hash}:{format}
//
// For user-scoped (private manifests):
//
//	artifact:user:{user_id}:{layout_hash}:{format}
func (KeyBuilder) ArtifactCacheKey(scope Scope, userID, layoutHash, format string) string {
	if scope == ScopeGlobal {
		return fmt.Sprintf("artifact:global:%s:%s", layoutHash, format)
	}
	return fmt.Sprintf("artifact:user:%s:%s:%s", userID, layoutHash, format)
}

// =============================================================================
// Parse Inputs (for pipeline compatibility)
// =============================================================================

// ParseInputs defines the inputs that affect graph parsing.
// This struct is hashed to generate graph cache keys.
// Duplicated from pipeline package to avoid circular imports.
type ParseInputs struct {
	Language         string `json:"language"`
	Package          string `json:"package,omitempty"`
	ManifestHash     string `json:"manifest_hash,omitempty"`
	ManifestFilename string `json:"manifest_filename,omitempty"`
	MaxDepth         int    `json:"max_depth"`
	MaxNodes         int    `json:"max_nodes"`
	Normalize        bool   `json:"normalize"`
	SkipEnrich       bool   `json:"skip_enrich"`
}
