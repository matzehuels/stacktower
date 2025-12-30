// Package storage provides unified storage backends for Stacktower.
//
// This package consolidates caching and persistence into a single abstraction
// that works across all deployment modes: CLI, API, and Worker.
//
// # Architecture
//
// The storage package provides two levels of abstraction:
//
// 1. Backend (primary interface): Used by pipeline.Service and pkg/integrations
// for content-addressable caching of graphs, layouts, renders, and HTTP responses.
//
// 2. Index + DocumentStore (low-level): Used internally by DistributedBackend
// and directly by API/Worker for user-scoped data (render history).
//
//	┌─────────────────────────────────────────────────────────────────────────┐
//	│                         APPLICATION LAYER                               │
//	├─────────────────────────────────────────────────────────────────────────┤
//	│   CLI              API              Worker           Integrations       │
//	│    │                │                 │                   │             │
//	│    └────────────────┼─────────────────┼───────────────────┘             │
//	│                     │                 │                                 │
//	│                     ▼                 ▼                                 │
//	│              pipeline.Service   (also uses)                             │
//	│                     │               Index + DocumentStore               │
//	│                     │                 │                                 │
//	│                     │ uses            │ for user history                │
//	│                     ▼                 ▼                                 │
//	│                  Backend ◄─────────────────────────────────────────     │
//	│                     │               Unified caching abstraction         │
//	│         ┌───────────┼───────────┐                                       │
//	│         │           │           │                                       │
//	│         ▼           ▼           ▼                                       │
//	│   FileBackend  Distributed  NullBackend                                 │
//	│    (CLI)        Backend      (testing)                                  │
//	│         │           │                                                   │
//	│         ▼           ▼                                                   │
//	│   Local Files   Redis + MongoDB                                         │
//	└─────────────────────────────────────────────────────────────────────────┘
//
// # Backend Implementations
//
//   - FileBackend: File-based storage for CLI (uses local filesystem)
//   - DistributedBackend: Production storage for API/Worker (Redis + MongoDB)
//   - MemoryBackend: In-memory storage for testing and development
//   - NullBackend: No-op backend that never caches (for testing)
//
// # Cache Key Schema
//
// All cache keys follow a consistent namespaced format managed by [Keys]:
//
//	{type}:{scope}:{identifiers}:{options_hash}
//
// Graph Keys (dependency graphs):
//
//	Global packages (shared across all users):
//	  graph:global:{language}:{package}:{options_hash}
//	  Example: graph:global:python:flask:a1b2c3d4e5f6g7h8
//
//	User-scoped manifests (private to user):
//	  graph:user:{user_id}:{language}:{manifest_hash}:{options_hash}
//	  Example: graph:user:12345:python:abc123def456ghi789jkl012:a1b2c3d4
//
// Layout Keys (computed layouts - scoped same as source graph):
//
//	Global (public packages):
//	  layout:global:{graph_hash}:{options_hash}
//	  Example: layout:global:deadbeef:a1b2c3d4
//
//	User-scoped (private manifests):
//	  layout:user:{user_id}:{graph_hash}:{options_hash}
//	  Example: layout:user:12345:deadbeef:a1b2c3d4
//
// Artifact Keys (rendered SVG/PNG/PDF - scoped same as source graph):
//
//	Global (public packages):
//	  artifact:global:{combined_hash}:{format}
//	  Example: artifact:global:cafebabe:svg
//
//	User-scoped (private manifests):
//	  artifact:user:{user_id}:{combined_hash}:{format}
//	  Example: artifact:user:12345:cafebabe:svg
//
// User History Keys (for history page):
//
//	render:user:{user_id}:{language}:{package}:{viz_type}
//	Example: render:user:12345:python:flask:tower
//
// # Public vs Private Data
//
// Public packages from registries (PyPI, npm, etc.) are stored with [ScopeGlobal].
// These are shared across all users - if User A visualizes "flask", User B
// benefits from the cached graph.
//
// Private manifests are stored with [ScopeUser]. The cache key includes the
// user ID, ensuring isolation. Authorization is enforced at the API layer
// via scoped methods like [DocumentStore.GetGraphDocScoped].
//
// # Usage
//
// CLI (local file caching):
//
//	backend, _ := storage.NewFileBackend(storage.FileConfig{
//	    CacheDir: "~/.stacktower/cache",
//	})
//	defer backend.Close()
//	svc := pipeline.NewService(backend)
//
// API/Worker (distributed caching):
//
//	redis, _ := infra.NewRedis(ctx, cfg.Redis)
//	mongo, _ := infra.NewMongo(ctx, cfg.Mongo)
//	backend := storage.NewDistributedBackend(redis.Index(), mongo.DocumentStore())
//	svc := pipeline.NewService(backend)
//
// Testing (no caching):
//
//	svc := pipeline.NewService(storage.NullBackend{})
//
// Integrations (HTTP caching):
//
//	client := integrations.NewClient(backend, "pypi:", 24*time.Hour, nil)
//
// Key Generation (always use [Keys]):
//
//	// Generate a graph cache key
//	key := storage.Keys.GraphKey(storage.ScopeGlobal, "", "python", "flask", opts)
//
//	// Generate a render history key
//	key := storage.Keys.RenderHistoryKey(userID, "python", "flask", "tower")
//
// # Two-Tier Caching (Distributed Mode)
//
// In production, DistributedBackend uses a two-tier caching strategy:
//
//   - Tier 1 (Index): Fast TTL-based lookups via Redis
//     "Do we have this content?" → returns document ID
//
//   - Tier 2 (DocumentStore): Durable storage via MongoDB
//     Actual graphs, renders, and binary artifacts (GridFS)
//
// This separation allows Redis to expire entries without losing MongoDB data,
// and enables efficient cache invalidation strategies.
package storage
