package cli

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/pkg/dag"
	dagtransform "github.com/matzehuels/stacktower/pkg/dag/transform"
	"github.com/matzehuels/stacktower/pkg/io"
	"github.com/matzehuels/stacktower/pkg/render/nodelink"
	"github.com/matzehuels/stacktower/pkg/render/tower/feature"
	"github.com/matzehuels/stacktower/pkg/render/tower/layout"
	"github.com/matzehuels/stacktower/pkg/render/tower/sink"
	"github.com/matzehuels/stacktower/pkg/render/tower/styles/handdrawn"
	"github.com/matzehuels/stacktower/pkg/render/tower/transform"
)

const (
	styleSimple    = "simple"    // plain rectangular blocks
	styleHanddrawn = "handdrawn" // hand-drawn sketch style with randomized widths
	defaultWidth   = 800         // default SVG viewport width
	defaultHeight  = 600         // default SVG viewport height
	defaultSeed    = 42          // random seed for reproducible randomization
)

// renderOpts holds the command-line flags for the render command.
// These options control visualization style, layout algorithms, and output formats.
type renderOpts struct {
	output       string   // output file path (or base path for multiple outputs)
	vizTypes     []string // visualization types: "tower", "nodelink"
	formats      []string // output formats: "svg", "pdf", "png", "json"
	detailed     bool     // show detailed metadata in nodelink diagrams
	normalize    bool     // apply DAG normalization (remove cycles, transitive edges, add subdividers)
	width        float64  // viewport width in pixels
	height       float64  // viewport height in pixels
	showEdges    bool     // draw dependency edges in tower view
	style        string   // visual style: "simple" or "handdrawn"
	ordering     string   // ordering algorithm: "optimal" or "barycentric"
	orderTimeout int      // timeout in seconds for optimal search
	randomize    bool     // randomize block widths for hand-drawn effect
	merge        bool     // merge subdivider blocks into single towers
	nebraska     bool     // show Nebraska guy maintainer ranking
	popups       bool     // enable hover popups with metadata
	topDown      bool     // use top-down width allocation (roots get equal width)
}

// newRenderCmd creates the render command for generating visualizations.
// It supports multiple visualization types (tower, nodelink) and output formats (SVG, PDF, PNG, JSON).
//
// Default settings:
//   - normalize: true (clean up cycles and transitive edges)
//   - style: handdrawn (sketch-style with randomized widths)
//   - width: 800px, height: 600px
//   - ordering: optimal (with 60s timeout)
//   - merge: true (combine subdividers into single towers)
//   - popups: true (show metadata on hover)
func newRenderCmd() *cobra.Command {
	var vizTypesStr, formatsStr string
	opts := renderOpts{
		normalize: true,
		width:     defaultWidth,
		height:    defaultHeight,
		style:     styleHanddrawn,
		randomize: true,
		merge:     true,
		popups:    true,
	}

	cmd := &cobra.Command{
		Use:   "render [file]",
		Short: "Render a dependency graph to SVG(s)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.vizTypes = parseVizTypes(vizTypesStr)
			opts.formats = parseFormats(formatsStr)
			if err := validateStyle(opts.style); err != nil {
				return err
			}
			if err := validateFormats(opts.formats); err != nil {
				return err
			}
			return runRender(cmd.Context(), args[0], &opts)
		},
	}

	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "output file (single type/format) or base path (multiple)")
	cmd.Flags().StringVarP(&vizTypesStr, "type", "t", "", "visualization type(s): tower (default), nodelink (comma-separated)")
	cmd.Flags().StringVarP(&formatsStr, "format", "f", "", "output format(s): svg (default), json, pdf, png (comma-separated)")
	cmd.Flags().BoolVar(&opts.detailed, "detailed", false, "show detailed information (nodelink)")
	cmd.Flags().BoolVar(&opts.normalize, "normalize", opts.normalize, "apply normalization pipeline")
	cmd.Flags().Float64Var(&opts.width, "width", opts.width, "frame width")
	cmd.Flags().Float64Var(&opts.height, "height", opts.height, "frame height")
	cmd.Flags().BoolVar(&opts.showEdges, "edges", false, "show dependency edges")
	cmd.Flags().StringVar(&opts.style, "style", opts.style, "visual style: handdrawn (default), simple")
	cmd.Flags().StringVar(&opts.ordering, "ordering", "", "ordering algorithm: optimal (default), barycentric")
	cmd.Flags().IntVar(&opts.orderTimeout, "ordering-timeout", 60, "timeout in seconds for optimal search")
	cmd.Flags().BoolVar(&opts.randomize, "randomize", opts.randomize, "randomize block widths for hand-drawn effect")
	cmd.Flags().BoolVar(&opts.merge, "merge", opts.merge, "merge subdivider blocks into single towers")
	cmd.Flags().BoolVar(&opts.nebraska, "nebraska", false, "show Nebraska guy maintainer ranking")
	cmd.Flags().BoolVar(&opts.popups, "popups", opts.popups, "show hover popups with metadata")
	cmd.Flags().BoolVar(&opts.topDown, "top-down", false, "use top-down width flow (roots get equal width)")

	return cmd
}

