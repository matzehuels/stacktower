package cli

import (
	"os"
	"path/filepath"

	"github.com/matzehuels/stacktower/internal/cli/term"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
	"github.com/matzehuels/stacktower/pkg/pipeline"
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

// cacheDir returns the cache directory (~/.stacktower/cache/).
// This stores HTTP response caches for package registries.
func cacheDir() (string, error) {
	base, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "cache"), nil
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

// =============================================================================
// Pipeline Service
// =============================================================================

// newCLIPipeline creates a pipeline service configured for CLI usage
// with local file-based caching. If noCache is true, caching is disabled.
// Returns the service, a cleanup function, and any error.
func newCLIPipeline(noCache bool) (*pipeline.Service, func(), error) {
	if noCache {
		return pipeline.NewService(nil), func() {}, nil
	}

	dir, err := cacheDir()
	if err != nil {
		term.PrintWarning("Cache unavailable: %v", err)
		return pipeline.NewService(nil), func() {}, nil
	}

	backend, err := storage.NewFileBackend(storage.FileConfig{
		CacheDir: dir,
	})
	if err != nil {
		term.PrintWarning("Cache unavailable: %v", err)
		return pipeline.NewService(nil), func() {}, nil
	}

	cleanup := func() {
		backend.Close()
	}

	return pipeline.NewService(backend), cleanup, nil
}
