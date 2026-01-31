package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/ordering"
	"github.com/matzehuels/stacktower/pkg/graph"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// renderCommand creates the render command for generating visualizations.
func (c *CLI) renderCommand() *cobra.Command {
	var (
		formatsStr   string
		output       string
		noCache      bool
		orderTimeout int
	)
	opts := pipeline.Options{}
	setCLIDefaults(&opts)

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
			return c.runRender(cmd.Context(), args[0], opts, output, noCache, orderTimeout)
		},
	}

	// Common flags
	cmd.Flags().StringVarP(&output, "output", "o", "", "output file (single format) or base path (multiple)")
	cmd.Flags().BoolVar(&noCache, "no-cache", false, "disable caching")

	// Layout flags
	cmd.Flags().StringVarP(&opts.VizType, "type", "t", opts.VizType, "visualization type: tower (default), nodelink")
	cmd.Flags().BoolVar(&opts.Normalize, "normalize", opts.Normalize, "apply graph normalization")
	cmd.Flags().Float64Var(&opts.Width, "width", opts.Width, "frame width")
	cmd.Flags().Float64Var(&opts.Height, "height", opts.Height, "frame height")
	cmd.Flags().StringVar(&opts.Ordering, "ordering", opts.Ordering, "ordering algorithm: optimal (default), barycentric")
	cmd.Flags().BoolVar(&opts.Randomize, "randomize", opts.Randomize, "randomize block widths (tower)")
	cmd.Flags().BoolVar(&opts.Merge, "merge", opts.Merge, "merge subdivider blocks (tower)")
	cmd.Flags().BoolVar(&opts.Nebraska, "nebraska", opts.Nebraska, "show Nebraska maintainer ranking (tower)")
	cmd.Flags().IntVar(&orderTimeout, "ordering-timeout", defaultOrderTimeout, "timeout in seconds for optimal ordering search")

	// Render flags
	cmd.Flags().StringVar(&opts.Style, "style", opts.Style, "visual style: handdrawn (default), simple")
	cmd.Flags().BoolVar(&opts.ShowEdges, "edges", opts.ShowEdges, "show dependency edges (tower)")
	cmd.Flags().BoolVar(&opts.Popups, "popups", opts.Popups, "show hover popups with metadata")
	cmd.Flags().StringVarP(&formatsStr, "format", "f", "", "output format(s): svg (default), pdf, png (comma-separated)")

	return cmd
}

// runRender loads the graph and renders via pipeline.
func (c *CLI) runRender(ctx context.Context, input string, opts pipeline.Options, output string, noCache bool, orderTimeout int) error {
	g, err := graph.ReadGraphFile(input)
	if err != nil {
		return fmt.Errorf("load graph %s: %w", input, err)
	}

	runner, err := c.newRunner(noCache)
	if err != nil {
		return fmt.Errorf("initialize runner: %w", err)
	}
	defer runner.Close()

	opts.Logger = c.Logger
	if opts.NeedsOptimalOrderer() {
		opts.Orderer = c.newOptimalOrderer(orderTimeout)
	}

	workGraph := runner.PrepareGraph(g, opts)

	spinner := newSpinnerWithContext(ctx, fmt.Sprintf("Rendering %s...", opts.VizType))
	spinner.Start()

	layout, layoutHit, err := runner.GenerateLayoutWithCacheInfo(ctx, workGraph, opts)
	if err != nil {
		spinner.StopWithError("Render failed")
		return fmt.Errorf("layout: %w", err)
	}

	if ctx.Err() != nil {
		spinner.Stop()
		return ctx.Err()
	}

	artifacts, renderHit, err := runner.RenderWithCacheInfo(ctx, layout, workGraph, opts)
	if err != nil {
		spinner.StopWithError("Render failed")
		return fmt.Errorf("render: %w", err)
	}
	spinner.Stop()

	return writeArtifacts(artifactWriteParams{
		artifacts: artifacts,
		formats:   opts.Formats,
		input:     input,
		output:    output,
		nodeCount: g.NodeCount(),
		edgeCount: g.EdgeCount(),
		cacheHit:  layoutHit && renderHit,
	})
}

// =============================================================================
// Optimal Orderer
// =============================================================================

// optimalOrderer wraps ordering.OptimalSearch with CLI progress feedback.
type optimalOrderer struct {
	ordering.OptimalSearch
	cli *CLI
}

// newOptimalOrderer creates an optimal orderer with a timeout.
func (c *CLI) newOptimalOrderer(timeoutSec int) ordering.Orderer {
	o := &optimalOrderer{cli: c}
	o.OptimalSearch = ordering.OptimalSearch{
		Timeout:  time.Duration(timeoutSec) * time.Second,
		Progress: o.onProgress,
		Debug:    o.onDebug,
	}
	return o
}

func (o *optimalOrderer) onProgress(explored, pruned, bestScore int) {
	if bestScore >= 0 {
		o.cli.Logger.Debug("search progress", "explored", explored, "pruned", pruned, "crossings", bestScore)
	}
}

func (o *optimalOrderer) onDebug(info ordering.DebugInfo) {
	o.cli.Logger.Debug("search complete", "rows", info.TotalRows, "depth", info.MaxDepth)
}

// OrderRows implements ordering.Orderer.
func (o *optimalOrderer) OrderRows(g *dag.DAG) map[int][]string {
	result := o.OptimalSearch.OrderRows(g)
	crossings := dag.CountCrossings(g, result)

	o.cli.Logger.Debug("ordering result", "crossings", crossings)

	if crossings > 0 {
		printWarning("Layout has %d edge crossings (try --ordering-timeout to increase search time)", crossings)
	}

	return result
}

// =============================================================================
// Artifact Writing
// =============================================================================

// artifactWriteParams configures artifact file writing.
type artifactWriteParams struct {
	artifacts map[string][]byte
	formats   []string
	input     string
	output    string
	nodeCount int
	edgeCount int
	cacheHit  bool
}

// writeArtifacts writes rendered artifacts to files and prints a summary.
func writeArtifacts(p artifactWriteParams) error {
	base := deriveBasePath(p.input, p.output)
	var paths []string

	for _, format := range p.formats {
		data, ok := p.artifacts[format]
		if !ok {
			return fmt.Errorf("missing artifact for format: %s", format)
		}

		path := p.output
		if path == "" || len(p.formats) > 1 {
			path = base + "." + format
		}

		if err := writeFile(data, path); err != nil {
			return err
		}
		paths = append(paths, path)
	}

	printSuccess("Render complete")
	for _, path := range paths {
		printFile(path)
	}
	printStats(p.nodeCount, p.edgeCount, p.cacheHit)
	return nil
}

// deriveBasePath computes the base path for output files.
func deriveBasePath(input, output string) string {
	if output != "" {
		ext := filepath.Ext(output)
		if pipeline.ValidFormats[strings.TrimPrefix(ext, ".")] {
			return strings.TrimSuffix(output, ext)
		}
		return output
	}
	base := strings.TrimSuffix(input, filepath.Ext(input))
	return strings.TrimSuffix(base, ".layout")
}

// writeFile writes raw data to the specified path (or stdout if empty).
func writeFile(data []byte, path string) error {
	if path == "" {
		_, err := os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(path, data, 0644)
}
