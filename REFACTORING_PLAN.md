# Stacktower Infrastructure Cleanup Plan

> **Status:** ✅ Completed  
> **Created:** 2024-12-24  
> **Completed:** 2024-12-25  
> **Author:** Code Review Analysis

## Executive Summary

The `pkg/` directory has **significant duplication** and **overlapping abstractions** that have grown organically. This document provides a phased plan to clean up the codebase.

### Key Issues Identified

1. **Duplicate error sentinels** across 6 packages
2. **Duplicate TTL constants** across 3 packages (with inconsistencies!)
3. **Duplicate hash utilities** across 3 packages
4. **Overlapping storage/cache abstractions** (6+ interfaces doing similar things)
5. **Import bug** in `integrations` package (`httpclient` doesn't exist)

---

## Table of Contents

- [Current Architecture](#current-architecture)
- [Target Architecture](#target-architecture)
- [Detailed Findings](#detailed-findings)
- [Implementation Phases](#implementation-phases)
  - [Phase 0: Fix Critical Bug](#phase-0-fix-critical-bug)
  - [Phase 1: Consolidate Constants](#phase-1-consolidate-constants)
  - [Phase 2: Merge backend into httpcache](#phase-2-merge-backend-into-httpcache)
  - [Phase 3: Simplify Artifact Package](#phase-3-simplify-artifact-package)
  - [Phase 4: Clean Up Infra Package](#phase-4-clean-up-infra-package)
  - [Phase 5: Update Documentation](#phase-5-update-documentation)
- [Implementation Order](#implementation-order)
- [Risk Assessment](#risk-assessment)
- [Testing Strategy](#testing-strategy)
- [Out of Scope](#out-of-scope)

---

## Current Architecture

```
┌──────────────────────────────────────────────────────────────────────────┐
│                         CURRENT STATE                                     │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   CLI Path                            API Path                           │
│   ────────                            ────────                           │
│   artifact.LocalBackend               artifact.ProdBackend               │
│        │                                   │                             │
│        ↓                                   ↓                             │
│   storage.Storage                     cache.Cache                        │
│   (FilesystemStorage)                  │       │                         │
│        │                               │       │                         │
│        │                     ┌─────────┘       └──────────┐              │
│        │                     ↓                            ↓              │
│        │              cache.LookupCache            cache.Store           │
│        │              (Redis via infra)           (MongoDB via infra)    │
│        │                     │                            │              │
│        │                     └────────────┬───────────────┘              │
│        │                                  │                              │
│        ↓                                  ↓                              │
│   Local Files                      Redis + MongoDB                       │
│                                                                          │
│   ALSO: httpcache.Cache ─→ backend.Backend (separate caching layer!)     │
│         session.Store (another TTL K-V!)                                 │
│         queue.Queue (another interface!)                                 │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### Problems with Current Architecture

1. **`backend.Backend`** and **`cache.LookupCache`** are nearly identical interfaces
2. **`storage.Storage`** and **`cache.Store`** overlap for artifact storage
3. **`artifact.LocalBackend`** uses `storage.Storage` but **`artifact.ProdBackend`** uses `cache.Cache` — inconsistent
4. **`session.Store`** is another K-V with TTL, same as `backend.Backend`
5. **`httpcache.Cache`** wraps `backend.Backend` but they're in separate packages

---

## Target Architecture

```
┌──────────────────────────────────────────────────────────────────────────┐
│                         TARGET STATE                                      │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   pkg/errors/errors.go          ← All sentinel errors                    │
│   pkg/config/ttl.go             ← All TTL constants                      │
│   pkg/hash/hash.go              ← All hash utilities                     │
│                                                                          │
│   ┌──────────────────────────────────────────────────────────────────┐   │
│   │  pkg/kv/                                                         │   │
│   │  ─────────                                                       │   │
│   │  Generic K-V store with TTL (replaces backend + httpcache.Cache) │   │
│   │  • kv.Store interface                                            │   │
│   │  • kv.Memory, kv.Filesystem, kv.Redis implementations            │   │
│   │  • kv.Cache wrapper (JSON serialization + namespacing)           │   │
│   └──────────────────────────────────────────────────────────────────┘   │
│                                                                          │
│   pkg/cache/         (Tier 2 - unchanged, clean)                         │
│   pkg/storage/       (Blob storage - unchanged, clean)                   │
│   pkg/session/       (Uses kv.Store internally)                          │
│   pkg/queue/         (Unchanged)                                         │
│   pkg/artifact/      (Unchanged interface, cleaner internals)            │
│   pkg/infra/         (Factory for production implementations)            │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## Detailed Findings

### 1. Duplicate Error Sentinels

The same errors are defined in multiple packages:

| Error | Locations |
|-------|-----------|
| `ErrNotFound` | `cache/cache.go:47`, `storage/storage.go:51`, `session/session.go:69`, `integrations/common.go:24` |
| `ErrExpired` | `backend/backend.go:40`, `cache/cache.go:50`, `session/session.go:72` |
| `ErrCacheMiss` | `artifact/artifact.go:60` (essentially the same as ErrNotFound) |

### 2. Duplicate TTL Constants

The same TTLs are defined in **3 different places** with **inconsistencies**:

| Constant | `artifact/artifact.go` | `cache/cache.go` | `pipeline/service.go` |
|----------|------------------------|------------------|----------------------|
| `GraphTTL` | 7 days | 7 days | 7 days |
| `LayoutTTL` | 30 days | - | 30 days |
| `RenderTTL` | **90 days** | **24 hours** ⚠️ | **90 days** |

> ⚠️ **BUG:** `cache.RenderTTL` is 24 hours but `artifact.RenderTTL` is 90 days!

### 3. Duplicate Hash Utilities

SHA256 hashing is implemented multiple times:

```go
// artifact/local_backend.go:263-272
func Hash(data []byte) string {
    h := sha256.Sum256(data)
    return hex.EncodeToString(h[:])
}
func HashJSON(v interface{}) string { ... }

// cache/cache.go:216-226
func ContentHash(data []byte) string { ... }  // Same as above
func OptionsHash(opts interface{}) string { ... } // Same as HashJSON but truncated

// httpcache/cache.go:157-160
func (c *Cache) hashKey(key string) string { ... } // Same logic
```

### 4. Overlapping Abstractions

Six+ interfaces doing similar things:

| Interface | Package | Purpose |
|-----------|---------|---------|
| `backend.Backend` | `pkg/backend` | K-V with TTL |
| `cache.LookupCache` | `pkg/cache` | K-V with TTL (Tier 1) |
| `cache.Store` | `pkg/cache` | Document storage (Tier 2) |
| `cache.Cache` | `pkg/cache` | Combined interface |
| `storage.Storage` | `pkg/storage` | Blob storage |
| `artifact.Backend` | `pkg/artifact` | Pipeline artifact caching |
| `session.Store` | `pkg/session` | Session K-V with TTL |

### 5. Import Bug

```go
// pkg/integrations/client.go - BROKEN!
import "github.com/matzehuels/stacktower/pkg/httpclient"  // ← DOES NOT EXIST

// pkg/integrations/common.go - Correct
import "github.com/matzehuels/stacktower/pkg/httpcache"
```

The package is `httpcache`, not `httpclient`. This will cause compilation failures.

### 6. Memory Implementation Duplication

Five nearly identical in-memory implementations:

| Package | Type | Pattern |
|---------|------|---------|
| `backend` | `Memory` | map + sync.RWMutex + expiration |
| `cache` | `MemoryCache` | map + sync.RWMutex + expiration |
| `storage` | `MemoryStorage` | map + sync.RWMutex |
| `session` | `MemoryStore` | map + sync.RWMutex + expiration |
| `queue` | `MemoryQueue` | map + sync.RWMutex |

### 7. Filesystem Implementation Duplication

Similar pattern with filesystem storage:

| Package | Type | Default Path |
|---------|------|--------------|
| `backend` | `Filesystem` | `~/.cache/stacktower/` |
| `storage` | `FilesystemStorage` | Configurable root |
| `session` | `FileStore` | `~/.config/stacktower/sessions/` |

---

## Implementation Phases

### Phase 0: Fix Critical Bug

**Time Estimate:** 1 hour  
**Risk:** Low

#### Task 0.1: Fix `httpclient` Import Bug

**Files to modify:**
- `pkg/integrations/client.go`
- `pkg/integrations/client_test.go`

**Changes:**

```go
// BEFORE
import "github.com/matzehuels/stacktower/pkg/httpclient"

// AFTER
import "github.com/matzehuels/stacktower/pkg/httpcache"
```

**Also update function calls:**

```go
// BEFORE
httpclient.RetryWithBackoff(...)
httpclient.Retryable(...)

// AFTER
httpcache.RetryWithBackoff(...)
httpcache.Retryable(...)
```

**Verification:**
```bash
go build ./...
```

---

### Phase 1: Consolidate Constants

**Time Estimate:** 2-3 hours  
**Risk:** Low

#### Task 1.1: Create Shared Errors Package

**Create:** `pkg/errors/errors.go`

```go
package errors

import "errors"

// Sentinel errors used across packages
var (
    ErrNotFound   = errors.New("not found")
    ErrExpired    = errors.New("expired")
    ErrCacheMiss  = errors.New("cache miss")
    ErrNetwork    = errors.New("network error")
    ErrInvalid    = errors.New("invalid")
)
```

**Files to update:**

| File | Errors to Replace |
|------|-------------------|
| `backend/backend.go:40` | `ErrExpired` |
| `cache/cache.go:47-50` | `ErrNotFound`, `ErrExpired` |
| `storage/storage.go:51` | `ErrNotFound` |
| `session/session.go:69-75` | `ErrNotFound`, `ErrExpired`, `ErrInvalidState` |
| `integrations/common.go:24-30` | `ErrNotFound`, `ErrNetwork` |
| `artifact/artifact.go:60` | `ErrCacheMiss` |

**Migration pattern (backward compatible):**

```go
// BEFORE (in cache/cache.go)
var ErrNotFound = errors.New("not found")

// AFTER
import pkgerr "github.com/matzehuels/stacktower/pkg/errors"
var ErrNotFound = pkgerr.ErrNotFound  // Alias for backward compat
```

#### Task 1.2: Consolidate TTL Constants

**Create:** `pkg/config/ttl.go`

```go
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
```

**Files to update:**

| File | Constants to Replace |
|------|---------------------|
| `artifact/artifact.go` | `GraphTTL`, `LayoutTTL`, `RenderTTL` |
| `cache/cache.go` | `GraphTTL`, `RenderTTL` ⚠️ Fix 24h→90d bug |
| `pipeline/service.go` | `GraphTTL`, `LayoutTTL`, `RenderTTL` |
| `session/session.go` | `DefaultTTL`, `DefaultStateTTL` |

#### Task 1.3: Consolidate Hash Utilities

**Create:** `pkg/hash/hash.go`

```go
package hash

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
)

// Bytes computes SHA256 hash of data and returns hex string.
func Bytes(data []byte) string {
    h := sha256.Sum256(data)
    return hex.EncodeToString(h[:])
}

// JSON computes SHA256 hash of JSON-serialized value.
func JSON(v interface{}) string {
    data, _ := json.Marshal(v)
    return Bytes(data)
}

// Short returns first n bytes of hash (for readable cache keys).
func Short(data []byte, n int) string {
    h := sha256.Sum256(data)
    if n > len(h) {
        n = len(h)
    }
    return hex.EncodeToString(h[:n])
}
```

**Files to update:**

| File | Functions to Replace |
|------|---------------------|
| `artifact/local_backend.go` | `Hash()`, `HashJSON()` |
| `cache/cache.go` | `ContentHash()`, `OptionsHash()` |
| `httpcache/cache.go` | `hashKey()` method |

---

### Phase 2: Merge backend into httpcache

**Time Estimate:** 2-3 hours  
**Risk:** Medium

The `backend` package exists only to serve `httpcache`. Merge them into a unified `kv` package.

#### Task 2.1: Reorganize httpcache Package

**Rename:** `pkg/httpcache/` → `pkg/kv/`

**New structure:**
```
pkg/kv/
├── store.go          # Store interface (from backend.Backend)
├── memory.go         # MemoryStore (from backend.Memory)
├── filesystem.go     # FilesystemStore (from backend.Filesystem)
├── redis.go          # RedisStore (from backend.Redis)
├── cache.go          # Cache wrapper (from httpcache.Cache)
├── retry.go          # Retry utilities (from httpcache.retry.go)
└── doc.go
```

**Interface definition:**

```go
// pkg/kv/store.go
package kv

import (
    "context"
    "time"
)

// Store defines the interface for key-value storage with TTL.
type Store interface {
    // Get retrieves a value by key.
    // Returns (value, true, nil) on hit, (nil, false, nil) on miss.
    Get(ctx context.Context, key string) ([]byte, bool, error)

    // Set stores a value with TTL. TTL of 0 means no expiration.
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

    // Delete removes a key. Returns nil if key doesn't exist.
    Delete(ctx context.Context, key string) error

    // Close releases resources.
    Close() error
}
```

#### Task 2.2: Update All Imports

**Search and replace:**

```
github.com/matzehuels/stacktower/pkg/backend → github.com/matzehuels/stacktower/pkg/kv
github.com/matzehuels/stacktower/pkg/httpcache → github.com/matzehuels/stacktower/pkg/kv
```

**Files to update:**
- `pkg/integrations/common.go`
- `pkg/integrations/client.go`
- `pkg/infra/redis.go`
- All files in `pkg/deps/` that use caching

#### Task 2.3: Delete Old Packages

After all imports are updated:
- Delete `pkg/backend/`
- Delete `pkg/httpcache/`

---

### Phase 3: Simplify Artifact Package

**Time Estimate:** 1-2 hours  
**Risk:** Low (documentation only)

#### Task 3.1: Document the Divergent Backends

Currently:
- `LocalBackend` uses `storage.Storage` + a local index file
- `ProdBackend` uses `cache.Cache`

This is intentional but confusing. Add clear documentation.

**Update:** `pkg/artifact/doc.go`

```go
// Package artifact provides caching for pipeline artifacts.
//
// # Backend Implementations
//
// Two backends are provided for different deployment scenarios:
//
// ## LocalBackend (CLI)
//
// For CLI usage. Uses pkg/storage for file storage and a local JSON
// index for TTL tracking. Artifacts are stored in ~/.stacktower/cache/artifacts/.
//
// Example:
//
//     backend, _ := artifact.NewLocalBackend(artifact.LocalBackendConfig{
//         CacheDir: "~/.stacktower/cache",
//     })
//     defer backend.Close()
//     svc := pipeline.NewService(backend)
//
// ## ProdBackend (API)
//
// For API usage. Uses pkg/cache (Redis + MongoDB) for distributed
// caching with automatic TTL expiration handled by Redis.
//
// Example:
//
//     backend := artifact.NewProdBackend(cache)
//     svc := pipeline.NewService(backend)
//
// # Interface
//
// Both implement the Backend interface and are interchangeable
// from the pipeline.Service perspective.
package artifact
```

---

### Phase 4: Clean Up Infra Package

**Time Estimate:** 1-2 hours  
**Risk:** Low (documentation only)

#### Task 4.1: Document the Infra Package

The `infra` package is well-designed—document the pattern.

**Update:** `pkg/infra/doc.go`

```go
// Package infra provides production infrastructure clients.
//
// # Design Philosophy
//
// This package consolidates database connections into unified clients
// that implement multiple interfaces from other packages. This:
//
//   - Reduces connection overhead (one Redis client, multiple uses)
//   - Centralizes configuration (all config from environment)
//   - Simplifies dependency injection (pass one client, get many interfaces)
//
// # Redis Client
//
// The Redis struct wraps a single redis.Client and provides:
//
//   - Queue() → queue.Queue (job queue via Redis Streams)
//   - Sessions() → session.Store (session storage)
//   - OAuthStates() → session.StateStore (OAuth state tokens)
//   - Cache() → cache.LookupCache (fast TTL-based lookups)
//   - Raw() → *redis.Client (for advanced operations)
//
// # MongoDB Client
//
// The Mongo struct wraps a single mongo.Client and provides:
//
//   - Store() → cache.Store (graph and render storage)
//   - Database() → *mongo.Database (for GridFS and custom queries)
//
// # Usage Example
//
//     cfg := infra.Load()
//     
//     redis, err := infra.NewRedis(ctx, cfg.Redis)
//     if err != nil {
//         log.Fatal(err)
//     }
//     defer redis.Close()
//     
//     mongo, err := infra.NewMongo(ctx, cfg.Mongo)
//     if err != nil {
//         log.Fatal(err)
//     }
//     defer mongo.Close()
//     
//     // Use unified clients
//     server := api.New(
//         redis.Queue(),
//         cache.NewCombinedCache(redis.Cache(), mongo.Store()),
//         api.WithSessions(redis.Sessions()),
//         api.WithStates(redis.OAuthStates()),
//         api.WithStorage(storage.NewGridFSStorage(mongo.Database())),
//     )
package infra
```

---

### Phase 5: Update Documentation

**Time Estimate:** 1-2 hours  
**Risk:** Low

#### Task 5.1: Update pkg/doc.go

After all changes, update the main package documentation to reflect the new structure, particularly the changes to `backend` → `kv`.

#### Task 5.2: Create Architecture Decision Record

**Create:** `docs/adr/001-infrastructure-packages.md`

Document:
- Why we have separate packages
- How they relate to each other
- When to use each one
- Historical context for the design decisions

---

## Implementation Order

```
Phase 0 ─────────────────────────────────────────────────────────────────
    │
    └─→ Fix httpclient import bug (BLOCKING - code won't compile!)
         
Phase 1 ─────────────────────────────────────────────────────────────────
    │
    ├─→ 1.1 Create pkg/errors (no deps)
    │
    ├─→ 1.2 Create pkg/config/ttl.go (no deps)  
    │
    └─→ 1.3 Create pkg/hash (no deps)
         │
         └─→ Update all imports (depends on 1.1-1.3)

Phase 2 ─────────────────────────────────────────────────────────────────
    │
    └─→ Merge backend + httpcache → pkg/kv (depends on Phase 1)
         │
         └─→ Update all imports
              │
              └─→ Delete old packages

Phase 3 ─────────────────────────────────────────────────────────────────
    │
    └─→ Document artifact package (can run parallel to Phase 2)

Phase 4 ─────────────────────────────────────────────────────────────────
    │
    └─→ Document infra package (can run parallel to Phase 2-3)

Phase 5 ─────────────────────────────────────────────────────────────────
    │
    └─→ Update all docs (after all code changes)
```

---

## Risk Assessment

| Phase | Risk Level | Description | Mitigation |
|-------|------------|-------------|------------|
| 0 | Low | Simple search/replace | Test with `go build` |
| 1 | Low | Adding new packages, keeping aliases | Backward compatible aliases |
| 2 | Medium | Major refactor, breaking changes | Comprehensive testing, staged rollout |
| 3 | Low | Documentation only | Review for accuracy |
| 4 | Low | Documentation only | Review for accuracy |
| 5 | Low | Documentation only | Review for accuracy |

---

## Testing Strategy

### After Each Phase

```bash
# 1. Compile check
go build ./...

# 2. Run all unit tests
go test ./pkg/...

# 3. Run integration tests
go test -tags integration ./pkg/...

# 4. Run CLI smoke test
./bin/stacktower render python flask --format svg -o test.svg

# 5. Run API smoke test (if applicable)
curl http://localhost:8080/health
```

### Phase-Specific Testing

#### Phase 0
```bash
# Verify the import fix
go build ./pkg/integrations/...
go test ./pkg/integrations/...
```

#### Phase 1
```bash
# Test new packages
go test ./pkg/errors/...
go test ./pkg/config/...
go test ./pkg/hash/...

# Test that existing code still works with aliases
go test ./pkg/cache/...
go test ./pkg/backend/...
```

#### Phase 2
```bash
# Full integration test after merge
go test ./pkg/kv/...
go test ./pkg/deps/...  # Heavy user of caching
```

---

## Time Estimates

| Phase | Estimated Time |
|-------|----------------|
| Phase 0 | 1 hour |
| Phase 1 | 2-3 hours |
| Phase 2 | 2-3 hours |
| Phase 3 | 1-2 hours |
| Phase 4 | 1-2 hours |
| Phase 5 | 1-2 hours |
| **Total** | **8-13 hours** |

---

## Out of Scope

These packages are well-designed and should remain as-is:

| Package | Reason |
|---------|--------|
| `pkg/dag/` | Clean graph implementation |
| `pkg/deps/` | Well-structured language support |
| `pkg/render/` | Clean rendering pipeline |
| `pkg/io/` | Clean import/export |
| `pkg/pipeline/` | Clean orchestration |
| `pkg/queue/` | Clean interface, good implementations |
| `pkg/logging/` | Simple and works |
| `pkg/jobs/` | Job definitions, no duplication |

---

## Appendix: File Inventory

### Files to Create

| File | Phase |
|------|-------|
| `pkg/errors/errors.go` | 1.1 |
| `pkg/config/ttl.go` | 1.2 |
| `pkg/hash/hash.go` | 1.3 |
| `pkg/kv/store.go` | 2.1 |
| `pkg/kv/memory.go` | 2.1 |
| `pkg/kv/filesystem.go` | 2.1 |
| `pkg/kv/redis.go` | 2.1 |
| `pkg/kv/cache.go` | 2.1 |
| `pkg/kv/retry.go` | 2.1 |
| `pkg/kv/doc.go` | 2.1 |
| `docs/adr/001-infrastructure-packages.md` | 5.2 |

### Files to Modify

| File | Phase | Changes |
|------|-------|---------|
| `pkg/integrations/client.go` | 0.1 | Fix import |
| `pkg/integrations/client_test.go` | 0.1 | Fix import |
| `pkg/backend/backend.go` | 1.1 | Import shared errors |
| `pkg/cache/cache.go` | 1.1, 1.2, 1.3 | Import shared errors, TTLs, hash |
| `pkg/storage/storage.go` | 1.1 | Import shared errors |
| `pkg/session/session.go` | 1.1, 1.2 | Import shared errors, TTLs |
| `pkg/integrations/common.go` | 1.1, 2.2 | Import shared errors, update kv |
| `pkg/artifact/artifact.go` | 1.1, 1.2 | Import shared errors, TTLs |
| `pkg/artifact/local_backend.go` | 1.3 | Import shared hash |
| `pkg/pipeline/service.go` | 1.2 | Import shared TTLs |
| `pkg/artifact/doc.go` | 3.1 | Enhanced documentation |
| `pkg/infra/doc.go` | 4.1 | Enhanced documentation |
| `pkg/doc.go` | 5.1 | Update for new structure |

### Files to Delete

| File | Phase | Status |
|------|-------|--------|
| `pkg/backend/` (entire directory) | 2.3 | ✅ Deleted |
| `pkg/httpcache/` (entire directory) | 2.3 | ✅ Deleted |

---

## Changelog

| Date | Author | Changes |
|------|--------|---------|
| 2024-12-24 | Analysis | Initial draft |
| 2024-12-25 | Cleanup | Deleted orphan `pkg/backend/` and `pkg/httpcache/` directories. All phases complete. |

