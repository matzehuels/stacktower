package httputil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCache_GetSet(t *testing.T) {
	c := &Cache{Dir: t.TempDir(), TTL: time.Hour}

	tests := []struct {
		name  string
		key   string
		value any
	}{
		{"simple", "key1", map[string]string{"foo": "bar"}},
		{"string", "key2", "test"},
		{"nested", "key3", map[string]any{"a": map[string]int{"b": 1}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := c.Set(tt.key, tt.value); err != nil {
				t.Fatalf("Set() failed: %v", err)
			}

			var result any
			switch tt.value.(type) {
			case map[string]string:
				result = &map[string]string{}
			case string:
				result = new(string)
			case map[string]any:
				result = &map[string]any{}
			}

			ok, err := c.Get(tt.key, result)
			if err != nil {
				t.Fatalf("Get() failed: %v", err)
			}
			if !ok {
				t.Fatal("Get() returned false for existing key")
			}
		})
	}
}

func TestCache_Miss(t *testing.T) {
	c := &Cache{Dir: t.TempDir(), TTL: time.Hour}
	var result string
	ok, err := c.Get("missing", &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("Get() returned true for missing key")
	}
}

func TestCache_Expiration(t *testing.T) {
	c := &Cache{Dir: t.TempDir(), TTL: 10 * time.Millisecond}

	if err := c.Set("key", "value"); err != nil {
		t.Fatalf("Set() failed: %v", err)
	}

	var res string
	ok, err := c.Get("key", &res)
	if err != nil || !ok {
		t.Fatalf("Get() = %v, %v; want true, nil", ok, err)
	}

	time.Sleep(20 * time.Millisecond)

	ok, err = c.Get("key", &res)
	if !errors.Is(err, ErrExpired) {
		t.Errorf("got error %v, want ErrExpired", err)
	}
	if ok {
		t.Error("Get() returned true for expired key")
	}
}

func TestCache_KeyStability(t *testing.T) {
	c := &Cache{Dir: t.TempDir(), TTL: time.Hour}
	p1 := c.path("test")
	p2 := c.path("test")
	if p1 != p2 {
		t.Error("path should be deterministic")
	}
	p3 := c.path("other")
	if p1 == p3 {
		t.Error("different keys should produce different paths")
	}
}

func TestNewCache_DefaultDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	c, err := NewCache("", time.Hour)
	if err != nil {
		t.Fatalf("NewCache() failed: %v", err)
	}

	want := filepath.Join(home, ".cache", "stacktower")
	if c.Dir != want {
		t.Errorf("got Dir = %s, want %s", c.Dir, want)
	}
	if c.TTL != time.Hour {
		t.Errorf("got TTL = %v, want 1h", c.TTL)
	}
	if _, err := os.Stat(c.Dir); err != nil {
		t.Errorf("directory not created: %v", err)
	}
}
