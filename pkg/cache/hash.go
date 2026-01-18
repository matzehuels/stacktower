package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// hashKey generates a cache key by hashing the components.
// The key format is: prefix:hash(parts...)
func hashKey(prefix string, parts ...interface{}) string {
	data, _ := json.Marshal(parts)
	hash := sha256.Sum256(data)
	// Use full SHA-256 hash (64 hex chars / 256 bits) to prevent collisions
	return fmt.Sprintf("%s:%s", prefix, hex.EncodeToString(hash[:]))
}

// Hash computes a SHA-256 hash of the input data.
// Returns the full 64-character hex string.
func Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
