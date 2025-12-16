package httputil

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

var ErrExpired = errors.New("cache entry expired")

type Cache struct {
	dir string
	ttl time.Duration
}

func NewCache(dir string, ttl time.Duration) (*Cache, error) {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(home, ".cache", "stacktower")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Cache{dir: dir, ttl: ttl}, nil
}

func (c *Cache) Dir() string        { return c.dir }
func (c *Cache) TTL() time.Duration { return c.ttl }

func (c *Cache) Get(key string, v any) (bool, error) {
	path := c.keyPath(key)
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if c.ttl > 0 && time.Since(info.ModTime()) > c.ttl {
		return false, ErrExpired
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal(data, v)
}

func (c *Cache) Set(key string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(c.keyPath(key), data, 0o644)
}

func (c *Cache) keyPath(key string) string {
	h := sha256.Sum256([]byte(key))
	return filepath.Join(c.dir, hex.EncodeToString(h[:]))
}
