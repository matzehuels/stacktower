package cli

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// NewRunner creates a pipeline runner for CLI use.
// If noCache is true, caching is disabled.
// Otherwise, a file-based cache is created in the user's cache directory.
func NewRunner(noCache bool, logger *log.Logger) (*pipeline.Runner, error) {
	c, err := newCLICache(noCache)
	if err != nil {
		return nil, fmt.Errorf("initialize cache: %w", err)
	}
	return pipeline.NewRunner(c, nil, logger), nil
}
