package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/internal/cli/term"
	"github.com/matzehuels/stacktower/pkg/infra"
	"github.com/matzehuels/stacktower/pkg/io"
)

// NewLayoutCmd creates the layout command for computing visualization layouts.
func NewLayoutCmd() *cobra.Command {
	opts := DefaultLayoutCmdOpts()

	cmd := &cobra.Command{
		Use:   "layout [graph.json]",
		Short: "Compute visualization layout from a dependency graph",
		Long: `Compute visualization layout from a dependency graph.

The layout command takes a graph.json file (produced by 'parse') and computes
the positions and dimensions for visualization. The output is a layout.json
file that can be rendered to SVG/PNG/PDF using the 'visualize' command.

Results are cached locally for faster subsequent runs.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLayout(cmd.Context(), args[0], opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Output, "output", "o", "", "output file (default: input with .layout.json extension)")
	cmd.Flags().StringVarP(&opts.VizType, "type", "t", opts.VizType, "visualization type: tower (default), nodelink")
	cmd.Flags().BoolVar(&opts.Normalize, "normalize", opts.Normalize, "apply normalization pipeline")
	cmd.Flags().Float64Var(&opts.Width, "width", opts.Width, "frame width")
	cmd.Flags().Float64Var(&opts.Height, "height", opts.Height, "frame height")
	cmd.Flags().StringVar(&opts.Ordering, "ordering", opts.Ordering, "ordering algorithm: optimal (default), barycentric")
	cmd.Flags().IntVar(&opts.OrderTimeout, "ordering-timeout", opts.OrderTimeout, "timeout in seconds for optimal search")
	cmd.Flags().BoolVar(&opts.Randomize, "randomize", opts.Randomize, "randomize block widths (tower)")
	cmd.Flags().BoolVar(&opts.Merge, "merge", opts.Merge, "merge subdivider blocks (tower)")
	cmd.Flags().BoolVar(&opts.Nebraska, "nebraska", false, "include Nebraska maintainer ranking (tower)")
	cmd.Flags().BoolVar(&opts.NoCache, "no-cache", false, "disable caching")

	return cmd
}

// runLayout loads the graph, computes the layout via pipeline, and writes output.
func runLayout(ctx context.Context, input string, opts LayoutCmdOpts) error {
	logger := infra.LoggerFromContext(ctx)

	// Load graph
	g, err := io.ImportJSON(input)
	if err != nil {
		return fmt.Errorf("load graph %s: %w", input, err)
	}

	// Create pipeline service
	svc, cleanup, err := newCLIPipeline(opts.NoCache)
	if err != nil {
		return fmt.Errorf("initialize pipeline: %w", err)
	}
	defer cleanup()

	// Configure pipeline options (opts embeds pipeline.Options)
	opts.Logger = logger
	if opts.NeedsOptimalOrderer() {
		opts.Orderer = newOptimalOrderer(ctx, opts.OrderTimeout)
	}

	// Show spinner while computing layout
	spinner := term.NewSpinner(fmt.Sprintf("Computing %s layout...", opts.VizType))
	spinner.Start()

	// Compute layout via pipeline
	layoutData, _, cacheHit, err := svc.Layout(ctx, g, opts.Options)
	if err != nil {
		spinner.StopWithError("Layout failed")
		return fmt.Errorf("compute layout: %w", err)
	}
	spinner.Stop()

	// Determine output path
	outputPath := opts.Output
	if outputPath == "" {
		base := strings.TrimSuffix(input, filepath.Ext(input))
		outputPath = base + ".layout.json"
	}

	// Write output
	if err := writeDataToFile(layoutData, outputPath); err != nil {
		return fmt.Errorf("write output %s: %w", outputPath, err)
	}

	// Show success
	summary := term.NewSuccessSummary("Layout complete")
	summary.AddFile(outputPath)
	summary.Print()
	term.PrintStats(g.NodeCount(), g.EdgeCount(), cacheHit)
	term.PrintNewline()
	term.PrintNextStep("Render", "stacktower visualize "+outputPath)

	return nil
}
