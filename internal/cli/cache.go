package cli

import (
	"github.com/matzehuels/stacktower/pkg/cache"
)

// newCLICache creates a cache for CLI use.
// If noCache is true, returns a null cache.
// Otherwise, returns a file-based cache in the user's cache directory.
func newCLICache(noCache bool) (cache.Cache, error) {
	if noCache {
		return cache.NewNullCache(), nil
	}

	dir, err := cacheDir()
	if err != nil {
		// Fall back to null cache if we can't get cache dir
		return cache.NewNullCache(), nil
	}

	return cache.NewFileCache(dir)
}
