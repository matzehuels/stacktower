package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/internal/cli"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			os.Exit(130) // Standard shell convention for SIGINT
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	var verbose bool

	// Create CLI - verbose flag will be handled via PersistentPreRun
	// Create a temporary CLI to build the root command structure
	c := cli.New(os.Stderr, cli.LogInfo)
	root := c.RootCommand()

	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")

	// Recreate CLI with correct log level before command execution
	originalPreRun := root.PersistentPreRunE
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		level := cli.LogInfo
		if verbose {
			level = cli.LogDebug
		}
		// Update the CLI's logger
		c.SetLogLevel(level)

		if originalPreRun != nil {
			return originalPreRun(cmd, args)
		}
		return nil
	}

	return root.ExecuteContext(ctx)
}
