package session

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// Encryption errors.
var (
	// ErrInvalidKey is returned when the encryption key is invalid.
	ErrInvalidKey = errors.New("encryption key must be 32 bytes (256 bits)")

	// ErrDecryptionFailed is returned when decryption fails (wrong key, corrupted data, etc.)
	ErrDecryptionFailed = errors.New("decryption failed")
)

// TokenEncryptor handles encryption and decryption of access tokens.
// Uses AES-256-GCM for authenticated encryption.
type TokenEncryptor struct {
	gcm cipher.AEAD
}

// NewTokenEncryptor creates a new encryptor with the given 32-byte key.
// The key should be loaded from a secure source (environment variable, secret manager).
func NewTokenEncryptor(key []byte) (*TokenEncryptor, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	return &TokenEncryptor{gcm: gcm}, nil
}

// Encrypt encrypts plaintext and returns base64-encoded ciphertext.
// The nonce is prepended to the ciphertext.
func (e *TokenEncryptor) Encrypt(plaintext string) (string, error) {
	// Generate a random nonce
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	// Encrypt and prepend nonce
	ciphertext := e.gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Return as base64 for safe storage in JSON
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded ciphertext and returns plaintext.
func (e *TokenEncryptor) Decrypt(encoded string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	nonceSize := e.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", ErrDecryptionFailed
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := e.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}

// GenerateKey generates a random 32-byte encryption key.
// Use this to generate a key for STACKTOWER_SESSION_KEY.
// The returned string is base64-encoded for easy storage in environment variables.
func GenerateKey() (string, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// ParseKey decodes a base64-encoded key.
func ParseKey(encoded string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode key: %w", err)
	}
	if len(key) != 32 {
		return nil, ErrInvalidKey
	}
	return key, nil
}
