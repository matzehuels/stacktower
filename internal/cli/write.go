package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/matzehuels/stacktower/internal/cli/term"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// ArtifactWriteOpts configures artifact file writing.
type ArtifactWriteOpts struct {
	Artifacts map[string][]byte // format -> data
	Formats   []string          // requested formats in order
	Input     string            // input file path (for deriving output names)
	Output    string            // explicit output path (optional)
	NodeCount int               // for stats display
	EdgeCount int               // for stats display
	CacheHit  bool              // for stats display
}

// writeArtifacts writes rendered artifacts to files and prints a summary.
func writeArtifacts(opts ArtifactWriteOpts) error {
	paths, err := writeArtifactFiles(opts.Artifacts, opts.Formats, opts.Input, opts.Output)
	if err != nil {
		return err
	}

	printArtifactSummary(paths, opts.NodeCount, opts.EdgeCount, opts.CacheHit)
	return nil
}

// writeArtifactFiles writes artifacts to disk and returns the written paths.
func writeArtifactFiles(artifacts map[string][]byte, formats []string, input, output string) ([]string, error) {
	base := deriveBasePath(input, output)

	var paths []string
	for _, format := range formats {
		data, ok := artifacts[format]
		if !ok {
			return nil, fmt.Errorf("missing artifact for format: %s", format)
		}

		path := output
		if path == "" || len(formats) > 1 {
			path = base + "." + format
		}

		if err := writeDataToFile(data, path); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}

	return paths, nil
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

// printArtifactSummary displays a styled summary of written artifacts.
func printArtifactSummary(paths []string, nodeCount, edgeCount int, cacheHit bool) {
	summary := term.NewSuccessSummary("Render complete")
	for _, path := range paths {
		summary.AddFile(path)
	}
	summary.Print()
	term.PrintStats(nodeCount, edgeCount, cacheHit)
}

// writeDataToFile writes raw data to the specified path (or stdout if empty).
func writeDataToFile(data []byte, path string) error {
	out, err := openOutput(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = out.Write(data)
	return err
}

// nopCloser wraps an io.Writer with a no-op Close method.
type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }

// openOutput returns a WriteCloser for the given path.
func openOutput(path string) (io.WriteCloser, error) {
	if path == "" {
		return nopCloser{os.Stdout}, nil
	}
	return os.Create(path)
}
