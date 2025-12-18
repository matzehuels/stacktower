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
	styleSimple    = "simple"
	styleHanddrawn = "handdrawn"
	defaultWidth   = 800
	defaultHeight  = 600
	defaultSeed    = 42
)

type renderOpts struct {
	output       string
	vizTypes     []string
	formats      []string
	detailed     bool
	normalize    bool
	width        float64
	height       float64
	showEdges    bool
	style        string
	ordering     string
	orderTimeout int
	randomize    bool
	merge        bool
	nebraska     bool
	popups       bool
	topDown      bool
}

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

func parseVizTypes(s string) []string {
	if s == "" {
		return []string{"tower"}
	}
	return strings.Split(s, ",")
}

func parseFormats(s string) []string {
	if s == "" {
		return []string{"svg"}
	}
	return strings.Split(s, ",")
}

var validFormats = map[string]bool{"svg": true, "json": true, "pdf": true, "png": true}

func validateFormats(formats []string) error {
	for _, f := range formats {
		if !validFormats[f] {
			return fmt.Errorf("invalid format: %s (must be 'svg', 'json', 'pdf', or 'png')", f)
		}
	}
	return nil
}

func validateStyle(s string) error {
	if s != styleSimple && s != styleHanddrawn {
		return fmt.Errorf("invalid style: %s (must be 'simple' or 'handdrawn')", s)
	}
	return nil
}

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

func runRender(ctx context.Context, input string, opts *renderOpts) error {
	logger := loggerFromContext(ctx)
	logger.Infof("Rendering %s", input)

	g, err := io.ImportJSON(input)
	if err != nil {
		return err
	}
	logger.Infof("Loaded graph: %d nodes, %d edges", g.NodeCount(), g.EdgeCount())

	if opts.normalize {
		before := g.NodeCount()
		g = dagtransform.Normalize(g)
		logger.Infof("Normalized: %d nodes (%+d), %d edges", g.NodeCount(), g.NodeCount()-before, g.EdgeCount())
	}

	if len(opts.vizTypes) == 1 && len(opts.formats) == 1 {
		return renderSingle(ctx, g, opts.vizTypes[0], opts.formats[0], input, opts)
	}
	return renderMultiple(ctx, g, input, opts)
}

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

var errSkipFormat = fmt.Errorf("skip unsupported format")

func renderGraph(ctx context.Context, g *dag.DAG, vizType, format string, opts *renderOpts) ([]byte, error) {
	switch vizType {
	case "nodelink":
		if format != "svg" {
			return nil, errSkipFormat
		}
		return renderNodeLink(ctx, g, opts)
	case "tower":
		return renderTower(ctx, g, format, opts)
	default:
		return nil, fmt.Errorf("unknown visualization type: %s", vizType)
	}
}

func renderNodeLink(ctx context.Context, g *dag.DAG, opts *renderOpts) ([]byte, error) {
	loggerFromContext(ctx).Info("Generating node-link diagram")
	dot := nodelink.ToDOT(g, nodelink.Options{Detailed: opts.detailed})
	return nodelink.RenderSVG(dot)
}

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
