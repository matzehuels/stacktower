package cli

import (
	"context"
	"fmt"
	"os"

	charmlog "github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func SetVersion(v, c, d string) {
	version = v
	commit = c
	date = d
}

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
	root.AddCommand(newPQTreeCmd())

	return root.ExecuteContext(context.Background())
}
