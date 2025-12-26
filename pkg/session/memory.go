package session

import (
	"context"
	"sync"
	"time"
)

// MemoryStore is an in-memory session store.
// Use for development and testing only.
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewMemoryStore creates a new in-memory session store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{sessions: make(map[string]*Session)}
}

func (s *MemoryStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[sessionID]
	if !ok || sess.IsExpired() {
		return nil, nil
	}
	return sess, nil
}

func (s *MemoryStore) Set(ctx context.Context, sess *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess.ID] = sess
	return nil
}

func (s *MemoryStore) Delete(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
	return nil
}

func (s *MemoryStore) Cleanup(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for id, sess := range s.sessions {
		if now.After(sess.ExpiresAt) {
			delete(s.sessions, id)
		}
	}
	return nil
}

func (s *MemoryStore) Close() error { return nil }

var _ Store = (*MemoryStore)(nil)

// MemoryStateStore is an in-memory OAuth state store.
type MemoryStateStore struct {
	mu     sync.Mutex
	states map[string]time.Time
}

// NewMemoryStateStore creates a new in-memory state store.
func NewMemoryStateStore() *MemoryStateStore {
	return &MemoryStateStore{states: make(map[string]time.Time)}
}

func (s *MemoryStateStore) Generate(ctx context.Context, ttl time.Duration) (string, error) {
	state, err := GenerateState()
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	s.states[state] = time.Now().Add(ttl)
	s.mu.Unlock()
	return state, nil
}

func (s *MemoryStateStore) Validate(ctx context.Context, state string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	expiry, ok := s.states[state]
	if !ok {
		return false, nil
	}
	delete(s.states, state)
	return time.Now().Before(expiry), nil
}

func (s *MemoryStateStore) Cleanup(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for state, expiry := range s.states {
		if now.After(expiry) {
			delete(s.states, state)
		}
	}
	return nil
}

var _ StateStore = (*MemoryStateStore)(nil)
