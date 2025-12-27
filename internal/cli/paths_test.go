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

	// Should end with "cache" (under .stacktower)
	if !strings.HasSuffix(dir, "cache") {
		t.Errorf("cacheDir() = %q, should end with 'cache'", dir)
	}

	// Should contain ".stacktower" in path
	if !strings.Contains(dir, ".stacktower") {
		t.Errorf("cacheDir() = %q, should contain '.stacktower'", dir)
	}
}

func TestCacheDirStructure(t *testing.T) {
	dir, err := cacheDir()
	if err != nil {
		t.Fatalf("cacheDir() error: %v", err)
	}

	// Verify the expected structure: $HOME/.stacktower/cache
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".stacktower", "cache")
	if dir != expected {
		t.Errorf("cacheDir() = %q, want %q", dir, expected)
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
	// All paths should share the same base config directory
	config, _ := configDir()
	cache, _ := cacheDir()
	sessions, _ := sessionsDir()

	if !strings.HasPrefix(cache, config) {
		t.Errorf("cacheDir() = %q should be under configDir() = %q", cache, config)
	}
	if !strings.HasPrefix(sessions, config) {
		t.Errorf("sessionsDir() = %q should be under configDir() = %q", sessions, config)
	}
}
