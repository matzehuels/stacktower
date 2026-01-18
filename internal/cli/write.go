package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/matzehuels/stacktower/internal/cli/term"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// ArtifactWriteOpts configures artifact file writing.
type ArtifactWriteOpts struct {
	Artifacts map[string][]byte
	Formats   []string
	Input     string // for deriving output names
	Output    string // explicit output path (optional)
	NodeCount int
	EdgeCount int
	CacheHit  bool
}

// writeArtifacts writes rendered artifacts to files and prints a summary.
func writeArtifacts(opts ArtifactWriteOpts) error {
	base := deriveBasePath(opts.Input, opts.Output)
	var paths []string

	for _, format := range opts.Formats {
		data, ok := opts.Artifacts[format]
		if !ok {
			return fmt.Errorf("missing artifact for format: %s", format)
		}

		path := opts.Output
		if path == "" || len(opts.Formats) > 1 {
			path = base + "." + format
		}

		if err := writeDataToFile(data, path); err != nil {
			return err
		}
		paths = append(paths, path)
	}

	// Print summary
	summary := term.NewSuccessSummary("Render complete")
	for _, path := range paths {
		summary.AddFile(path)
	}
	summary.Print()
	term.PrintStats(opts.NodeCount, opts.EdgeCount, opts.CacheHit)
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

// writeDataToFile writes raw data to the specified path (or stdout if empty).
func writeDataToFile(data []byte, path string) error {
	if path == "" {
		_, err := os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(path, data, 0644)
}
