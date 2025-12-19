package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

	// Should end with "stacktower"
	if !strings.HasSuffix(dir, "stacktower") {
		t.Errorf("cacheDir() = %q, should end with 'stacktower'", dir)
	}

	// Should contain ".cache" in path
	if !strings.Contains(dir, ".cache") {
		t.Errorf("cacheDir() = %q, should contain '.cache'", dir)
	}
}

func TestCacheDirStructure(t *testing.T) {
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
