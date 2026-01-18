package cli

import (
	"os"
	"path/filepath"
)

// =============================================================================
// Path Helpers
// Centralized path management for CLI-specific directories.
// =============================================================================

const (
	// appDirName is the application directory name under user home.
	appDirName = ".stacktower"
)

// configDir returns the base config directory (~/.stacktower/).
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, appDirName), nil
}

// cacheDir returns the cache directory using XDG standard (~/.cache/stacktower/).
// This stores HTTP response caches for package registries and pipeline artifacts.
// Uses XDG_CACHE_HOME if set, otherwise defaults to ~/.cache/stacktower.
func cacheDir() (string, error) {
	// Check XDG_CACHE_HOME environment variable first
	if cacheHome := os.Getenv("XDG_CACHE_HOME"); cacheHome != "" {
		return filepath.Join(cacheHome, "stacktower"), nil
	}

	// Fall back to ~/.cache/stacktower
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "stacktower"), nil
}

// sessionsDir returns the sessions directory (~/.stacktower/sessions/).
// This stores OAuth session tokens.
func sessionsDir() (string, error) {
	base, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "sessions"), nil
}
