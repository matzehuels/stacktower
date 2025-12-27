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

	// DeleteRenderDoc removes a render and its associated artifacts.
	DeleteRenderDoc(ctx context.Context, id string) error

	// DeleteRenderDocScoped removes a render only if owned by userID.
	// Returns ErrAccessDenied if user doesn't own this render.
	DeleteRenderDocScoped(ctx context.Context, id string, userID string) error

	// ListRenderDocs returns a user's render history (paginated).
	ListRenderDocs(ctx context.Context, userID string, limit, offset int) ([]*Render, int64, error)

	// StoreArtifact saves a binary file (SVG, PNG, PDF) to GridFS.
	// Returns the artifact ID for later retrieval.
	StoreArtifact(ctx context.Context, renderID, filename string, data []byte) (string, error)

	// GetArtifact retrieves a binary file from GridFS by ID.
	GetArtifact(ctx context.Context, artifactID string) ([]byte, error)

	// GetArtifactScoped retrieves an artifact with authorization check.
	// renderID is used to verify ownership via the parent render document.
	// Returns ErrAccessDenied if user doesn't own the parent render.
	GetArtifactScoped(ctx context.Context, artifactID string, userID string) ([]byte, error)

	// Ping checks if the document store backend is reachable.
	Ping(ctx context.Context) error

	Close() error
}

// OperationStore records user operation history for all pipeline stages.
// This enables tracking "which user triggered how many layouts" etc.
//
// # Implementations
//
//   - Production: infra.Mongo provides OperationStore via MongoDB
//   - Development: MemoryBackend implements OperationStore
type OperationStore interface {
	// RecordOperation logs a pipeline operation for a user.
	RecordOperation(ctx context.Context, op *Operation) error

	// ListOperations returns a user's operation history (paginated).
	// opType can filter by operation type (empty string = all types).
	ListOperations(ctx context.Context, userID string, opType OperationType, limit, offset int) ([]*Operation, int64, error)

	// CountOperations counts operations for a user within a time window.
	// Used for rate limiting checks.
	CountOperationsInWindow(ctx context.Context, userID string, opType OperationType, windowStart int64) (int64, error)

	// GetOperationStats returns aggregate stats for a user.
	GetOperationStats(ctx context.Context, userID string) (*UserOperationStats, error)
}

// UserOperationStats contains aggregate operation statistics for a user.
type UserOperationStats struct {
	TotalOperations  int64 `json:"total_operations" bson:"total_operations"`
	TotalParses      int64 `json:"total_parses" bson:"total_parses"`
	TotalLayouts     int64 `json:"total_layouts" bson:"total_layouts"`
	TotalRenders     int64 `json:"total_renders" bson:"total_renders"`
	TotalCacheHits   int64 `json:"total_cache_hits" bson:"total_cache_hits"`
	StorageBytesUsed int64 `json:"storage_bytes_used" bson:"storage_bytes_used"`
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
