package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// Bytes computes SHA256 hash of data and returns hex string.
func Bytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// JSON computes SHA256 hash of JSON-serialized value.
func JSON(v interface{}) string {
	data, _ := json.Marshal(v)
	return Bytes(data)
}

// Short returns first n characters of hash (for readable cache keys).
func Short(data []byte, n int) string {
	h := sha256.Sum256(data)
	s := hex.EncodeToString(h[:])
	if n > len(s) {
		n = len(s)
	}
	return s[:n]
}
