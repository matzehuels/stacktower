package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileStore is a file-based session store for CLI applications.
// Sessions are stored as JSON files in a config directory.
type FileStore struct {
	mu      sync.RWMutex
	baseDir string
}

// NewFileStore creates a new file-based session store.
// If baseDir is empty, defaults to ~/.config/stacktower/sessions/
func NewFileStore(baseDir string) (*FileStore, error) {
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		baseDir = filepath.Join(home, ".config", "stacktower", "sessions")
	}
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("create session dir: %w", err)
	}
	return &FileStore{baseDir: baseDir}, nil
}

func (s *FileStore) sessionPath(sessionID string) string {
	return filepath.Join(s.baseDir, sessionID+".json")
}

func (s *FileStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.sessionPath(sessionID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read session file: %w", err)
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}

	if sess.IsExpired() {
		os.Remove(path)
		return nil, nil
	}
	return &sess, nil
}

func (s *FileStore) Set(ctx context.Context, sess *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	path := s.sessionPath(sess.ID)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write session file: %w", err)
	}
	return nil
}

func (s *FileStore) Delete(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.sessionPath(sessionID)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove session file: %w", err)
	}
	return nil
}

func (s *FileStore) Cleanup(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return fmt.Errorf("read session dir: %w", err)
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(s.baseDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var sess Session
		if err := json.Unmarshal(data, &sess); err != nil {
			continue
		}
		if now.After(sess.ExpiresAt) {
			os.Remove(path)
		}
	}
	return nil
}

func (s *FileStore) Close() error { return nil }

// Path returns the base directory for session files.
func (s *FileStore) Path() string {
	return s.baseDir
}

var _ Store = (*FileStore)(nil)

// =============================================================================
// CLI convenience wrapper
// =============================================================================

const defaultCLISessionID = "github"

// CLIStore wraps FileStore for simple CLI token storage.
type CLIStore struct {
	store     *FileStore
	sessionID string
}

// NewCLIStore creates a store for CLI token storage.
func NewCLIStore() (*CLIStore, error) {
	store, err := NewFileStore("")
	if err != nil {
		return nil, err
	}
	return &CLIStore{store: store, sessionID: defaultCLISessionID}, nil
}

// GetSession retrieves the CLI session.
func (c *CLIStore) GetSession(ctx context.Context) (*Session, error) {
	return c.store.Get(ctx, c.sessionID)
}

// SaveSession stores the CLI session.
func (c *CLIStore) SaveSession(ctx context.Context, sess *Session) error {
	sess.ID = c.sessionID
	return c.store.Set(ctx, sess)
}

// DeleteSession removes the CLI session.
func (c *CLIStore) DeleteSession(ctx context.Context) error {
	return c.store.Delete(ctx, c.sessionID)
}

// Path returns the session file path.
func (c *CLIStore) Path() string {
	return c.store.sessionPath(c.sessionID)
}
