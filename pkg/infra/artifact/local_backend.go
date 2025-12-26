package artifact

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	pkgio "github.com/matzehuels/stacktower/pkg/io"
)

// LocalBackend implements Backend using local filesystem storage.
// This is suitable for CLI usage where artifacts are cached locally.
type LocalBackend struct {
	root  string
	index *localIndex
}

// LocalBackendConfig configures the local backend.
type LocalBackendConfig struct {
	// CacheDir is the root directory for cache storage.
	// Defaults to ~/.stacktower/cache
	CacheDir string
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

	// Create artifacts directory
	artifactsDir := filepath.Join(cfg.CacheDir, "artifacts")
	if err := os.MkdirAll(artifactsDir, 0755); err != nil {
		return nil, fmt.Errorf("create artifacts dir: %w", err)
	}

	// Get absolute path for security checks
	absRoot, err := filepath.Abs(artifactsDir)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute path: %w", err)
	}

	indexPath := filepath.Join(cfg.CacheDir, "index.json")
	index, err := loadLocalIndex(indexPath)
	if err != nil {
		return nil, fmt.Errorf("load index: %w", err)
	}

	return &LocalBackend{
		root:  absRoot,
		index: index,
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

// GetHTTP retrieves a cached HTTP response.
func (b *LocalBackend) GetHTTP(ctx context.Context, namespace, key string) ([]byte, bool, error) {
	cacheKey := namespace + HashKey(key)
	return b.get(ctx, TypeHTTP, cacheKey)
}

// SetHTTP stores an HTTP response.
func (b *LocalBackend) SetHTTP(ctx context.Context, namespace, key string, data []byte, ttl time.Duration) error {
	cacheKey := namespace + HashKey(key)
	return b.put(ctx, TypeHTTP, cacheKey, data, ttl)
}

// DeleteHTTP removes a cached HTTP response.
func (b *LocalBackend) DeleteHTTP(ctx context.Context, namespace, key string) error {
	cacheKey := fmt.Sprintf("%s:%s%s", TypeHTTP, namespace, HashKey(key))

	b.index.mu.Lock()
	entry, ok := b.index.entries[cacheKey]
	if ok {
		delete(b.index.entries, cacheKey)
		// Best effort cleanup of the file
		_ = os.Remove(filepath.Join(b.root, entry.StorageKey))
	}
	b.index.mu.Unlock()

	return nil
}

// Close releases resources.
func (b *LocalBackend) Close() error {
	return b.index.save()
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

	data, err := b.readFile(entry.StorageKey)
	if err != nil {
		return nil, false, nil // treat storage errors as cache miss
	}

	return data, true, nil
}

func (b *LocalBackend) put(ctx context.Context, artifactType, hash string, data []byte, ttl time.Duration) error {
	cacheKey := fmt.Sprintf("%s:%s", artifactType, hash)
	storageKey := filepath.Join(artifactType, hash+".bin")

	// Store file
	if err := b.writeFile(storageKey, data); err != nil {
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

// readFile reads a file from the storage directory with path traversal protection.
func (b *LocalBackend) readFile(relativePath string) ([]byte, error) {
	fullPath := filepath.Join(b.root, relativePath)
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}
	if !strings.HasPrefix(absPath, b.root) {
		return nil, fmt.Errorf("path traversal detected: %s", relativePath)
	}
	return os.ReadFile(fullPath)
}

// writeFile writes a file to the storage directory, creating subdirectories as needed.
func (b *LocalBackend) writeFile(relativePath string, data []byte) error {
	fullPath := filepath.Join(b.root, relativePath)
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	if !strings.HasPrefix(absPath, b.root) {
		return fmt.Errorf("path traversal detected: %s", relativePath)
	}

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	return os.WriteFile(fullPath, data, 0644)
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
