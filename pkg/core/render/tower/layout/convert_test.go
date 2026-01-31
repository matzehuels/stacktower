package layout

import (
	"encoding/json"
	"testing"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/feature"
	"github.com/matzehuels/stacktower/pkg/graph"
)

func TestExport(t *testing.T) {
	l := Layout{
		FrameWidth:  800,
		FrameHeight: 600,
		MarginX:     40,
		MarginY:     30,
		Blocks: map[string]Block{
			"pkg-a": {NodeID: "pkg-a", Left: 40, Right: 200, Bottom: 30, Top: 100},
		},
		RowOrders: map[int][]string{
			0: {"pkg-a"},
		},
	}

	exported, err := l.Export(nil)
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}

	if exported.Width != 800 {
		t.Errorf("Width = %v, want 800", exported.Width)
	}
	if exported.Height != 600 {
		t.Errorf("Height = %v, want 600", exported.Height)
	}
	if len(exported.Blocks) != 1 {
		t.Errorf("Blocks count = %d, want 1", len(exported.Blocks))
	}
}

func TestExportWithOptions(t *testing.T) {
	l := Layout{
		FrameWidth:  400,
		FrameHeight: 300,
		Blocks:      map[string]Block{},
		Style:       "handdrawn",
		Merged:      true,
		Randomize:   true,
		Seed:        12345,
	}

	exported, err := l.Export(nil)
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}

	if exported.Style != "handdrawn" {
		t.Errorf("Style = %q, want %q", exported.Style, "handdrawn")
	}
	if !exported.Merged {
		t.Error("Merged should be true")
	}
	if !exported.Randomize {
		t.Error("Randomize should be true")
	}
	if exported.Seed != 12345 {
		t.Errorf("Seed = %d, want 12345", exported.Seed)
	}
}

func TestExportWithGraph(t *testing.T) {
	g := dag.New(nil)
	g.AddNode(dag.Node{ID: "a", Row: 0})
	g.AddNode(dag.Node{ID: "b", Row: 1})
	g.AddEdge(dag.Edge{From: "a", To: "b"})

	l := Layout{
		FrameWidth:  400,
		FrameHeight: 300,
		Blocks: map[string]Block{
			"a": {NodeID: "a", Left: 0, Right: 100, Bottom: 0, Top: 50},
			"b": {NodeID: "b", Left: 0, Right: 100, Bottom: 50, Top: 100},
		},
	}

	exported, err := l.Export(g)
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}

	if len(exported.Edges) != 1 {
		t.Errorf("Edges count = %d, want 1", len(exported.Edges))
	}
}

func TestParse(t *testing.T) {
	input := graph.Layout{
		VizType:   graph.VizTypeTower,
		Width:     800,
		Height:    600,
		MarginX:   40,
		MarginY:   30,
		Style:     "handdrawn",
		Seed:      42,
		Randomize: true,
		Merged:    true,
		Blocks: []graph.Block{
			{ID: "pkg-a", Label: "pkg-a", X: 40, Y: 30, Width: 160, Height: 70},
			{ID: "pkg-b", Label: "pkg-b", X: 200, Y: 100, Width: 200, Height: 100},
		},
		Rows: map[int][]string{
			0: {"pkg-a"},
			1: {"pkg-b"},
		},
	}

	l, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	// Check layout
	if l.FrameWidth != 800 {
		t.Errorf("FrameWidth = %v, want 800", l.FrameWidth)
	}
	if l.FrameHeight != 600 {
		t.Errorf("FrameHeight = %v, want 600", l.FrameHeight)
	}
	if len(l.Blocks) != 2 {
		t.Errorf("Blocks count = %d, want 2", len(l.Blocks))
	}

	// Check block reconstruction
	block, ok := l.Blocks["pkg-a"]
	if !ok {
		t.Fatal("Block pkg-a not found")
	}
	if block.Left != 40 {
		t.Errorf("block.Left = %v, want 40", block.Left)
	}
	if block.Right != 200 { // X + Width = 40 + 160
		t.Errorf("block.Right = %v, want 200", block.Right)
	}
	if block.Bottom != 30 {
		t.Errorf("block.Bottom = %v, want 30", block.Bottom)
	}
	if block.Top != 100 { // Y + Height = 30 + 70
		t.Errorf("block.Top = %v, want 100", block.Top)
	}

	// Check metadata
	if l.Style != "handdrawn" {
		t.Errorf("l.Style = %q, want %q", l.Style, "handdrawn")
	}
	if l.Seed != 42 {
		t.Errorf("l.Seed = %d, want 42", l.Seed)
	}
	if !l.Randomize {
		t.Error("l.Randomize should be true")
	}
	if !l.Merged {
		t.Error("l.Merged should be true")
	}
}