// parseVizTypes parses the --type flag into a slice of visualization types.
// If empty, defaults to ["tower"].
func parseVizTypes(s string) []string {
	if s == "" {
		return []string{"tower"}
	}
	return strings.Split(s, ",")
}

// parseFormats parses the --format flag into a slice of output formats.
// If empty, defaults to ["svg"].
func parseFormats(s string) []string {
	if s == "" {
		return []string{"svg"}
	}
	return strings.Split(s, ",")
}

// validFormats is the set of supported output formats.
var validFormats = map[string]bool{"svg": true, "json": true, "pdf": true, "png": true}

// validateFormats checks that all requested formats are valid.
// It returns an error if any format is not in validFormats.
func validateFormats(formats []string) error {
	for _, f := range formats {
		if !validFormats[f] {
			return fmt.Errorf("invalid format: %s (must be 'svg', 'json', 'pdf', or 'png')", f)
		}
	}
	return nil
}

// validateStyle checks that the style is either "simple" or "handdrawn".
func validateStyle(s string) error {
	if s != styleSimple && s != styleHanddrawn {
		return fmt.Errorf("invalid style: %s (must be 'simple' or 'handdrawn')", s)
	}
	return nil
}

// basePath derives the base output path from the output and input file paths.
// If output is empty, it strips the extension from input.
// If output has a format extension (.svg, .pdf, etc.), it strips that extension.
// This is used when generating multiple files (e.g., graph_tower.svg, graph_nodelink.svg).
func basePath(output, input string) string {
	if output == "" {
		return strings.TrimSuffix(input, filepath.Ext(input))
	}
	// Strip known format extensions from output path
	ext := filepath.Ext(output)
	if validFormats[strings.TrimPrefix(ext, ".")] {
		return strings.TrimSuffix(output, ext)
	}
	return output
}

// runRender loads the graph from input, optionally normalizes it, and renders it to the requested formats.
// If opts.normalize is true, the graph is cleaned up by removing cycles, transitive edges, and adding subdividers.
func runRender(ctx context.Context, input string, opts *renderOpts) error {
	logger := loggerFromContext(ctx)
	logger.Infof("Rendering %s", input)

	g, err := io.ImportJSON(input)
	if err != nil {
		return err
	}
	logger.Infof("Loaded graph: %d nodes, %d edges", g.NodeCount(), g.EdgeCount())

	if opts.normalize {
		result := dagtransform.Normalize(g)
		logger.Infof("Normalized: %d nodes, %d edges (removed %d cycles, %d transitive edges; added %d subdividers, %d separators)",
			g.NodeCount(), g.EdgeCount(),
			result.CyclesRemoved, result.TransitiveEdgesRemoved,
			result.SubdividersAdded, result.SeparatorsAdded)
	}

	if len(opts.vizTypes) == 1 && len(opts.formats) == 1 {
		return renderSingle(ctx, g, opts.vizTypes[0], opts.formats[0], input, opts)
	}
	return renderMultiple(ctx, g, input, opts)
}

