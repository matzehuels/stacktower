package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigDir(t *testing.T) {
	dir, err := configDir()
	if err != nil {
		t.Fatalf("configDir() error: %v", err)
	}

	if dir == "" {
		t.Error("configDir() returned empty string")
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".stacktower")
	if dir != expected {
		t.Errorf("configDir() = %q, want %q", dir, expected)
	}
}

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

	// Should be under home directory
	home, _ := os.UserHomeDir()
	if !strings.HasPrefix(dir, home) {
		t.Errorf("cacheDir() = %q, should be under home %q", dir, home)
	}

	// Should end with "stacktower"
	if !strings.HasSuffix(dir, "stacktower") {
		t.Errorf("cacheDir() = %q, should end with 'stacktower'", dir)
	}

	// Should use XDG standard: ~/.cache/stacktower
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

	// Verify the expected structure: $HOME/.cache/stacktower
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".cache", "stacktower")
	if dir != expected {
		t.Errorf("cacheDir() = %q, want %q", dir, expected)
	}
}

func TestCacheDirXDG(t *testing.T) {
	// Test XDG_CACHE_HOME override
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

	expected := filepath.Join(customCache, "stacktower")
	if dir != expected {
		t.Errorf("cacheDir() with XDG_CACHE_HOME = %q, want %q", dir, expected)
	}
}

func TestSessionsDir(t *testing.T) {
	dir, err := sessionsDir()
	if err != nil {
		t.Fatalf("sessionsDir() error: %v", err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".stacktower", "sessions")
	if dir != expected {
		t.Errorf("sessionsDir() = %q, want %q", dir, expected)
	}
}

func TestPathsConsistency(t *testing.T) {
	// Config-related paths should share the same base config directory
	config, _ := configDir()
	sessions, _ := sessionsDir()

	// Sessions should be under config directory
	if !strings.HasPrefix(sessions, config) {
		t.Errorf("sessionsDir() = %q should be under configDir() = %q", sessions, config)
	}

	// Cache is separate (XDG standard), should NOT be under config
	cache, _ := cacheDir()
	home, _ := os.UserHomeDir()

	// Cache should be under home but not under .stacktower config
	if !strings.HasPrefix(cache, home) {
		t.Errorf("cacheDir() = %q should be under home %q", cache, home)
	}
	if strings.HasPrefix(cache, config) {
		t.Errorf("cacheDir() = %q should NOT be under configDir() = %q (cache uses XDG standard)", cache, config)
	}
}