func TestRoundTrip(t *testing.T) {
	original := Layout{
		FrameWidth:  800,
		FrameHeight: 600,
		MarginX:     40,
		MarginY:     30,
		Blocks: map[string]Block{
			"pkg-a": {NodeID: "pkg-a", Left: 40, Right: 200, Bottom: 30, Top: 100},
			"pkg-b": {NodeID: "pkg-b", Left: 200, Right: 400, Bottom: 100, Top: 200},
		},
		RowOrders: map[int][]string{
			0: {"pkg-a"},
			1: {"pkg-b"},
		},
		Style:     "handdrawn",
		Seed:      42,
		Randomize: true,
	}

	// Export
	exported, err := original.Export(nil)
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}

	// Parse back
	imported, err := Parse(exported)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	// Verify dimensions
	if imported.FrameWidth != original.FrameWidth {
		t.Errorf("FrameWidth = %v, want %v", imported.FrameWidth, original.FrameWidth)
	}
	if imported.FrameHeight != original.FrameHeight {
		t.Errorf("FrameHeight = %v, want %v", imported.FrameHeight, original.FrameHeight)
	}
	if imported.MarginX != original.MarginX {
		t.Errorf("MarginX = %v, want %v", imported.MarginX, original.MarginX)
	}
	if imported.MarginY != original.MarginY {
		t.Errorf("MarginY = %v, want %v", imported.MarginY, original.MarginY)
	}

	// Verify blocks
	if len(imported.Blocks) != len(original.Blocks) {
		t.Errorf("Blocks count = %d, want %d", len(imported.Blocks), len(original.Blocks))
	}

	for id, origBlock := range original.Blocks {
		impBlock, ok := imported.Blocks[id]
		if !ok {
			t.Errorf("Block %q not found in imported layout", id)
			continue
		}
		if impBlock.Left != origBlock.Left {
			t.Errorf("Block %q Left = %v, want %v", id, impBlock.Left, origBlock.Left)
		}
	}

	// Verify metadata
	if imported.Style != "handdrawn" {
		t.Errorf("imported.Style = %q, want %q", imported.Style, "handdrawn")
	}
	if imported.Seed != 42 {
		t.Errorf("imported.Seed = %d, want 42", imported.Seed)
	}
}

func TestExportWithNebraska(t *testing.T) {
	rankings := []feature.NebraskaRanking{
		{
			Maintainer: "maintainer1",
			Score:      10.5,
			Packages: []feature.PackageRole{
				{Package: "pkg-a", Role: feature.RoleOwner, URL: "https://github.com/a"},
			},
		},
	}

	l := Layout{
		FrameWidth:  400,
		FrameHeight: 300,
		Blocks:      map[string]Block{},
		Nebraska:    rankings,
	}

	exported, err := l.Export(nil)
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}

	if len(exported.Nebraska) != 1 {
		t.Fatalf("Nebraska count = %d, want 1", len(exported.Nebraska))
	}

	neb := exported.Nebraska[0]
	if neb.Maintainer != "maintainer1" {
		t.Errorf("Maintainer = %q, want %q", neb.Maintainer, "maintainer1")
	}
}

func TestParseWithNebraska(t *testing.T) {
	input := graph.Layout{
		VizType: graph.VizTypeTower,
		Width:   400,
		Height:  300,
		Blocks:  []graph.Block{},
		Nebraska: []graph.NebraskaRanking{
			{
				Maintainer: "maintainer1",
				Score:      10.5,
				Packages: []graph.NebraskaPackage{
					{Package: "pkg-a", Role: "owner", URL: "https://github.com/a"},
				},
			},
		},
	}

	l, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if len(l.Nebraska) != 1 {
		t.Fatalf("Nebraska count = %d, want 1", len(l.Nebraska))
	}

	if l.Nebraska[0].Maintainer != "maintainer1" {
		t.Errorf("Maintainer = %q, want %q", l.Nebraska[0].Maintainer, "maintainer1")
	}
}

func TestExportWithBlockMeta(t *testing.T) {
	g := dag.New(nil)
	g.AddNode(dag.Node{
		ID:  "pkg",
		Row: 0,
		Meta: dag.Metadata{
			"repo_description":  "A test package",
			"repo_url":          "https://github.com/test/pkg",
			"repo_stars":        1000,
			"repo_last_commit":  "2024-01-01",
			"repo_last_release": "v1.0.0",
		},
	})

	l := Layout{
		FrameWidth:  400,
		FrameHeight: 300,
		Blocks: map[string]Block{
			"pkg": {NodeID: "pkg", Left: 0, Right: 100, Bottom: 0, Top: 50},
		},
	}

	exported, err := l.Export(g)
	if err != nil {
		t.Fatalf("Export() error: %v", err)
	}

	if len(exported.Blocks) != 1 {
		t.Fatalf("Blocks count = %d, want 1", len(exported.Blocks))
	}

	block := exported.Blocks[0]
	if block.URL != "https://github.com/test/pkg" {
		t.Errorf("URL = %q, want %q", block.URL, "https://github.com/test/pkg")
	}
	if block.Meta == nil {
		t.Fatal("Meta should not be nil")
	}
	if block.Meta.Description != "A test package" {
		t.Errorf("Description = %q, want %q", block.Meta.Description, "A test package")
	}
	if block.Meta.Stars != 1000 {
		t.Errorf("Stars = %d, want 1000", block.Meta.Stars)
	}
}

func TestReadBlocksData(t *testing.T) {
	// This test verifies JSON unmarshaling into graph.Layout for blocks.
	input := graph.Layout{
		Width:  400,
		Height: 300,
		Blocks: []graph.Block{
			{ID: "a", Label: "Package A", URL: "https://example.com/a"},
			{ID: "b", Label: "Package B", Meta: &graph.BlockMeta{Stars: 100}},
		},
	}

	data, _ := json.Marshal(input)
	var out graph.Layout
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	blocks := out.Blocks
	if len(blocks) != 2 {
		t.Fatalf("Blocks count = %d, want 2", len(blocks))
	}

	if blocks[0].ID != "a" || blocks[0].URL != "https://example.com/a" {
		t.Errorf("Block 0 = %+v, unexpected", blocks[0])
	}
}
