// Package cli implements the stacktower command-line interface.
package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/pkg/buildinfo"
	"github.com/matzehuels/stacktower/pkg/cache"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// =============================================================================
// Constants
// =============================================================================

const (
	// appName is the application name used for directories and display.
	appName = "stacktower"

	// defaultOrderTimeout is the default timeout for optimal ordering search (seconds).
	defaultOrderTimeout = 60
)

// Log levels exported for use in main.go.
const (
	LogDebug = log.DebugLevel
	LogInfo  = log.InfoLevel
)

// =============================================================================
// CLI - Central CLI State
// =============================================================================

// CLI holds shared state for all commands.
type CLI struct {
	Logger *log.Logger
}

// New creates a new CLI instance with a default logger.
func New(w io.Writer, level log.Level) *CLI {
	return &CLI{
		Logger: log.NewWithOptions(w, log.Options{
			ReportTimestamp: true,
			TimeFormat:      "15:04:05.00",
			Level:           level,
		}),
	}
}

// SetLogLevel updates the logger's level.
func (c *CLI) SetLogLevel(level log.Level) {
	c.Logger.SetLevel(level)
}

// RootCommand creates the root cobra command with all subcommands registered.
func (c *CLI) RootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:          "stacktower",
		Short:        "Stacktower visualizes dependency graphs as towers",
		Long:         `Stacktower is a CLI tool for visualizing complex dependency graphs as tiered tower structures, making it easier to understand layering and flow.`,
		Version:      buildinfo.Version,
		SilenceUsage: true,
	}

	root.SetVersionTemplate(buildinfo.Template())

	// Register all subcommands
	root.AddCommand(c.parseCommand())
	root.AddCommand(c.layoutCommand())
	root.AddCommand(c.visualizeCommand())
	root.AddCommand(c.renderCommand())
	root.AddCommand(c.cacheCommand())
	root.AddCommand(c.pqtreeCommand())
	root.AddCommand(c.githubCommand())
	root.AddCommand(c.completionCommand())

	return root
}

// =============================================================================
// Runner Factory
// =============================================================================

// newRunner creates a pipeline runner for CLI use.
func (c *CLI) newRunner(noCache bool) (*pipeline.Runner, error) {
	cache, err := newCache(noCache)
	if err != nil {
		return nil, err
	}
	return pipeline.NewRunner(cache, nil, c.Logger), nil
}

func newCache(noCache bool) (cache.Cache, error) {
	if noCache {
		return cache.NewNullCache(), nil
	}
	dir, err := cacheDir()
	if err != nil {
		return cache.NewNullCache(), nil
	}
	return cache.NewFileCache(dir)
}

// =============================================================================
// Paths
// =============================================================================

// cacheDir returns the cache directory using XDG standard (~/.cache/stacktower/).
func cacheDir() (string, error) {
	if cacheHome := os.Getenv("XDG_CACHE_HOME"); cacheHome != "" {
		return filepath.Join(cacheHome, appName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", appName), nil
}

// =============================================================================
// Options Helpers
// =============================================================================

// setCLIDefaults applies CLI-specific defaults on top of pipeline defaults.
func setCLIDefaults(opts *pipeline.Options) {
	opts.SetLayoutDefaults()
	opts.SetRenderDefaults()
	// CLI-specific preferences (override pipeline defaults)
	opts.Randomize = true
	opts.Merge = true
	opts.Normalize = true
	opts.Popups = true
}

// parseFormats parses a comma-separated format string into a slice.
func parseFormats(s string) []string {
	if s == "" {
		return []string{pipeline.FormatSVG}
	}
	return strings.Split(s, ",")
}
