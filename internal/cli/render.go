package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/pkg/io"
	"github.com/matzehuels/stacktower/pkg/logging"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

const (
	styleSimple    = "simple"    // plain rectangular blocks
	styleHanddrawn = "handdrawn" // hand-drawn sketch style
	defaultWidth   = 800         // default SVG viewport width
	defaultHeight  = 600         // default SVG viewport height
	defaultSeed    = 42          // random seed for reproducibility
)

// renderOpts holds the command-line flags for the render command.
type renderOpts struct {
	output       string   // output file path
	vizType      string   // visualization type: "tower" or "nodelink"
	formats      []string // output formats: "svg", "pdf", "png"
	normalize    bool     // apply DAG normalization
	width        float64  // viewport width in pixels
	height       float64  // viewport height in pixels
	showEdges    bool     // draw dependency edges in tower view
	style        string   // visual style: "simple" or "handdrawn"
	ordering     string   // ordering algorithm: "optimal" or "barycentric"
	orderTimeout int      // timeout in seconds for optimal search
	randomize    bool     // randomize block widths
	merge        bool     // merge subdivider blocks
	nebraska     bool     // show Nebraska guy maintainer ranking
	popups       bool     // enable hover popups with metadata
	topDown      bool     // use top-down width allocation
	noCache      bool     // disable caching
}

// newRenderCmd creates the render command for generating visualizations.
func newRenderCmd() *cobra.Command {
	var formatsStr string
	opts := renderOpts{
		vizType:      "tower",
		normalize:    true,
		width:        defaultWidth,
		height:       defaultHeight,
		style:        styleHanddrawn,
		ordering:     "optimal",
		orderTimeout: 60,
		randomize:    true,
		merge:        true,
		popups:       true,
	}

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
			opts.formats = parseFormats(formatsStr)
			if err := validateFormats(opts.formats); err != nil {
				return err
			}
			if err := validateStyle(opts.style); err != nil {
				return err
			}
			return runRender(cmd.Context(), args[0], &opts)
		},
	}

	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "output file (single format) or base path (multiple)")
	cmd.Flags().StringVarP(&opts.vizType, "type", "t", opts.vizType, "visualization type: tower (default), nodelink")
	cmd.Flags().StringVarP(&formatsStr, "format", "f", "", "output format(s): svg (default), pdf, png (comma-separated)")
	cmd.Flags().BoolVar(&opts.normalize, "normalize", opts.normalize, "apply normalization pipeline")
	cmd.Flags().Float64Var(&opts.width, "width", opts.width, "frame width")
	cmd.Flags().Float64Var(&opts.height, "height", opts.height, "frame height")
	cmd.Flags().BoolVar(&opts.showEdges, "edges", false, "show dependency edges (tower)")
	cmd.Flags().StringVar(&opts.style, "style", opts.style, "visual style: handdrawn (default), simple (tower)")
	cmd.Flags().StringVar(&opts.ordering, "ordering", opts.ordering, "ordering algorithm: optimal (default), barycentric")
	cmd.Flags().IntVar(&opts.orderTimeout, "ordering-timeout", opts.orderTimeout, "timeout in seconds for optimal search")
	cmd.Flags().BoolVar(&opts.randomize, "randomize", opts.randomize, "randomize block widths")
	cmd.Flags().BoolVar(&opts.merge, "merge", opts.merge, "merge subdivider blocks")
	cmd.Flags().BoolVar(&opts.nebraska, "nebraska", false, "show Nebraska guy maintainer ranking")
	cmd.Flags().BoolVar(&opts.popups, "popups", opts.popups, "show hover popups with metadata")
	cmd.Flags().BoolVar(&opts.topDown, "top-down", false, "use top-down width flow")
	cmd.Flags().BoolVar(&opts.noCache, "no-cache", false, "disable caching")

	return cmd
}

// parseFormats parses a comma-separated format string.
func parseFormats(s string) []string {
	if s == "" {
		return []string{"svg"}
	}
	return strings.Split(s, ",")
}

// validFormats is the set of supported output formats.
var validFormats = map[string]bool{"svg": true, "pdf": true, "png": true}

// validateFormats checks that all requested formats are valid.
func validateFormats(formats []string) error {
	for _, f := range formats {
		if !validFormats[f] {
			return fmt.Errorf("invalid format: %s (must be 'svg', 'pdf', or 'png')", f)
		}
	}
	return nil
}

// validateStyle checks that the style is valid.
func validateStyle(s string) error {
	if s != styleSimple && s != styleHanddrawn {
		return fmt.Errorf("invalid style: %s (must be 'simple' or 'handdrawn')", s)
	}
	return nil
}

// runRender loads the graph and renders via pipeline.
func runRender(ctx context.Context, input string, opts *renderOpts) error {
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
		Formats:   opts.formats,
		Style:     opts.style,
		ShowEdges: opts.showEdges,
		Nebraska:  opts.nebraska,
		Popups:    opts.popups,
		Logger:    logger,
	}

	// Set orderer for optimal ordering
	if opts.ordering == "optimal" || opts.ordering == "" {
		pipelineOpts.Orderer = newOptimalOrderer(ctx, opts.orderTimeout)
	}

	// Render via pipeline (layout + visualize)
	artifacts, cacheHit, err := svc.Render(ctx, g, pipelineOpts)
	if err != nil {
		return err
	}

	// Write outputs
	status := formatCacheStatus(cacheHit)
	return writeRenderArtifacts(ctx, artifacts, opts, input, status)
}

// writeRenderArtifacts writes rendered artifacts to files.
func writeRenderArtifacts(ctx context.Context, artifacts map[string][]byte, opts *renderOpts, input, status string) error {
	logger := logging.FromContext(ctx)

	// Derive base path
	base := opts.output
	if base == "" {
		base = strings.TrimSuffix(input, filepath.Ext(input))
	} else {
		// Strip format extension if present
		ext := filepath.Ext(base)
		if validFormats[strings.TrimPrefix(ext, ".")] {
			base = strings.TrimSuffix(base, ext)
		}
	}

	// Single format: use output directly if specified
	if len(opts.formats) == 1 {
		format := opts.formats[0]
		data, ok := artifacts[format]
		if !ok {
			return fmt.Errorf("missing artifact for format: %s", format)
		}

		path := opts.output
		if path == "" {
			path = base + "." + format
		}

		if err := writeData(data, path, logger); err != nil {
			return err
		}
		logger.Infof("Generated %s (%s)", path, status)
		return nil
	}

	// Multiple formats: write each
	for _, format := range opts.formats {
		data, ok := artifacts[format]
		if !ok {
			return fmt.Errorf("missing artifact for format: %s", format)
		}

		path := fmt.Sprintf("%s.%s", base, format)
		if err := writeData(data, path, logger); err != nil {
			return err
		}
		logger.Infof("Generated %s (%s)", path, status)
	}
	return nil
}
