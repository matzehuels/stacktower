package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/internal/cli/term"
	"github.com/matzehuels/stacktower/pkg/infra"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// NewVisualizeCmd creates the visualize command for rendering from a layout.
func NewVisualizeCmd() *cobra.Command {
	var formatsStr string
	opts := DefaultVisualizeCmdOpts()

	cmd := &cobra.Command{
		Use:   "visualize [layout.json]",
		Short: "Render visualization from a computed layout",
		Long: `Render visualization from a computed layout.

The visualize command takes a layout.json file (produced by 'layout') and
renders it to SVG, PNG, or PDF format. The layout contains all positioning
information, so this step is purely about rendering.

Results are cached locally for faster subsequent runs.

Use 'render' as a shortcut to go directly from graph.json to visual output.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Formats = parseFormats(formatsStr)
			if err := pipeline.ValidateFormats(opts.Formats); err != nil {
				return err
			}
			if err := pipeline.ValidateStyle(opts.Style); err != nil {
				return err
			}
			return runVisualize(cmd.Context(), args[0], opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Output, "output", "o", "", "output file (single format) or base path (multiple)")
	cmd.Flags().StringVarP(&formatsStr, "format", "f", "", "output format(s): svg (default), pdf, png (comma-separated)")
	cmd.Flags().BoolVar(&opts.ShowEdges, "edges", false, "show dependency edges (tower)")
	cmd.Flags().StringVar(&opts.Style, "style", opts.Style, "visual style: handdrawn (default), simple (tower)")
	cmd.Flags().BoolVar(&opts.Popups, "popups", opts.Popups, "show hover popups with metadata (tower)")
	cmd.Flags().BoolVar(&opts.NoCache, "no-cache", false, "disable caching")

	return cmd
}

// runVisualize loads the layout and renders it via pipeline.
func runVisualize(ctx context.Context, input string, opts VisualizeCmdOpts) error {
	logger := infra.LoggerFromContext(ctx)

	// Read layout file
	layoutData, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("read layout %s: %w", input, err)
	}

	// Detect viz type from layout
	vizType, err := detectVizType(layoutData)
	if err != nil {
		return fmt.Errorf("detect visualization type: %w", err)
	}

	// Create pipeline service
	svc, cleanup, err := newCLIPipeline(opts.NoCache)
	if err != nil {
		return fmt.Errorf("initialize pipeline: %w", err)
	}
	defer cleanup()

	// Configure pipeline options (opts embeds pipeline.Options)
	opts.VizType = vizType
	opts.Logger = logger

	// Show spinner while rendering
	spinner := term.NewSpinner(fmt.Sprintf("Rendering %s...", vizType))
	spinner.Start()

	// Visualize via pipeline
	artifacts, cacheHit, err := svc.Visualize(ctx, layoutData, nil, opts.Options)
	if err != nil {
		spinner.StopWithError("Visualization failed")
		return fmt.Errorf("visualize: %w", err)
	}
	spinner.Stop()

	// Write outputs (no graph stats available for visualize)
	return writeArtifacts(ArtifactWriteOpts{
		Artifacts: artifacts,
		Formats:   opts.Formats,
		Input:     input,
		Output:    opts.Output,
		CacheHit:  cacheHit,
	})
}

// detectVizType reads the viz_type field from a layout JSON.
func detectVizType(data []byte) (string, error) {
	var header struct {
		VizType string `json:"viz_type"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return "", fmt.Errorf("invalid layout JSON: %w", err)
	}
	if header.VizType == "" {
		return pipeline.VizTypeTower, nil // default
	}
	return header.VizType, nil
}