// renderSingle renders a single visualization type and format to a single output file.
// If opts.output is empty, the output path is derived from the input file name.
func renderSingle(ctx context.Context, g *dag.DAG, vizType, format string, input string, opts *renderOpts) error {
	logger := loggerFromContext(ctx)

	data, err := renderGraph(ctx, g, vizType, format, opts)
	if err != nil {
		return err
	}
	logger.Debugf("Generated %s: %d bytes", format, len(data))

	// Determine output path: use provided output or derive from input
	outputPath := opts.output
	if outputPath == "" {
		outputPath = basePath("", input) + "." + format
	}

	out, err := openOutput(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = out.Write(data); err != nil {
		return err
	}

	logger.Infof("Generated %s", outputPath)
	return nil
}

// renderMultiple renders all requested visualization type/format combinations to separate files.
// File names are derived from basePath and include the visualization type when multiple types are requested.
func renderMultiple(ctx context.Context, g *dag.DAG, input string, opts *renderOpts) error {
	base := basePath(opts.output, input)

	for _, vizType := range opts.vizTypes {
		for _, format := range opts.formats {
			if err := renderAndWrite(ctx, g, vizType, format, base, opts); err != nil {
				return err
			}
		}
	}
	return nil
}

// renderAndWrite renders a single viz/format combination and writes it to a file.
// If the combination is unsupported (e.g., nodelink JSON), it is silently skipped with a debug log.
func renderAndWrite(ctx context.Context, g *dag.DAG, vizType, format, basePath string, opts *renderOpts) error {
	logger := loggerFromContext(ctx)

	data, err := renderGraph(ctx, g, vizType, format, opts)
	if errors.Is(err, errSkipFormat) {
		logger.Debugf("Skipping %s/%s (unsupported combination)", vizType, format)
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s/%s: %w", vizType, format, err)
	}

	// Build filename: base_type.format (or base.format if single type)
	var path string
	if len(opts.vizTypes) == 1 {
		path = fmt.Sprintf("%s.%s", basePath, format)
	} else {
		path = fmt.Sprintf("%s_%s.%s", basePath, vizType, format)
	}

	out, err := openOutput(path)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := out.Write(data); err != nil {
		return err
	}

	logger.Infof("Generated %s", path)
	return nil
}

// errSkipFormat is a sentinel error indicating an unsupported format/visualization combination.
var errSkipFormat = fmt.Errorf("skip unsupported format")

// renderGraph dispatches to the appropriate renderer based on vizType.
// It returns errSkipFormat for unsupported combinations (e.g., nodelink JSON).
func renderGraph(ctx context.Context, g *dag.DAG, vizType, format string, opts *renderOpts) ([]byte, error) {
	switch vizType {
	case "nodelink":
		return renderNodeLink(ctx, g, format, opts)
	case "tower":
		return renderTower(ctx, g, format, opts)
	default:
		return nil, fmt.Errorf("unknown visualization type: %s", vizType)
	}
}

// renderNodeLink generates a node-link (force-directed) diagram using Graphviz.
// It supports SVG, PDF, and PNG formats. JSON is not supported (returns errSkipFormat).
func renderNodeLink(ctx context.Context, g *dag.DAG, format string, opts *renderOpts) ([]byte, error) {
	logger := loggerFromContext(ctx)
	logger.Info("Generating node-link diagram")
	dot := nodelink.ToDOT(g, nodelink.Options{Detailed: opts.detailed})

	switch format {
	case "svg":
		logger.Info("Rendering node-link SVG")
		return nodelink.RenderSVG(dot)
	case "pdf":
		logger.Info("Rendering node-link PDF")
		return nodelink.RenderPDF(dot)
	case "png":
		logger.Info("Rendering node-link PNG")
		return nodelink.RenderPNG(dot, 2.0)
	case "json":
		return nil, errSkipFormat // JSON layout export only makes sense for tower
	default:
		return nil, fmt.Errorf("unknown format: %s", format)
	}
}

// renderTower generates a tower visualization with layered blocks.
// It computes the layout using the specified ordering algorithm (optimal or barycentric),
// optionally merges subdividers and randomizes widths, then renders to the requested format.
func renderTower(ctx context.Context, g *dag.DAG, format string, opts *renderOpts) ([]byte, error) {
	logger := loggerFromContext(ctx)

	algo := opts.ordering
	if algo == "" {
		algo = "optimal"
	}
	logger.Infof("Computing tower layout using %s ordering", algo)

	layoutOpts, err := buildLayoutOpts(ctx, opts)
	if err != nil {
		return nil, err
	}

	l := layout.Build(g, opts.width, opts.height, layoutOpts...)
	logger.Debugf("Layout computed: %d blocks", len(l.Blocks))

	if opts.merge {
		before := len(l.Blocks)
		l = transform.MergeSubdividers(l, g)
		logger.Debugf("Merged subdividers: %d → %d blocks", before, len(l.Blocks))
	}
	if opts.randomize {
		l = transform.Randomize(l, g, defaultSeed, nil)
	}

	switch format {
	case "json":
		logger.Info("Rendering tower layout as JSON")
		return sink.RenderJSON(l, buildJSONOpts(g, opts)...)
	case "pdf":
		logger.Info("Rendering tower PDF")
		return sink.RenderPDF(l, sink.WithPDFSVGOptions(buildRenderOpts(g, opts)...))
	case "png":
		logger.Info("Rendering tower PNG")
		return sink.RenderPNG(l, sink.WithPNGSVGOptions(buildRenderOpts(g, opts)...))
	case "svg":
		logger.Infof("Rendering tower SVG (%s style)", opts.style)
		return sink.RenderSVG(l, buildRenderOpts(g, opts)...), nil
	default:
		return nil, fmt.Errorf("unknown format: %s", format)
	}
}

// buildLayoutOpts constructs layout.Options based on the ordering algorithm and width flow settings.
// The "optimal" algorithm uses branch-and-bound search with a timeout; "barycentric" is the default.
func buildLayoutOpts(ctx context.Context, opts *renderOpts) ([]layout.Option, error) {
	logger := loggerFromContext(ctx)
	var layoutOpts []layout.Option

	switch opts.ordering {
	case "barycentric":
		// default barycentric, no option needed
	case "optimal", "":
		logger.Debugf("Using optimal search with %ds timeout", opts.orderTimeout)
		layoutOpts = append(layoutOpts, layout.WithOrderer(newOptimalOrderer(ctx, opts.orderTimeout)))
	default:
		return nil, fmt.Errorf("unknown ordering: %s", opts.ordering)
	}

	if opts.topDown {
		logger.Debug("Using top-down width flow")
		layoutOpts = append(layoutOpts, layout.WithTopDownWidths())
	}
	return layoutOpts, nil
}

// buildRenderOpts constructs SVG rendering options based on style, edges, and feature flags.
// The handdrawn style supports Nebraska maintainer ranking and hover popups.
func buildRenderOpts(g *dag.DAG, opts *renderOpts) []sink.SVGOption {
	result := []sink.SVGOption{sink.WithGraph(g)}
	if opts.showEdges {
		result = append(result, sink.WithEdges())
	}
	if opts.merge {
		result = append(result, sink.WithMerged())
	}
	if opts.style == styleHanddrawn {
		result = append(result, sink.WithStyle(handdrawn.New(defaultSeed)))
		if opts.nebraska {
			result = append(result, sink.WithNebraska(feature.RankNebraska(g, 5)))
		}
		if opts.popups {
			result = append(result, sink.WithPopups())
		}
	}
	return result
}

// buildJSONOpts constructs JSON rendering options including graph metadata and feature flags.
func buildJSONOpts(g *dag.DAG, opts *renderOpts) []sink.JSONOption {
	result := []sink.JSONOption{sink.WithJSONGraph(g)}
	if opts.merge {
		result = append(result, sink.WithJSONMerged())
	}
	if opts.randomize {
		result = append(result, sink.WithJSONRandomize(defaultSeed))
	}
	if opts.style != "" {
		result = append(result, sink.WithJSONStyle(opts.style))
	}
	if opts.nebraska {
		result = append(result, sink.WithJSONNebraska(feature.RankNebraska(g, 5)))
	}
	return result
}
