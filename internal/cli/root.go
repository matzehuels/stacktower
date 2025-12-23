package cli

import (
	"context"
	"fmt"
	"os"

	charmlog "github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	version string // semantic version (e.g., "v1.2.3")
	commit  string // git commit SHA
	date    string // build timestamp
)

// SetVersion sets the version information displayed by --version.
// This is typically called by the main package during initialization with values
// injected via ldflags at build time.
//
// Parameters:
//   - v: semantic version string (e.g., "v1.2.3")
//   - c: git commit SHA (short or long form)
//   - d: build timestamp (e.g., "2025-12-20T14:32:01Z")
func SetVersion(v, c, d string) {
	version = v
	commit = c
	date = d
}

// Execute runs the stacktower CLI and returns an error if any command fails.
// This is the main entry point for the CLI application.
//
// The function sets up the root command with all subcommands (parse, render, cache, pqtree),
// configures logging based on the --verbose flag, and executes the command tree.
//
// Logging:
//   - Default: info level (logs to stderr)
//   - With --verbose (-v): debug level
//
// The logger is attached to the context and accessible to all commands via loggerFromContext.
//
// Example:
//
//	func main() {
//	    cli.SetVersion("v1.0.0", "abc123", "2025-12-20")
//	    if err := cli.Execute(); err != nil {
//	        os.Exit(1)
//	    }
//	}
func Execute() error {
	var verbose bool

	root := &cobra.Command{
		Use:          "stacktower",
		Short:        "StackTower visualizes dependency graphs as towers",
		Long:         `StackTower is a CLI tool for visualizing complex dependency graphs as tiered tower structures, making it easier to understand layering and flow.`,
		Version:      version,
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			level := charmlog.InfoLevel
			if verbose {
				level = charmlog.DebugLevel
			}
			ctx := withLogger(cmd.Context(), newLogger(os.Stderr, level))
			cmd.SetContext(ctx)
		},
	}

	root.SetVersionTemplate(fmt.Sprintf("stacktower %s\ncommit: %s\nbuilt: %s\n", version, commit, date))
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")

	root.AddCommand(newParseCmd())
	root.AddCommand(newRenderCmd())
	root.AddCommand(newCacheCmd())
	root.AddCommand(newPQTreeCmd())

	return root.ExecuteContext(context.Background())
}
