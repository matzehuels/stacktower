package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"stacktower/pkg/dag"
	dagtransform "stacktower/pkg/dag/transform"
	"stacktower/pkg/io"
	"stacktower/pkg/render/nodelink"
	"stacktower/pkg/render/tower"
	"stacktower/pkg/render/tower/ordering"
	"stacktower/pkg/render/tower/styles/handdrawn"
	layouttransform "stacktower/pkg/render/tower/transform"
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
	var vizTypesStr string
	opts := renderOpts{
		normalize: true,
		width:     defaultWidth,
		height:    defaultHeight,
		style:     styleSimple,
	}

	cmd := &cobra.Command{
		Use:   "render [file]",
		Short: "Render a dependency graph to SVG(s)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.vizTypes = parseVizTypes(vizTypesStr)
			if err := validateStyle(opts.style); err != nil {
				return err
			}
			return runRender(cmd.Context(), args[0], &opts)
		},
	}

	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "output file (single type) or base path (multiple types)")
	cmd.Flags().StringVarP(&vizTypesStr, "type", "t", "", "visualization types: nodelink, tower (comma-separated)")
	cmd.Flags().BoolVar(&opts.detailed, "detailed", false, "show detailed information (nodelink)")
	cmd.Flags().BoolVar(&opts.normalize, "normalize", opts.normalize, "apply normalization pipeline")
	cmd.Flags().Float64Var(&opts.width, "width", opts.width, "frame width (tower)")
	cmd.Flags().Float64Var(&opts.height, "height", opts.height, "frame height (tower)")
	cmd.Flags().BoolVar(&opts.showEdges, "edges", false, "show edges (tower)")
	cmd.Flags().StringVar(&opts.style, "style", opts.style, "visual style: simple or handdrawn (tower)")
	cmd.Flags().StringVar(&opts.ordering, "ordering", "", "ordering algorithm: optimal (default), barycentric")
	cmd.Flags().IntVar(&opts.orderTimeout, "ordering-timeout", 60, "timeout in seconds for optimal search")
	cmd.Flags().BoolVar(&opts.randomize, "randomize", false, "randomize positions for hand-drawn effect (tower)")
	cmd.Flags().BoolVar(&opts.merge, "merge", false, "merge subdivider blocks (tower)")
	cmd.Flags().BoolVar(&opts.nebraska, "nebraska", false, "show Nebraska guy ranking (handdrawn)")
	cmd.Flags().BoolVar(&opts.popups, "popups", false, "show hover popups (handdrawn)")
	cmd.Flags().BoolVar(&opts.topDown, "top-down", false, "use top-down width flow (roots get equal width)")

	return cmd
}

func parseVizTypes(s string) []string {
	if s == "" {
		return []string{"nodelink"}
	}
	return strings.Split(s, ",")
}

func validateStyle(s string) error {
	if s != styleSimple && s != styleHanddrawn {
		return fmt.Errorf("invalid style: %s (must be 'simple' or 'handdrawn')", s)
	}
	return nil
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

	if len(opts.vizTypes) == 1 {
		return renderSingle(ctx, g, opts.vizTypes[0], opts)
	}
	return renderMultiple(ctx, g, input, opts)
}

func renderSingle(ctx context.Context, g *dag.DAG, vizType string, opts *renderOpts) error {
	logger := loggerFromContext(ctx)

	svg, err := renderGraph(ctx, g, vizType, opts)
	if err != nil {
		return err
	}
	logger.Debugf("Generated SVG: %d bytes", len(svg))

	out, err := openOutput(opts.output)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = out.Write(svg); err != nil {
		return err
	}

	if opts.output != "" {
		logger.Infof("Generated %s", opts.output)
	}
	return nil
}

func renderMultiple(ctx context.Context, g *dag.DAG, input string, opts *renderOpts) error {
	basePath := opts.output
	if basePath == "" {
		basePath = strings.TrimSuffix(input, filepath.Ext(input))
	}

	for _, vizType := range opts.vizTypes {
		if err := renderAndWrite(ctx, g, vizType, basePath, opts); err != nil {
			return err
		}
	}
	return nil
}

func renderAndWrite(ctx context.Context, g *dag.DAG, vizType, basePath string, opts *renderOpts) error {
	logger := loggerFromContext(ctx)

	svg, err := renderGraph(ctx, g, vizType, opts)
	if err != nil {
		return fmt.Errorf("%s: %w", vizType, err)
	}

	path := fmt.Sprintf("%s_%s.svg", basePath, vizType)
	out, err := openOutput(path)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := out.Write(svg); err != nil {
		return err
	}

	logger.Infof("Generated %s", path)
	return nil
}

