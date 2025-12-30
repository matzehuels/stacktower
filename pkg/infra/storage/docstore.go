package storage

import "context"

// Note: ErrAccessDenied is defined in types.go (re-exported from infra)

// DocumentStore is Tier 2 of the two-tier cache: durable document storage.
//
// In production, this is backed by MongoDB. It stores the actual data:
// Graph documents, Render documents, and binary Artifacts (via GridFS).
//
// The IDs returned by DocumentStore methods are what go into CacheEntry.DocumentID
// for the Index.
//
// Note: Method names use "Doc" suffix to distinguish from Backend methods.
// Backend.GetGraph(hash) → *dag.DAG (for pipeline)
// DocumentStore.GetGraphDoc(id) → *Graph (for low-level access)
//
// # Implementations
//
//   - Production: infra.Mongo.DocumentStore() returns a MongoDB-backed DocumentStore
//   - Development: MemoryBackend implements DocumentStore (and Index)
type DocumentStore interface {
	// GetGraphDoc retrieves a graph document by its document ID.
	// Returns (graph, nil) on success, (nil, nil) if not found.
	GetGraphDoc(ctx context.Context, id string) (*Graph, error)

	// GetGraphDocScoped retrieves a graph document with authorization check.
	// For ScopeUser graphs, verifies the requesting user matches the owner.
	// Returns ErrAccessDenied if user doesn't have access.
	GetGraphDocScoped(ctx context.Context, id string, userID string) (*Graph, error)

	// StoreGraphDoc saves a graph document. Sets graph.ID if empty (new graph).
	// Returns the stored graph ID via graph.ID.
	StoreGraphDoc(ctx context.Context, graph *Graph) error

	// GetRenderDoc retrieves a render document by its document ID.
	GetRenderDoc(ctx context.Context, id string) (*Render, error)

	// GetRenderDocScoped retrieves a render document with authorization check.
	// Returns ErrAccessDenied if user doesn't own this render.
	GetRenderDocScoped(ctx context.Context, id string, userID string) (*Render, error)

	// StoreRenderDoc saves a render document. Sets render.ID if empty.
	StoreRenderDoc(ctx context.Context, render *Render) error

	// UpsertRenderDoc inserts or updates a render document by ID.
	// If a render with the same ID exists, it updates; otherwise creates new.
	// This prevents duplicate history entries when re-rendering.
	UpsertRenderDoc(ctx context.Context, render *Render) error

	// DeleteRenderDoc removes a render and its associated artifacts.
	DeleteRenderDoc(ctx context.Context, id string) error

	// DeleteRenderDocScoped removes a render only if owned by userID.
	// Returns ErrAccessDenied if user doesn't own this render.
	DeleteRenderDocScoped(ctx context.Context, id string, userID string) error

	// StoreArtifact saves a binary file (SVG, PNG, PDF) to GridFS.
	// Returns the artifact ID for later retrieval.
	StoreArtifact(ctx context.Context, renderID, filename string, data []byte, userID string) (string, error)

	// GetArtifact retrieves a binary file from GridFS by ID.
	GetArtifact(ctx context.Context, artifactID string) ([]byte, error)

	// GetArtifactScoped retrieves an artifact with authorization check.
	// renderID is used to verify ownership via the parent render document.
	// Returns ErrAccessDenied if user doesn't own the parent render.
	GetArtifactScoped(ctx context.Context, artifactID string, userID string) ([]byte, error)

	// Ping checks if the document store backend is reachable.
	Ping(ctx context.Context) error

	// CountUniqueTowers returns the number of unique packages/manifests visualized.
	// This dedupes by (language, package) - different viz types for same package count as 1.
	CountUniqueTowers(ctx context.Context) (int64, error)

	// CountUniqueUsers returns the number of unique users with renders (for stats).
	CountUniqueUsers(ctx context.Context) (int64, error)

	// CountUniqueDependencies returns the count of unique dependency nodes across all graphs.
	// This requires extracting node IDs from graph JSON and deduping.
	CountUniqueDependencies(ctx context.Context) (int64, error)

	// ListPackageSuggestions returns package names that match a query prefix for a given language.
	// This is used for autocomplete in the frontend, drawing from global history.
	// Results are ordered by popularity (most frequently rendered first).
	ListPackageSuggestions(ctx context.Context, language string, query string, limit int) ([]PackageSuggestion, error)

	// ListExplore returns public towers for the explore page.
	// Groups by (language, package), includes popularity count.
	// sortBy: "popular" (default) or "recent"
	ListExplore(ctx context.Context, language, sortBy string, limit, offset int) ([]ExploreEntry, int64, error)

	// ==========================================================================
	// Canonical Renders & User Library
	// ==========================================================================

	// GetCanonicalRender looks up a canonical (shared) render for a public package.
	// Canonical renders have user_id="" and are shared across all users.
	GetCanonicalRender(ctx context.Context, language, pkg, vizType string) (*Render, error)

	// SaveToLibrary adds a package to a user's library. Idempotent.
	SaveToLibrary(ctx context.Context, userID, language, pkg string) error

	// RemoveFromLibrary removes a package from a user's library. Idempotent.
	RemoveFromLibrary(ctx context.Context, userID, language, pkg string) error

	// IsInLibrary checks if a package is in a user's library.
	IsInLibrary(ctx context.Context, userID, language, pkg string) (bool, error)

	// ListLibrary returns a user's saved public packages.
	ListLibrary(ctx context.Context, userID string, limit, offset int) ([]LibraryEntry, int64, error)

	// ListPrivateRenders returns a user's private repo renders (manifests).
	ListPrivateRenders(ctx context.Context, userID string, limit, offset int) ([]*Render, int64, error)

	Close() error
}

// Cache combines both tiers (Index + DocumentStore) for unified access.
//
// Use this interface when you need access to both tiers, such as in
// DistributedBackend or the Worker for user history operations.
//
// # Implementations
//
//   - Production: storage.NewDistributedBackend(redis.Index(), mongo.DocumentStore())
//   - Development: storage.NewMemoryBackend() implements Cache directly
type Cache interface {
	Index
	DocumentStore
}
