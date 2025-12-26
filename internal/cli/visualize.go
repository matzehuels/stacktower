package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/pkg/infra/common"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// visualizeOpts holds the command-line flags for the visualize command.
type visualizeOpts struct {
	output    string   // output file path
	formats   []string // output formats: "svg", "pdf", "png"
	showEdges bool     // draw dependency edges in tower view
	style     string   // visual style: "simple" or "handdrawn"
	popups    bool     // enable hover popups with metadata
	noCache   bool     // disable caching
}

// newVisualizeCmd creates the visualize command for rendering from a layout.
func newVisualizeCmd() *cobra.Command {
	var formatsStr string
	opts := visualizeOpts{
		style:  styleHanddrawn,
		popups: true,
	}

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
			opts.formats = parseFormats(formatsStr)
			if err := validateFormats(opts.formats); err != nil {
				return err
			}
			if err := validateStyle(opts.style); err != nil {
				return err
			}
			return runVisualize(cmd.Context(), args[0], &opts)
		},
	}

	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "output file (single format) or base path (multiple)")
	cmd.Flags().StringVarP(&formatsStr, "format", "f", "", "output format(s): svg (default), pdf, png (comma-separated)")
	cmd.Flags().BoolVar(&opts.showEdges, "edges", false, "show dependency edges (tower)")
	cmd.Flags().StringVar(&opts.style, "style", opts.style, "visual style: handdrawn (default), simple (tower)")
	cmd.Flags().BoolVar(&opts.popups, "popups", opts.popups, "show hover popups with metadata (tower)")
	cmd.Flags().BoolVar(&opts.noCache, "no-cache", false, "disable caching")

	return cmd
}

// runVisualize loads the layout and renders it via pipeline.
func runVisualize(ctx context.Context, input string, opts *visualizeOpts) error {
	logger := common.LoggerFromContext(ctx)

	// Read layout file
	layoutData, err := os.ReadFile(input)
	if err != nil {
		return err
	}

	// Detect viz type from layout
	vizType, err := detectVizType(layoutData)
	if err != nil {
		return err
	}
	logger.Infof("Visualizing %s (%s)", input, vizType)

	// Create pipeline service
	svc, cleanup, err := newPipelineService(opts.noCache, logger)
	if err != nil {
		return err
	}
	defer cleanup()

	// Build pipeline options
	pipelineOpts := pipeline.Options{
		VizType:   vizType,
		Formats:   opts.formats,
		Style:     opts.style,
		ShowEdges: opts.showEdges,
		Popups:    opts.popups,
		Logger:    logger,
	}

	// Visualize via pipeline
	artifacts, cacheHit, err := svc.Visualize(ctx, layoutData, nil, pipelineOpts)
	if err != nil {
		return err
	}

	// Write outputs
	status := formatCacheStatus(cacheHit)
	return writeArtifacts(ctx, artifacts, opts.formats, input, opts.output, status)
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
		return "tower", nil // default
	}
	return header.VizType, nil
}

// writeArtifacts writes all artifacts to files.
func writeArtifacts(ctx context.Context, artifacts map[string][]byte, formats []string, input, output, status string) error {
	logger := common.LoggerFromContext(ctx)

	// Derive base path
	base := output
	if base == "" {
		base = strings.TrimSuffix(input, filepath.Ext(input))
		base = strings.TrimSuffix(base, ".layout")
	}

	// Single format: use output directly if specified
	if len(formats) == 1 {
		format := formats[0]
		data, ok := artifacts[format]
		if !ok {
			return fmt.Errorf("missing artifact for format: %s", format)
		}

		path := output
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
	for _, format := range formats {
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
