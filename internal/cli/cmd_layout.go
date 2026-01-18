package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/internal/cli/term"
	"github.com/matzehuels/stacktower/pkg/dto"
)

// NewLayoutCmd creates the layout command for computing visualization layouts.
func NewLayoutCmd() *cobra.Command {
	opts := DefaultLayoutCmdOpts()

	cmd := &cobra.Command{
		Use:   "layout [graph.json]",
		Short: "Compute visualization layout from a dependency graph",
		Long: `Compute visualization layout from a dependency graph.

The layout command takes a graph.json file (produced by 'parse') and computes
the layout for visualization. The output is a layout.json file (same format as
'render -f json') that can be rendered to SVG/PNG/PDF using the 'visualize' command.

Supports both tower (-t tower) and nodelink (-t nodelink) visualization types.

Results are cached locally for faster subsequent runs.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLayout(cmd.Context(), args[0], opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Output, "output", "o", "", "output file (default: input with .layout.json extension)")
	cmd.Flags().StringVarP(&opts.VizType, "type", "t", opts.VizType, "visualization type: tower (default), nodelink")
	cmd.Flags().BoolVar(&opts.Normalize, "normalize", opts.Normalize, "apply graph normalization (adds subdividers, assigns layers)")
	cmd.Flags().Float64Var(&opts.Width, "width", opts.Width, "frame width")
	cmd.Flags().Float64Var(&opts.Height, "height", opts.Height, "frame height")
	cmd.Flags().StringVar(&opts.Style, "style", opts.Style, "visual style: handdrawn (default), simple")
	cmd.Flags().StringVar(&opts.Ordering, "ordering", opts.Ordering, "ordering algorithm: optimal (default), barycentric (tower)")
	cmd.Flags().IntVar(&opts.OrderTimeout, "ordering-timeout", opts.OrderTimeout, "timeout in seconds for optimal search (tower)")
	cmd.Flags().BoolVar(&opts.Randomize, "randomize", opts.Randomize, "randomize block widths (tower)")
	cmd.Flags().BoolVar(&opts.Merge, "merge", opts.Merge, "merge subdivider blocks (default: true, tower)")
	cmd.Flags().BoolVar(&opts.Nebraska, "nebraska", false, "include Nebraska maintainer ranking (tower)")
	cmd.Flags().BoolVar(&opts.NoCache, "no-cache", false, "disable caching")

	return cmd
}

// runLayout loads the graph, computes the layout via pipeline, and writes output.
func runLayout(ctx context.Context, input string, opts LayoutCmdOpts) error {
	logger := loggerFromContext(ctx)

	// Load graph
	g, err := dto.ReadGraphFile(input)
	if err != nil {
		return fmt.Errorf("load graph %s: %w", input, err)
	}

	// Create pipeline runner
	runner, err := NewRunner(opts.NoCache, logger)
	if err != nil {
		return fmt.Errorf("initialize runner: %w", err)
	}
	defer runner.Close()

	// Configure options
	pipeOpts := opts.Options
	pipeOpts.SetLayoutDefaults()
	pipeOpts.SetRenderDefaults()

	// Configure optimal orderer if needed
	if opts.NeedsOptimalOrderer() {
		pipeOpts.Orderer = newOptimalOrderer(ctx, opts.OrderTimeout)
	}

	// Apply normalization if requested (adds subdividers, transitive reduction, etc.)
	// Tower layout will automatically assign layers if not already present
	workGraph := runner.PrepareGraph(g, pipeOpts)

	// Show spinner while computing layout
	spinner := term.NewSpinner(fmt.Sprintf("Computing %s layout...", opts.VizType))
	spinner.Start()

	// Generate layout DTO (unified for both tower and nodelink)
	layoutDTO, cacheHit, err := runner.GenerateLayoutWithCacheInfo(ctx, workGraph, pipeOpts)
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

	// Export unified layout directly to file
	if err := dto.WriteLayoutFile(layoutDTO, outputPath); err != nil {
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
