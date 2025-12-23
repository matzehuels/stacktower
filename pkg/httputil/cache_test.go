package httputil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCache_GetSet(t *testing.T) {
	c, _ := NewCache(t.TempDir(), time.Hour)

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
	c, _ := NewCache(t.TempDir(), time.Hour)
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
	c, _ := NewCache(t.TempDir(), 10*time.Millisecond)

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
	c, _ := NewCache(t.TempDir(), time.Hour)
	p1 := c.keyPath("test")
	p2 := c.keyPath("test")
	if p1 != p2 {
		t.Error("path should be deterministic")
	}
	p3 := c.keyPath("other")
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
	if c.Dir() != want {
		t.Errorf("got Dir = %s, want %s", c.Dir(), want)
	}
	if c.TTL() != time.Hour {
		t.Errorf("got TTL = %v, want 1h", c.TTL())
	}
	if _, err := os.Stat(c.Dir()); err != nil {
		t.Errorf("directory not created: %v", err)
	}
}

func TestCache_Namespace(t *testing.T) {
	c, _ := NewCache(t.TempDir(), time.Hour)

	t.Run("basicNamespacing", func(t *testing.T) {
		pypi := c.Namespace("pypi:")
		npm := c.Namespace("npm:")

		// Set values in different namespaces
		if err := pypi.Set("requests", "pypi-data"); err != nil {
			t.Fatalf("pypi.Set() failed: %v", err)
		}
		if err := npm.Set("requests", "npm-data"); err != nil {
			t.Fatalf("npm.Set() failed: %v", err)
		}

		// Retrieve from namespaced caches
		var pypiVal, npmVal string
		ok, err := pypi.Get("requests", &pypiVal)
		if !ok || err != nil {
			t.Fatalf("pypi.Get() = %v, %v; want true, nil", ok, err)
		}
		ok, err = npm.Get("requests", &npmVal)
		if !ok || err != nil {
			t.Fatalf("npm.Get() = %v, %v; want true, nil", ok, err)
		}

		if pypiVal != "pypi-data" {
			t.Errorf("got pypi value %q, want %q", pypiVal, "pypi-data")
		}
		if npmVal != "npm-data" {
			t.Errorf("got npm value %q, want %q", npmVal, "npm-data")
		}

		// Values should not cross-contaminate
		_, _ = pypi.Get("requests", &npmVal)
		if npmVal != "pypi-data" {
			t.Error("namespace isolation violated")
		}
	})

	t.Run("chainedNamespacing", func(t *testing.T) {
		python := c.Namespace("python:")
		pypi := python.Namespace("pypi:")

		if err := pypi.Set("test", "value"); err != nil {
			t.Fatalf("Set() failed: %v", err)
		}

		var result string
		ok, err := pypi.Get("test", &result)
		if !ok || err != nil || result != "value" {
			t.Errorf("Get() = %v, %v, %q; want true, nil, %q", ok, err, result, "value")
		}

		// Should not be accessible without full prefix
		found, _ := python.Get("test", &result)
		if found {
			t.Error("value accessible without full namespace chain")
		}
	})

	t.Run("emptyPrefix", func(t *testing.T) {
		ns := c.Namespace("")
		if err := ns.Set("key", "value"); err != nil {
			t.Fatalf("Set() failed: %v", err)
		}

		var result string
		ok, err := ns.Get("key", &result)
		if !ok || err != nil || result != "value" {
			t.Errorf("Get() = %v, %v, %q; want true, nil, %q", ok, err, result, "value")
		}

		// Should be same as parent cache
		ok, err = c.Get("key", &result)
		if !ok || err != nil || result != "value" {
			t.Error("empty namespace should behave like parent")
		}
	})

	t.Run("preservesDirAndTTL", func(t *testing.T) {
		ns := c.Namespace("test:")
		if ns.Dir() != c.Dir() {
			t.Errorf("Dir() = %s, want %s", ns.Dir(), c.Dir())
		}
		if ns.TTL() != c.TTL() {
			t.Errorf("TTL() = %v, want %v", ns.TTL(), c.TTL())
		}
	})
}
