package artifact

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/matzehuels/stacktower/pkg/dag"
	pkgio "github.com/matzehuels/stacktower/pkg/io"
	"github.com/matzehuels/stacktower/pkg/storage"
)

// LocalBackend implements Backend using local filesystem storage.
// This is suitable for CLI usage where artifacts are cached locally.
type LocalBackend struct {
	storage   storage.Storage
	index     *localIndex
	ownsStore bool
}

// LocalBackendConfig configures the local backend.
type LocalBackendConfig struct {
	// CacheDir is the root directory for cache storage.
	// Defaults to ~/.stacktower/cache
	CacheDir string

	// Storage is an optional custom storage backend.
	// If nil, uses filesystem storage at CacheDir/artifacts.
	Storage storage.Storage
}

// NewLocalBackend creates a new local file-based backend.
func NewLocalBackend(cfg LocalBackendConfig) (*LocalBackend, error) {
	if cfg.CacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		cfg.CacheDir = filepath.Join(home, ".stacktower", "cache")
	}

	if cfg.CacheDir[0] == '~' {
		home, _ := os.UserHomeDir()
		cfg.CacheDir = filepath.Join(home, cfg.CacheDir[1:])
	}

	var store storage.Storage
	ownsStore := false
	if cfg.Storage != nil {
		store = cfg.Storage
	} else {
		var err error
		store, err = storage.NewFilesystemStorage(filepath.Join(cfg.CacheDir, "artifacts"))
		if err != nil {
			return nil, fmt.Errorf("create storage: %w", err)
		}
		ownsStore = true
	}

	indexPath := filepath.Join(cfg.CacheDir, "index.json")
	index, err := loadLocalIndex(indexPath)
	if err != nil {
		if ownsStore {
			store.Close()
		}
		return nil, fmt.Errorf("load index: %w", err)
	}

	return &LocalBackend{
		storage:   store,
		index:     index,
		ownsStore: ownsStore,
	}, nil
}

// GetGraph retrieves a cached graph.
func (b *LocalBackend) GetGraph(ctx context.Context, hash string) (*dag.DAG, bool, error) {
	data, hit, err := b.get(ctx, TypeGraph, hash)
	if err != nil || !hit {
		return nil, false, err
	}

	g, err := pkgio.ReadJSON(bytes.NewReader(data))
	if err != nil {
		return nil, false, nil // treat parse errors as cache miss
	}

	return g, true, nil
}

// PutGraph stores a graph.
func (b *LocalBackend) PutGraph(ctx context.Context, hash string, g *dag.DAG, ttl time.Duration) error {
	var buf bytes.Buffer
	if err := pkgio.WriteJSON(g, &buf); err != nil {
		return err
	}
	return b.put(ctx, TypeGraph, hash, buf.Bytes(), ttl)
}

// GetLayout retrieves cached layout data.
func (b *LocalBackend) GetLayout(ctx context.Context, hash string) ([]byte, bool, error) {
	return b.get(ctx, TypeLayout, hash)
}

// PutLayout stores layout data.
func (b *LocalBackend) PutLayout(ctx context.Context, hash string, data []byte, ttl time.Duration) error {
	return b.put(ctx, TypeLayout, hash, data, ttl)
}

// GetRender retrieves a cached render artifact.
func (b *LocalBackend) GetRender(ctx context.Context, hash, format string) ([]byte, bool, error) {
	key := fmt.Sprintf("%s:%s", hash, format)
	return b.get(ctx, TypeRender, key)
}

// PutRender stores a render artifact.
func (b *LocalBackend) PutRender(ctx context.Context, hash, format string, data []byte, ttl time.Duration) error {
	key := fmt.Sprintf("%s:%s", hash, format)
	return b.put(ctx, TypeRender, key, data, ttl)
}

// Close releases resources.
func (b *LocalBackend) Close() error {
	if err := b.index.save(); err != nil {
		return fmt.Errorf("save index: %w", err)
	}
	if b.ownsStore {
		return b.storage.Close()
	}
	return nil
}

// =============================================================================
// Internal helpers
// =============================================================================

func (b *LocalBackend) get(ctx context.Context, artifactType, hash string) ([]byte, bool, error) {
	cacheKey := fmt.Sprintf("%s:%s", artifactType, hash)

	b.index.mu.RLock()
	entry, ok := b.index.entries[cacheKey]
	b.index.mu.RUnlock()

	if !ok || entry.isExpired() {
		return nil, false, nil
	}

	reader, err := b.storage.Retrieve(ctx, entry.StorageKey)
	if err != nil {
		return nil, false, nil // treat storage errors as cache miss
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, false, nil
	}

	return data, true, nil
}

func (b *LocalBackend) put(ctx context.Context, artifactType, hash string, data []byte, ttl time.Duration) error {
	cacheKey := fmt.Sprintf("%s:%s", artifactType, hash)
	storageKey := fmt.Sprintf("%s/%s.bin", artifactType, hash)

	// Store in storage
	_, err := b.storage.Store(ctx, artifactType, fmt.Sprintf("%s.bin", hash), bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("store artifact: %w", err)
	}

	// Update index
	b.index.mu.Lock()
	b.index.entries[cacheKey] = &localIndexEntry{
		StorageKey: storageKey,
		ExpiresAt:  time.Now().Add(ttl),
		CreatedAt:  time.Now(),
		Size:       int64(len(data)),
	}
	b.index.mu.Unlock()

	return nil
}

// =============================================================================
// Local index for TTL tracking
// =============================================================================

type localIndex struct {
	mu      sync.RWMutex
	entries map[string]*localIndexEntry
	path    string
}

type localIndexEntry struct {
	StorageKey string    `json:"storage_key"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
	Size       int64     `json:"size"`
}

func (e *localIndexEntry) isExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

func loadLocalIndex(path string) (*localIndex, error) {
	index := &localIndex{
		entries: make(map[string]*localIndexEntry),
		path:    path,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return nil, err
			}
			return index, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, &index.entries); err != nil {
		return index, nil // corrupted index, start fresh
	}

	// Prune expired entries
	now := time.Now()
	for key, entry := range index.entries {
		if now.After(entry.ExpiresAt) {
			delete(index.entries, key)
		}
	}

	return index, nil
}

func (idx *localIndex) save() error {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	data, err := json.MarshalIndent(idx.entries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(idx.path, data, 0644)
}

// Ensure LocalBackend implements Backend
var _ Backend = (*LocalBackend)(nil)
