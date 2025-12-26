// Package artifact provides unified artifact caching and storage.
//
// This file defines the Backend interface that abstracts different storage
// implementations (file-based for CLI, Redis+MongoDB for production API).

package artifact

import (
	"context"
	"time"

	"github.com/matzehuels/stacktower/pkg/dag"
)

// Backend is the interface for artifact storage backends.
// Implementations provide caching and persistence for pipeline artifacts.
//
// There are two implementations:
//   - LocalBackend: Local file-based storage for CLI (uses local filesystem)
//   - ProdBackend: Production storage for API (uses Redis for lookup + MongoDB for storage)
type Backend interface {
	// GetGraph retrieves a cached graph by its content hash.
	// Returns nil, false if not found or expired.
	GetGraph(ctx context.Context, hash string) (*dag.DAG, bool, error)

	// PutGraph stores a graph with its content hash.
	PutGraph(ctx context.Context, hash string, g *dag.DAG, ttl time.Duration) error

	// GetLayout retrieves cached layout data by hash.
	// Returns nil, false if not found or expired.
	GetLayout(ctx context.Context, hash string) ([]byte, bool, error)

	// PutLayout stores layout data with hash.
	PutLayout(ctx context.Context, hash string, data []byte, ttl time.Duration) error

	// GetRender retrieves cached render artifacts by hash and format.
	// Returns nil, false if not found or expired.
	GetRender(ctx context.Context, hash, format string) ([]byte, bool, error)

	// PutRender stores a render artifact with hash.
	PutRender(ctx context.Context, hash, format string, data []byte, ttl time.Duration) error

	// Close releases any resources held by the backend.
	Close() error
}

// NullBackend is a no-op backend that never caches anything.
// Useful for testing or when caching should be disabled.
type NullBackend struct{}

func (NullBackend) GetGraph(ctx context.Context, hash string) (*dag.DAG, bool, error) {
	return nil, false, nil
}

func (NullBackend) PutGraph(ctx context.Context, hash string, g *dag.DAG, ttl time.Duration) error {
	return nil
}

func (NullBackend) GetLayout(ctx context.Context, hash string) ([]byte, bool, error) {
	return nil, false, nil
}

func (NullBackend) PutLayout(ctx context.Context, hash string, data []byte, ttl time.Duration) error {
	return nil
}

func (NullBackend) GetRender(ctx context.Context, hash, format string) ([]byte, bool, error) {
	return nil, false, nil
}

func (NullBackend) PutRender(ctx context.Context, hash, format string, data []byte, ttl time.Duration) error {
	return nil
}

func (NullBackend) Close() error {
	return nil
}

// Ensure NullBackend implements Backend
var _ Backend = NullBackend{}
