package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// =============================================================================
// Hash Utilities (canonical definitions)
// =============================================================================

// Hash computes a SHA256 hash of the given data and returns it as a hex string.
func Hash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// HashJSON computes a hash of a JSON-serializable value.
func HashJSON(v interface{}) string {
	data, _ := json.Marshal(v)
	return Hash(data)
}

// HashKey computes a hash of a cache key string.
// Used for HTTP cache keys to normalize key length.
func HashKey(key string) string {
	return Hash([]byte(key))
}
