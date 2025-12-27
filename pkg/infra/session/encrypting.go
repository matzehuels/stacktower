package session

import (
	"context"
	"fmt"
)

// EncryptingStore wraps a Store and encrypts access tokens before storage.
// This provides defense-in-depth: even if the storage backend is compromised,
// access tokens remain protected.
//
// Usage:
//
//	key, _ := session.ParseKey(os.Getenv("STACKTOWER_SESSION_KEY"))
//	encryptor, _ := session.NewTokenEncryptor(key)
//	store := session.NewEncryptingStore(redisStore, encryptor)
type EncryptingStore struct {
	inner     Store
	encryptor *TokenEncryptor
}

// NewEncryptingStore creates a new encrypting store wrapper.
// If encryptor is nil, tokens are stored in plaintext (not recommended for production).
func NewEncryptingStore(inner Store, encryptor *TokenEncryptor) *EncryptingStore {
	return &EncryptingStore{
		inner:     inner,
		encryptor: encryptor,
	}
}

// Get retrieves and decrypts a session.
func (s *EncryptingStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	sess, err := s.inner.Get(ctx, sessionID)
	if err != nil || sess == nil {
		return sess, err
	}

	// Decrypt the access token if encryption is enabled
	if s.encryptor != nil && sess.AccessToken != "" {
		decrypted, err := s.encryptor.Decrypt(sess.AccessToken)
		if err != nil {
			// Log but don't expose crypto details to caller
			return nil, fmt.Errorf("session corrupted or key changed")
		}
		sess.AccessToken = decrypted
	}

	return sess, nil
}

// Set encrypts and stores a session.
func (s *EncryptingStore) Set(ctx context.Context, sess *Session) error {
	// Create a copy to avoid modifying the original
	toStore := &Session{
		ID:          sess.ID,
		AccessToken: sess.AccessToken,
		User:        sess.User,
		ExpiresAt:   sess.ExpiresAt,
		CreatedAt:   sess.CreatedAt,
	}

	// Encrypt the access token if encryption is enabled
	if s.encryptor != nil && toStore.AccessToken != "" {
		encrypted, err := s.encryptor.Encrypt(toStore.AccessToken)
		if err != nil {
			return fmt.Errorf("encrypt token: %w", err)
		}
		toStore.AccessToken = encrypted
	}

	return s.inner.Set(ctx, toStore)
}

// Delete removes a session.
func (s *EncryptingStore) Delete(ctx context.Context, sessionID string) error {
	return s.inner.Delete(ctx, sessionID)
}

// Cleanup removes expired sessions.
func (s *EncryptingStore) Cleanup(ctx context.Context) error {
	return s.inner.Cleanup(ctx)
}

var _ Store = (*EncryptingStore)(nil)
