package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCacheDir(t *testing.T) {
	// Clear XDG_CACHE_HOME to test default behavior
	oldXdg := os.Getenv("XDG_CACHE_HOME")
	os.Unsetenv("XDG_CACHE_HOME")
	defer func() {
		if oldXdg != "" {
			os.Setenv("XDG_CACHE_HOME", oldXdg)
		}
	}()

	dir, err := cacheDir()
	if err != nil {
		t.Fatalf("cacheDir() error: %v", err)
	}

	if dir == "" {
		t.Error("cacheDir() returned empty string")
	}

	home, _ := os.UserHomeDir()
	if !strings.HasPrefix(dir, home) {
		t.Errorf("cacheDir() = %q, should be under home %q", dir, home)
	}

	if !strings.HasSuffix(dir, appName) {
		t.Errorf("cacheDir() = %q, should end with %q", dir, appName)
	}

	if !strings.Contains(dir, ".cache") {
		t.Errorf("cacheDir() = %q, should contain '.cache'", dir)
	}
}

func TestCacheDirStructure(t *testing.T) {
	// Clear XDG_CACHE_HOME to test default behavior
	oldXdg := os.Getenv("XDG_CACHE_HOME")
	os.Unsetenv("XDG_CACHE_HOME")
	defer func() {
		if oldXdg != "" {
			os.Setenv("XDG_CACHE_HOME", oldXdg)
		}
	}()

	dir, err := cacheDir()
	if err != nil {
		t.Fatalf("cacheDir() error: %v", err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".cache", appName)
	if dir != expected {
		t.Errorf("cacheDir() = %q, want %q", dir, expected)
	}
}

func TestCacheDirXDG(t *testing.T) {
	customCache := "/tmp/custom-cache"
	oldXdg := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", customCache)
	defer func() {
		if oldXdg != "" {
			os.Setenv("XDG_CACHE_HOME", oldXdg)
		} else {
			os.Unsetenv("XDG_CACHE_HOME")
		}
	}()

	dir, err := cacheDir()
	if err != nil {
		t.Fatalf("cacheDir() error: %v", err)
	}

	expected := filepath.Join(customCache, appName)
	if dir != expected {
		t.Errorf("cacheDir() with XDG_CACHE_HOME = %q, want %q", dir, expected)
	}
}
