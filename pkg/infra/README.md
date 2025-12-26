# Infrastructure Package (`pkg/infra`)

This package provides production infrastructure components for Stacktower. It follows a **layered architecture** that supports three deployment modes: CLI, API, and Worker.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          APPLICATION LAYER                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   internal/cli          internal/api           internal/worker              │
│        │                     │                      │                       │
│        │                     │                      │                       │
│        └─────────────────────┼──────────────────────┘                       │
│                              │                                              │
│                              ▼                                              │
│                     pipeline.Service                                        │
│                              │                                              │
│                              │ uses                                         │
│                              ▼                                              │
│                     artifact.Backend ◄────── Unified caching abstraction    │
│                              │               (pipeline artifacts + HTTP)    │
│                              │                                              │
│              ┌───────────────┼───────────────┐                              │
│              │               │               │                              │
│              ▼               ▼               ▼                              │
│       LocalBackend     ProdBackend     NullBackend                          │
│       (CLI only)       (API/Worker)    (testing)                            │
│              │               │                                              │
│              │               │ wraps                                        │
│              ▼               ▼                                              │
│      Local Files       cache.Cache ◄────────── Two-tier caching             │
│                              │                                              │
│              ┌───────────────┴───────────────┐                              │
│              │                               │                              │
│              ▼                               ▼                              │
│       LookupCache (Tier 1)            Store (Tier 2)                        │
│       "Do we have this?"              "Actual data"                         │
│              │                               │                              │
│              ▼                               ▼                              │
│           Redis                          MongoDB                            │
│       (TTL index + HTTP cache)       (Documents + GridFS)                   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Package Structure

```
pkg/infra/
├── redis.go           # Unified Redis client (queue, session, lookup cache, HTTP cache)
├── mongo.go           # Unified MongoDB client (document store, GridFS)
├── config.go          # Environment-based configuration
├── doc.go             # Package documentation
├── README.md          # This file
│
├── common/            # Shared utilities (errors, TTLs, hash, logging, retry)
│   └── common.go
│
├── cache/             # Two-tier caching for graphs, renders, artifacts, HTTP responses
│   ├── cache.go       # Interfaces: LookupCache, Store, Cache + data types
│   ├── memory.go      # In-memory implementation (local development)
│   └── combined.go    # Combines LookupCache + Store for production
│
├── artifact/          # Unified caching backend for pipeline + HTTP
│   ├── backend.go     # Backend interface (graphs, layouts, renders, HTTP)
│   ├── artifact.go    # Type constants and hash utilities
│   ├── local_backend.go   # CLI: filesystem-based storage
│   ├── prod_backend.go    # API/Worker: wraps cache.Cache
│   └── doc.go
│
├── session/           # User session management
│   ├── session.go     # Store and StateStore interfaces
│   ├── memory.go      # In-memory implementation
│   └── file.go        # File-based implementation (CLI)
│
└── queue/             # Job queue abstraction
    ├── queue.go       # Queue interface
    ├── types.go       # Job types (parse, layout, render)
    └── memory.go      # In-memory implementation (testing)
```

## Key Concepts

### 1. Two-Tier Caching (`cache/`)

The cache package implements a two-tier caching strategy:

| Tier | Interface | Implementation | Purpose |
|------|-----------|----------------|---------|
| **Tier 1** | `LookupCache` | Redis | Fast TTL-based index: "Do we have this content?" Returns MongoDB ID. Also stores HTTP response cache. |
| **Tier 2** | `Store` | MongoDB | Durable storage for graphs, renders, and binary artifacts (GridFS) |

**Why two tiers?**
- Redis is fast but volatile → good for TTL-based lookups and HTTP cache
- MongoDB is durable → good for actual data storage
- Separation allows Redis to expire entries without losing MongoDB data

```go
// Flow for cache lookup:
// 1. Check Redis for cache key → get MongoDB ID
// 2. Fetch from MongoDB using ID
// 3. If Redis miss → compute, store in MongoDB, update Redis
```

### 2. Artifact Backend (`artifact/`)

The artifact package provides a **unified interface** for both:
- **Pipeline artifacts**: Parsed dependency graphs, computed layouts, rendered artifacts (SVG, PNG, PDF)
- **HTTP response caching**: Package registry API responses (PyPI, npm, crates.io, etc.)

