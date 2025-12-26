package cli

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/pkg/io"
	"github.com/matzehuels/stacktower/pkg/logging"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// layoutOpts holds the command-line flags for the layout command.
type layoutOpts struct {
	output       string  // output file path
	vizType      string  // visualization type: "tower" or "nodelink"
	normalize    bool    // apply DAG normalization
	width        float64 // viewport width in pixels
	height       float64 // viewport height in pixels
	ordering     string  // ordering algorithm: "optimal" or "barycentric"
	orderTimeout int     // timeout in seconds for optimal search
	randomize    bool    // randomize block widths
	merge        bool    // merge subdivider blocks
	nebraska     bool    // include Nebraska maintainer ranking
	topDown      bool    // use top-down width allocation
	noCache      bool    // disable caching
}

// newLayoutCmd creates the layout command for computing visualization layouts.
func newLayoutCmd() *cobra.Command {
	opts := layoutOpts{
		vizType:      "tower",
		normalize:    true,
		width:        defaultWidth,
		height:       defaultHeight,
		ordering:     "optimal",
		orderTimeout: 60,
		randomize:    true,
		merge:        true,
	}

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
			return runLayout(cmd.Context(), args[0], &opts)
		},
	}

	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "output file (default: input with .layout.json extension)")
	cmd.Flags().StringVarP(&opts.vizType, "type", "t", opts.vizType, "visualization type: tower (default), nodelink")
	cmd.Flags().BoolVar(&opts.normalize, "normalize", opts.normalize, "apply normalization pipeline")
	cmd.Flags().Float64Var(&opts.width, "width", opts.width, "frame width")
	cmd.Flags().Float64Var(&opts.height, "height", opts.height, "frame height")
	cmd.Flags().StringVar(&opts.ordering, "ordering", opts.ordering, "ordering algorithm: optimal (default), barycentric")
	cmd.Flags().IntVar(&opts.orderTimeout, "ordering-timeout", opts.orderTimeout, "timeout in seconds for optimal search")
	cmd.Flags().BoolVar(&opts.randomize, "randomize", opts.randomize, "randomize block widths (tower)")
	cmd.Flags().BoolVar(&opts.merge, "merge", opts.merge, "merge subdivider blocks (tower)")
	cmd.Flags().BoolVar(&opts.nebraska, "nebraska", false, "include Nebraska maintainer ranking (tower)")
	cmd.Flags().BoolVar(&opts.topDown, "top-down", false, "use top-down width flow (tower)")
	cmd.Flags().BoolVar(&opts.noCache, "no-cache", false, "disable caching")

	return cmd
}

// runLayout loads the graph, computes the layout via pipeline, and writes output.
func runLayout(ctx context.Context, input string, opts *layoutOpts) error {
	logger := logging.FromContext(ctx)

	// Load graph
	g, err := io.ImportJSON(input)
	if err != nil {
		return err
	}
	logger.Infof("Loaded graph: %d nodes, %d edges", g.NodeCount(), g.EdgeCount())

	// Create pipeline service
	svc, cleanup, err := newPipelineService(opts.noCache, logger)
	if err != nil {
		return err
	}
	defer cleanup()

	// Build pipeline options (normalization handled by pipeline)
	pipelineOpts := pipeline.Options{
		Normalize: opts.normalize,
		VizType:   opts.vizType,
		Width:     opts.width,
		Height:    opts.height,
		Ordering:  opts.ordering,
		Merge:     opts.merge,
		Randomize: opts.randomize,
		Seed:      defaultSeed,
		Nebraska:  opts.nebraska,
		Logger:    logger,
	}

	// Set orderer for optimal ordering
	if opts.ordering == "optimal" || opts.ordering == "" {
		pipelineOpts.Orderer = newOptimalOrderer(ctx, opts.orderTimeout)
	}

	// Compute layout via pipeline
	layoutData, cacheHit, err := svc.Layout(ctx, g, pipelineOpts)
	if err != nil {
		return err
	}

	// Determine output path
	outputPath := opts.output
	if outputPath == "" {
		base := strings.TrimSuffix(input, filepath.Ext(input))
		outputPath = base + ".layout.json"
	}

	// Write output
	if err := writeData(layoutData, outputPath, logger); err != nil {
		return err
	}

	logger.Infof("Generated %s (%s)", outputPath, formatCacheStatus(cacheHit))
	return nil
}
