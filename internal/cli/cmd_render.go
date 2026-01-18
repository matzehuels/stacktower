package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/internal/cli/term"
	"github.com/matzehuels/stacktower/pkg/dto"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// NewRenderCmd creates the render command for generating visualizations.
func NewRenderCmd() *cobra.Command {
	var formatsStr string
	opts := DefaultRenderCmdOpts()

	cmd := &cobra.Command{
		Use:   "render [graph.json]",
		Short: "Render a dependency graph to SVG/PNG/PDF (shortcut for layout + visualize)",
		Long: `Render a dependency graph to visual output.

This command is a shortcut that combines 'layout' and 'visualize' in one step.
It takes a graph.json file (produced by 'parse') and outputs SVG, PNG, or PDF.

Results are cached locally for faster subsequent runs.

If you want to save the intermediate layout, use 'layout' followed by 'visualize'.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Formats = parseFormats(formatsStr)
			if err := pipeline.ValidateFormats(opts.Formats); err != nil {
				return err
			}
			if err := pipeline.ValidateStyle(opts.Style); err != nil {
				return err
			}
			return runRender(cmd.Context(), args[0], opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Output, "output", "o", "", "output file (single format) or base path (multiple)")
	cmd.Flags().StringVarP(&opts.VizType, "type", "t", opts.VizType, "visualization type: tower (default), nodelink")
	cmd.Flags().StringVarP(&formatsStr, "format", "f", "", "output format(s): svg (default), pdf, png (comma-separated)")
	cmd.Flags().BoolVar(&opts.Normalize, "normalize", opts.Normalize, "apply graph normalization (adds subdividers, assigns layers)")
	cmd.Flags().Float64Var(&opts.Width, "width", opts.Width, "frame width")
	cmd.Flags().Float64Var(&opts.Height, "height", opts.Height, "frame height")
	cmd.Flags().BoolVar(&opts.ShowEdges, "edges", false, "show dependency edges (tower)")
	cmd.Flags().StringVar(&opts.Style, "style", opts.Style, "visual style: handdrawn (default), simple (tower)")
	cmd.Flags().StringVar(&opts.Ordering, "ordering", opts.Ordering, "ordering algorithm: optimal (default), barycentric")
	cmd.Flags().IntVar(&opts.OrderTimeout, "ordering-timeout", opts.OrderTimeout, "timeout in seconds for optimal search")
	cmd.Flags().BoolVar(&opts.Randomize, "randomize", opts.Randomize, "randomize block widths")
	cmd.Flags().BoolVar(&opts.Merge, "merge", opts.Merge, "merge subdivider blocks (default: true)")
	cmd.Flags().BoolVar(&opts.Nebraska, "nebraska", false, "show Nebraska guy maintainer ranking")
	cmd.Flags().BoolVar(&opts.Popups, "popups", opts.Popups, "show hover popups with metadata")
	cmd.Flags().BoolVar(&opts.NoCache, "no-cache", false, "disable caching")

	return cmd
}

// parseFormats parses a comma-separated format string.
func parseFormats(s string) []string {
	if s == "" {
		return []string{pipeline.FormatSVG}
	}
	return strings.Split(s, ",")
}

// runRender loads the graph and renders via pipeline.
func runRender(ctx context.Context, input string, opts RenderCmdOpts) error {
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

	// Show spinner while rendering
	spinner := term.NewSpinner(fmt.Sprintf("Rendering %s...", opts.VizType))
	spinner.Start()

	// Generate layout DTO (unified for both tower and nodelink)
	layoutDTO, layoutHit, err := runner.GenerateLayoutWithCacheInfo(ctx, workGraph, pipeOpts)
	if err != nil {
		spinner.StopWithError("Render failed")
		return fmt.Errorf("layout: %w", err)
	}

	// Render to binary artifacts (SVG, PNG, PDF)
	artifacts, renderHit, err := runner.RenderWithCacheInfo(ctx, layoutDTO, workGraph, pipeOpts)
	if err != nil {
		spinner.StopWithError("Render failed")
		return fmt.Errorf("render: %w", err)
	}
	spinner.Stop()

	// Cache hit if both layout and render came from cache
	cacheHit := layoutHit && renderHit

	// Write artifact files
	return writeArtifacts(ArtifactWriteOpts{
		Artifacts: artifacts,
		Formats:   opts.Formats,
		Input:     input,
		Output:    opts.Output,
		NodeCount: g.NodeCount(),
		EdgeCount: g.EdgeCount(),
		CacheHit:  cacheHit,
	})
}
