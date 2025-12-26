package cli

import (
	"os"
	"path/filepath"

	"github.com/matzehuels/stacktower/pkg/infra/artifact"
	"github.com/matzehuels/stacktower/pkg/infra/common"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// artifactCacheDir returns the hardcoded cache directory for artifacts.
func artifactCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "stacktower", "cache")
	}
	return filepath.Join(home, ".stacktower", "cache")
}

// newPipelineService creates a pipeline service with optional caching.
// If noCache is true, returns a service without caching.
func newPipelineService(noCache bool, logger *common.Logger) (*pipeline.Service, func(), error) {
	if noCache {
		return pipeline.NewService(nil), func() {}, nil
	}

	backend, err := artifact.NewLocalBackend(artifact.LocalBackendConfig{
		CacheDir: artifactCacheDir(),
	})
	if err != nil {
		// Fall back to no caching on error
		logger.Warn("cache initialization failed, proceeding without cache", "error", err)
		return pipeline.NewService(nil), func() {}, nil
	}

	cleanup := func() {
		backend.Close()
	}

	return pipeline.NewService(backend), cleanup, nil
}

// formatCacheStatus returns a human-readable cache status string.
func formatCacheStatus(cacheHit bool) string {
	if cacheHit {
		return "cached"
	}
	return "computed"
}
