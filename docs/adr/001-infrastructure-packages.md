# ADR 001: Infrastructure Package Consolidation

## Status

Accepted

## Context

The Stacktower infrastructure packages (`pkg/backend`, `pkg/httpcache`, `pkg/artifact`, `pkg/cache`, `pkg/storage`, `pkg/session`) had grown organically, leading to significant duplication and overlapping abstractions:

1.  **Duplicate Error Sentinels:** `ErrNotFound` and `ErrExpired` were defined in multiple packages.
2.  **Duplicate TTL Constants:** Cache TTLs were defined in multiple places with inconsistencies (e.g., `RenderTTL` being 24h in one place and 90d in another).
3.  **Duplicate Hash Utilities:** SHA256 hashing was implemented multiple times across the codebase.
4.  **Overlapping Abstractions:** `backend.Backend` and `httpcache.Cache` were nearly identical to what a generic K-V store should be.
5.  **Implementation Duplication:** Multiple identical in-memory and filesystem storage implementations existed.

## Decision

We decided to consolidate the infrastructure packages to reduce duplication and provide a clearer architecture:

1.  **Consolidate Constants and Utilities:**
    *   Created `pkg/errors` for shared sentinel errors.
    *   Created `pkg/config` for shared TTL constants.
    *   Created `pkg/hash` for shared hashing utilities.
2.  **Merge K-V Storage:**
    *   Merged `pkg/backend` and `pkg/httpcache` into a new `pkg/kv` package.
    *   `pkg/kv` provides a unified `Store` interface and a `Cache` wrapper for JSON serialization and namespacing.
3.  **Unified Implementations:**
    *   `pkg/kv` now contains the canonical implementations for Memory, Filesystem, and Redis storage.
4.  **Enhanced Documentation:**
    *   Updated `pkg/artifact` and `pkg/infra` documentation to clearly explain their roles and how they use the lower-level `kv`, `cache`, and `storage` packages.

## Consequences

*   **Pros:**
    *   Reduced code duplication.
    *   Single source of truth for TTLs and errors.
    *   Clearer package boundaries and responsibilities.
    *   Easier maintenance and testing of core infrastructure.
*   **Cons:**
    *   Breaking changes for existing imports (mitigated by bulk search and replace).
    *   Small amount of churn in the `integrations` package.

