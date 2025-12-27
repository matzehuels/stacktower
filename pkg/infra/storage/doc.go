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
