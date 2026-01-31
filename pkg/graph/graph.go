package graph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/matzehuels/stacktower/pkg/core/dag"
)

// =============================================================================
// Graph Serialization API
// =============================================================================

// MarshalGraph converts a DAG to JSON bytes.
// Nodes are sorted by ID for deterministic output.
func MarshalGraph(g *dag.DAG) ([]byte, error) {
	var buf bytes.Buffer
	if err := writeGraphTo(g, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteGraphFile writes a DAG to a JSON file.
// The file is created with 0644 permissions.
func WriteGraphFile(g *dag.DAG, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()
	return writeGraphTo(g, f)
}

// WriteGraph writes a DAG as JSON to an io.Writer.
// Use MarshalGraph for in-memory serialization or WriteGraphFile for files.
func WriteGraph(g *dag.DAG, w io.Writer) error {
	return writeGraphTo(g, w)
}

// ReadGraphFile reads a JSON file and returns the decoded DAG.
// Returns validation errors for malformed graphs or DAG constraint violations.
func ReadGraphFile(path string) (*dag.DAG, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	return readGraphFrom(f)
}

// ReadGraph decodes a JSON graph from an io.Reader into a DAG.
// Use ReadGraphFile for files or pass bytes.NewReader for in-memory data.
func ReadGraph(r io.Reader) (*dag.DAG, error) {
	return readGraphFrom(r)
}

// =============================================================================
// Internal Implementation
// =============================================================================

func writeGraphTo(g *dag.DAG, w io.Writer) error {
	out := FromDAG(g)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	return nil
}

func readGraphFrom(r io.Reader) (*dag.DAG, error) {
	var data Graph
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return ToDAG(data)
}