func renderGraph(ctx context.Context, g *dag.DAG, vizType string, opts *renderOpts) ([]byte, error) {
	switch vizType {
	case "nodelink":
		return renderNodeLink(ctx, g, opts)
	case "tower":
		return renderTower(ctx, g, opts)
	default:
		return nil, fmt.Errorf("unknown visualization type: %s", vizType)
	}
}

func renderNodeLink(ctx context.Context, g *dag.DAG, opts *renderOpts) ([]byte, error) {
	logger := loggerFromContext(ctx)
	logger.Info("Generating node-link diagram")
	dot := nodelink.ToDOT(g, nodelink.Options{Detailed: opts.detailed})
	return nodelink.RenderSVG(dot)
}

func renderTower(ctx context.Context, g *dag.DAG, opts *renderOpts) ([]byte, error) {
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

	layout := tower.Build(g, opts.width, opts.height, layoutOpts...)
	logger.Debugf("Layout computed: %d blocks", len(layout.Blocks))

	if opts.merge {
		before := len(layout.Blocks)
		layout = layouttransform.MergeSubdividers(layout, g)
		logger.Debugf("Merged subdividers: %d → %d blocks", before, len(layout.Blocks))
	}
	if opts.randomize {
		layout = layouttransform.Randomize(layout, g, defaultSeed, nil)
	}

	logger.Infof("Rendering tower SVG (%s style)", opts.style)
	renderOpts := buildRenderOpts(g, opts)
	return tower.RenderSVG(layout, renderOpts...), nil
}

func buildLayoutOpts(ctx context.Context, opts *renderOpts) ([]tower.Option, error) {
	var layoutOpts []tower.Option

	switch opts.ordering {
	case "barycentric":
	case "optimal", "":
		loggerFromContext(ctx).Debugf("Using optimal search with %ds timeout", opts.orderTimeout)
		layoutOpts = append(layoutOpts, tower.WithOrderer(withOptimalSearchProgress(ctx, opts.orderTimeout)))
	default:
		return nil, fmt.Errorf("unknown ordering: %s", opts.ordering)
	}

	if opts.topDown {
		loggerFromContext(ctx).Debug("Using top-down width flow")
		layoutOpts = append(layoutOpts, tower.WithTopDownWidths())
	}

	return layoutOpts, nil
}

func withOptimalSearchProgress(ctx context.Context, timeoutSec int) ordering.Orderer {
	logger := loggerFromContext(ctx)
	o := &optimalSearchOrderer{prog: newProgress(logger), logger: logger, lastBest: -1}

	o.OptimalSearch = ordering.OptimalSearch{
		Timeout: time.Duration(timeoutSec) * time.Second,
		Progress: func(explored, pruned, bestScore int) {
			o.lastExplored, o.lastPruned = explored, pruned
			if bestScore < 0 || (explored == 0 && pruned == 0) {
				return
			}

			switch {
			case o.lastBest < 0:
				logger.Infof("Initial: %d crossings (explored: %d, pruned: %d)", bestScore, explored, pruned)
			case bestScore < o.lastBest:
				logger.Infof("Improved: %d crossings (↓%d)", bestScore, o.lastBest-bestScore)
			}
			o.lastBest = bestScore
		},
	}
	return o
}

type optimalSearchOrderer struct {
	ordering.OptimalSearch
	prog                     *progress
	logger                   *log.Logger
	lastExplored, lastPruned int
	lastBest                 int
}

func (o *optimalSearchOrderer) OrderRows(g *dag.DAG) map[int][]string {
	result := o.OptimalSearch.OrderRows(g)
	crossings := dag.CountCrossings(g, result)
	o.prog.done(fmt.Sprintf("Layout complete: %d crossings", crossings))
	if crossings >= 0 {
		o.logger.Infof("Best: %d crossings (explored: %d, pruned: %d)",
			crossings, o.lastExplored, o.lastPruned)
	}
	return result
}

func buildRenderOpts(g *dag.DAG, opts *renderOpts) []tower.RenderOption {
	result := []tower.RenderOption{tower.WithGraph(g)}
	if opts.showEdges {
		result = append(result, tower.WithEdges())
	}
	if opts.merge {
		result = append(result, tower.WithMerged())
	}
	if opts.style == styleHanddrawn {
		result = append(result, tower.WithStyle(handdrawn.New(defaultSeed)))
		if opts.nebraska {
			result = append(result, tower.WithNebraska(tower.RankNebraska(g, 5)))
		}
		if opts.popups {
			result = append(result, tower.WithPopups())
		}
	}
	return result
}