| Backend | Used By | Storage |
|---------|---------|---------|
| `LocalBackend` | CLI | Local filesystem + JSON index |
| `ProdBackend` | API, Worker | Wraps `cache.Cache` (Redis+MongoDB) |
| `NullBackend` | Tests | No caching |

```go
// pipeline.Service doesn't know about Redis/MongoDB
// It just calls backend.GetGraph() / backend.PutGraph()
svc := pipeline.NewService(artifact.NewLocalBackend(...))  // CLI
svc := pipeline.NewService(artifact.NewProdBackend(cache)) // API

// pkg/integrations also uses the same backend for HTTP caching
pypiClient := pypi.NewClient(backend, 24*time.Hour)
```

**Key insight**: By unifying pipeline artifacts and HTTP caching into one `Backend` interface, you configure caching once (CLI uses files, API uses Redis+MongoDB) and both pipeline and integrations work automatically.

### 3. Unified Redis/MongoDB Clients

The root `infra` package provides unified clients that implement multiple interfaces from a single connection:

```go
redis, _ := infra.NewRedis(ctx, cfg.Redis)
defer redis.Close()

// One connection, multiple interfaces:
queue := redis.Queue()           // queue.Queue
sessions := redis.Sessions()     // session.Store
states := redis.OAuthStates()    // session.StateStore
lookup := redis.Cache()          // cache.LookupCache (includes HTTP cache)
```

```go
mongo, _ := infra.NewMongo(ctx, cfg.Mongo)
defer mongo.Close()

store := mongo.Store()           // cache.Store
db := mongo.Database()           // *mongo.Database (for GridFS)
```

## Usage by Application

### CLI (`internal/cli`)

```go
// Uses LocalBackend for filesystem caching
backend, _ := artifact.NewLocalBackend(artifact.LocalBackendConfig{
    CacheDir: "~/.stacktower/cache",
})
defer backend.Close()

// Both pipeline and integrations use the same backend
svc := pipeline.NewService(backend)
result, _, _ := svc.ExecuteFull(ctx, opts)
```

### API (`internal/api`)

```go
// Uses Redis + MongoDB via ProdBackend
redis, _ := infra.NewRedis(ctx, cfg.Redis)
mongo, _ := infra.NewMongo(ctx, cfg.Mongo)

cache := cache.NewCombinedCache(redis.Cache(), mongo.Store())
backend := artifact.NewProdBackend(cache)

server := api.New(
    redis.Queue(),
    cache,
    api.WithSessions(redis.Sessions()),
)
```

### Worker (`internal/worker`)

```go
// Same as API - uses Redis + MongoDB
redis, _ := infra.NewRedis(ctx, cfg.Redis)
mongo, _ := infra.NewMongo(ctx, cfg.Mongo)

cache := cache.NewCombinedCache(redis.Cache(), mongo.Store())

worker := worker.New(redis.Queue(), cache, worker.Config{
    Concurrency: 4,
})
worker.Start(ctx)
```

## Common Mistakes to Avoid

1. **Don't use `cache.Cache` directly in pipeline code**
   - Use `artifact.Backend` instead
   - This keeps pipeline code backend-agnostic

2. **Don't create multiple Redis/MongoDB connections**
   - Use the unified clients in `infra/redis.go` and `infra/mongo.go`
   - They provide multiple interfaces from a single connection

3. **Don't create caching per integration client**
   - Pass the same `artifact.Backend` to all integration clients
   - They automatically namespace their keys (e.g., "pypi:", "npm:")

## Configuration

All configuration is loaded from environment variables via `infra.Load()`:

```bash
# Redis
STACKTOWER_REDIS_ADDR=localhost:6379
STACKTOWER_REDIS_PASSWORD=
STACKTOWER_REDIS_DB=0

# MongoDB
STACKTOWER_MONGODB_URI=mongodb://localhost:27017
STACKTOWER_MONGODB_DATABASE=stacktower

# GitHub (for OAuth)
GITHUB_CLIENT_ID=...
GITHUB_CLIENT_SECRET=...
```

See `config.go` for all available options.
