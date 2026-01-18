package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/internal/cli"
	"github.com/matzehuels/stacktower/pkg/buildinfo"
)

func main() {
	if err := execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func execute() error {
	var verbose bool

	root := &cobra.Command{
		Use:          "stacktower",
		Short:        "Stacktower visualizes dependency graphs as towers",
		Long:         `Stacktower is a CLI tool for visualizing complex dependency graphs as tiered tower structures, making it easier to understand layering and flow.`,
		Version:      buildinfo.Version,
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			level := cli.LogInfo
			if verbose {
				level = cli.LogDebug
			}
			ctx := cli.WithLogger(cmd.Context(), cli.NewLogger(os.Stderr, level))
			cmd.SetContext(ctx)
		},
	}

	root.SetVersionTemplate(buildinfo.Template())
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")

	// Register all CLI commands from the centralized registry
	for _, cmd := range cli.Commands() {
		root.AddCommand(cmd)
	}

	return root.ExecuteContext(context.Background())
}
