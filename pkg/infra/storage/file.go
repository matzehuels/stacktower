package storage

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

// FileBackend implements Backend using local filesystem storage.
// This is the recommended backend for CLI usage where artifacts are cached locally.
//
// Storage layout:
//
//	~/.stacktower/cache/
//	├── index.json          # TTL index for all cached items
//	└── artifacts/          # Actual cached files
//	    ├── graph/          # Parsed dependency graphs
//	    ├── layout/         # Computed layouts
//	    ├── render/         # Rendered artifacts (SVG, PNG, PDF)
//	    └── http/           # HTTP response cache
type FileBackend struct {
	root  string     // Absolute path to artifacts directory
	index *fileIndex // TTL index
}

// FileConfig configures the file backend.
type FileConfig struct {
	// CacheDir is the root directory for cache storage.
	// Defaults to ~/.stacktower/cache
	CacheDir string
}

// NewFileBackend creates a new file-based backend for CLI use.
func NewFileBackend(cfg FileConfig) (*FileBackend, error) {
	if cfg.CacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		cfg.CacheDir = filepath.Join(home, ".stacktower", "cache")
	}

	// Expand ~ if present
	if strings.HasPrefix(cfg.CacheDir, "~") {
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

	// Load or create index
	indexPath := filepath.Join(cfg.CacheDir, "index.json")
	index, err := loadFileIndex(indexPath)
	if err != nil {
		return nil, fmt.Errorf("load index: %w", err)
	}

	return &FileBackend{
		root:  absRoot,
		index: index,
	}, nil
}

// =============================================================================
// Backend interface implementation
// =============================================================================

func (b *FileBackend) GetGraph(ctx context.Context, hash string) (*dag.DAG, bool, error) {
	data, hit, err := b.get(ctx, "graph", hash)
	if err != nil || !hit {
		return nil, false, err
	}

	g, err := pkgio.ReadJSON(bytes.NewReader(data))
	if err != nil {
		return nil, false, nil // treat parse errors as cache miss
	}

	return g, true, nil
}

func (b *FileBackend) PutGraph(ctx context.Context, hash string, g *dag.DAG, ttl time.Duration) error {
	var buf bytes.Buffer
	if err := pkgio.WriteJSON(g, &buf); err != nil {
		return err
	}
	return b.put(ctx, "graph", hash, buf.Bytes(), ttl)
}

func (b *FileBackend) GetGraphScoped(ctx context.Context, hash string, userID string) (*dag.DAG, bool, error) {
	// FileBackend is single-user (CLI), so no scoping needed
	return b.GetGraph(ctx, hash)
}

func (b *FileBackend) PutGraphScoped(ctx context.Context, hash string, g *dag.DAG, ttl time.Duration, scope Scope, userID string, meta GraphMeta) error {
	// FileBackend is single-user (CLI), so ignore scope/user - just store the graph
	return b.PutGraph(ctx, hash, g, ttl)
}

func (b *FileBackend) GetLayout(ctx context.Context, hash string) ([]byte, bool, error) {
	return b.get(ctx, "layout", hash)
}

func (b *FileBackend) PutLayout(ctx context.Context, hash string, data []byte, ttl time.Duration) error {
	return b.put(ctx, "layout", hash, data, ttl)
}

func (b *FileBackend) GetRender(ctx context.Context, hash, format string) ([]byte, bool, error) {
	key := fmt.Sprintf("%s:%s", hash, format)
	return b.get(ctx, "render", key)
}

func (b *FileBackend) PutRender(ctx context.Context, hash, format string, data []byte, ttl time.Duration) error {
	key := fmt.Sprintf("%s:%s", hash, format)
	return b.put(ctx, "render", key, data, ttl)
}

func (b *FileBackend) GetHTTP(ctx context.Context, namespace, key string) ([]byte, bool, error) {
	cacheKey := namespace + HashKey(key)
	return b.get(ctx, "http", cacheKey)
}

func (b *FileBackend) SetHTTP(ctx context.Context, namespace, key string, data []byte, ttl time.Duration) error {
	cacheKey := namespace + HashKey(key)
	return b.put(ctx, "http", cacheKey, data, ttl)
}

func (b *FileBackend) DeleteHTTP(ctx context.Context, namespace, key string) error {
	cacheKey := fmt.Sprintf("http:%s%s", namespace, HashKey(key))

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

func (b *FileBackend) Close() error {
	return b.index.save()
}

// =============================================================================
// Internal helpers
// =============================================================================

func (b *FileBackend) get(ctx context.Context, artifactType, hash string) ([]byte, bool, error) {
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

func (b *FileBackend) put(ctx context.Context, artifactType, hash string, data []byte, ttl time.Duration) error {
	cacheKey := fmt.Sprintf("%s:%s", artifactType, hash)
	storageKey := filepath.Join(artifactType, hash+".bin")

	// Store file
	if err := b.writeFile(storageKey, data); err != nil {
		return fmt.Errorf("store artifact: %w", err)
	}

	// Update index
	b.index.mu.Lock()
	b.index.entries[cacheKey] = &fileIndexEntry{
		StorageKey: storageKey,
		ExpiresAt:  time.Now().Add(ttl),
		CreatedAt:  time.Now(),
		Size:       int64(len(data)),
	}
	b.index.mu.Unlock()

	return nil
}

// readFile reads a file from the storage directory with path traversal protection.
func (b *FileBackend) readFile(relativePath string) ([]byte, error) {
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
func (b *FileBackend) writeFile(relativePath string, data []byte) error {
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
// File index for TTL tracking
// =============================================================================

type fileIndex struct {
	mu      sync.RWMutex
	entries map[string]*fileIndexEntry
	path    string
}

type fileIndexEntry struct {
	StorageKey string    `json:"storage_key"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
	Size       int64     `json:"size"`
}

func (e *fileIndexEntry) isExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

func loadFileIndex(path string) (*fileIndex, error) {
	index := &fileIndex{
		entries: make(map[string]*fileIndexEntry),
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

func (idx *fileIndex) save() error {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	data, err := json.MarshalIndent(idx.entries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(idx.path, data, 0644)
}

// Ensure FileBackend implements Backend
var _ Backend = (*FileBackend)(nil)
