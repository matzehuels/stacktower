package io

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/render/tower/feature"
	"github.com/matzehuels/stacktower/pkg/render/tower/layout"
)

func TestWriteLayout(t *testing.T) {
	l := layout.Layout{
		FrameWidth:  800,
		FrameHeight: 600,
		MarginX:     40,
		MarginY:     30,
		Blocks: map[string]layout.Block{
			"pkg-a": {NodeID: "pkg-a", Left: 40, Right: 200, Bottom: 30, Top: 100},
		},
		RowOrders: map[int][]string{
			0: {"pkg-a"},
		},
	}

	data, err := WriteLayout(l)
	if err != nil {
		t.Fatalf("WriteLayout() error: %v", err)
	}

	var out LayoutData
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if out.Width != 800 {
		t.Errorf("Width = %v, want 800", out.Width)
	}
	if out.Height != 600 {
		t.Errorf("Height = %v, want 600", out.Height)
	}
}

func TestWriteLayoutWithOptions(t *testing.T) {
	l := layout.Layout{
		FrameWidth:  400,
		FrameHeight: 300,
		Blocks:      map[string]layout.Block{},
	}

	data, err := WriteLayout(l,
		WithStyle("handdrawn"),
		WithMerged(),
		WithRandomize(12345),
	)
	if err != nil {
		t.Fatalf("WriteLayout() error: %v", err)
	}

	var out LayoutData
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if out.Style != "handdrawn" {
		t.Errorf("Style = %q, want %q", out.Style, "handdrawn")
	}
	if !out.Merged {
		t.Error("Merged should be true")
	}
	if !out.Randomize {
		t.Error("Randomize should be true")
	}
	if out.Seed != 12345 {
		t.Errorf("Seed = %d, want 12345", out.Seed)
	}
}

func TestWriteLayoutWithGraph(t *testing.T) {
	g := dag.New(nil)
	g.AddNode(dag.Node{ID: "a", Row: 0})
	g.AddNode(dag.Node{ID: "b", Row: 1})
	g.AddEdge(dag.Edge{From: "a", To: "b"})

	l := layout.Layout{
		FrameWidth:  400,
		FrameHeight: 300,
		Blocks: map[string]layout.Block{
			"a": {NodeID: "a", Left: 0, Right: 100, Bottom: 0, Top: 50},
			"b": {NodeID: "b", Left: 0, Right: 100, Bottom: 50, Top: 100},
		},
	}

	data, err := WriteLayout(l, WithGraph(g))
	if err != nil {
		t.Fatalf("WriteLayout() error: %v", err)
	}

	var out LayoutData
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if len(out.Edges) != 1 {
		t.Errorf("Edges count = %d, want 1", len(out.Edges))
	}
}

func TestReadLayout(t *testing.T) {
	input := LayoutData{
		Width:     800,
		Height:    600,
		MarginX:   40,
		MarginY:   30,
		Style:     "handdrawn",
		Seed:      42,
		Randomize: true,
		Merged:    true,
		Blocks: []BlockData{
			{ID: "pkg-a", Label: "pkg-a", X: 40, Y: 30, Width: 160, Height: 70},
			{ID: "pkg-b", Label: "pkg-b", X: 200, Y: 100, Width: 200, Height: 100},
		},
		Rows: map[int][]string{
			0: {"pkg-a"},
			1: {"pkg-b"},
		},
	}

	data, _ := json.Marshal(input)
	l, meta, err := ReadLayout(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ReadLayout() error: %v", err)
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
	if meta.Style != "handdrawn" {
		t.Errorf("meta.Style = %q, want %q", meta.Style, "handdrawn")
	}
	if meta.Seed != 42 {
		t.Errorf("meta.Seed = %d, want 42", meta.Seed)
	}
	if !meta.Randomize {
		t.Error("meta.Randomize should be true")
	}
	if !meta.Merged {
		t.Error("meta.Merged should be true")
	}
}

func TestRoundTrip(t *testing.T) {
	original := layout.Layout{
		FrameWidth:  800,
		FrameHeight: 600,
		MarginX:     40,
		MarginY:     30,
		Blocks: map[string]layout.Block{
			"pkg-a": {NodeID: "pkg-a", Left: 40, Right: 200, Bottom: 30, Top: 100},
			"pkg-b": {NodeID: "pkg-b", Left: 200, Right: 400, Bottom: 100, Top: 200},
		},
		RowOrders: map[int][]string{
			0: {"pkg-a"},
			1: {"pkg-b"},
		},
	}

	// Export
	data, err := WriteLayout(original, WithStyle("handdrawn"), WithRandomize(42))
	if err != nil {
		t.Fatalf("WriteLayout() error: %v", err)
	}

	// Import
	imported, meta, err := ReadLayout(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ReadLayout() error: %v", err)
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
		if impBlock.Right != origBlock.Right {
			t.Errorf("Block %q Right = %v, want %v", id, impBlock.Right, origBlock.Right)
		}
		if impBlock.Bottom != origBlock.Bottom {
			t.Errorf("Block %q Bottom = %v, want %v", id, impBlock.Bottom, origBlock.Bottom)
		}
		if impBlock.Top != origBlock.Top {
			t.Errorf("Block %q Top = %v, want %v", id, impBlock.Top, origBlock.Top)
		}
	}

	// Verify row orders
	if len(imported.RowOrders) != len(original.RowOrders) {
		t.Errorf("RowOrders count = %d, want %d", len(imported.RowOrders), len(original.RowOrders))
	}

	// Verify metadata
	if meta.Style != "handdrawn" {
		t.Errorf("meta.Style = %q, want %q", meta.Style, "handdrawn")
	}
	if meta.Seed != 42 {
		t.Errorf("meta.Seed = %d, want 42", meta.Seed)
	}
}

func TestWriteLayoutWithNebraska(t *testing.T) {
	rankings := []feature.NebraskaRanking{
		{
			Maintainer: "maintainer1",
			Score:      10.5,
			Packages: []feature.PackageRole{
				{Package: "pkg-a", Role: feature.RoleOwner, URL: "https://github.com/a"},
			},
		},
	}

	l := layout.Layout{
		FrameWidth:  400,
		FrameHeight: 300,
		Blocks:      map[string]layout.Block{},
	}

	data, err := WriteLayout(l, WithNebraska(rankings))
	if err != nil {
		t.Fatalf("WriteLayout() error: %v", err)
	}

	var out LayoutData
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if len(out.Nebraska) != 1 {
		t.Fatalf("Nebraska count = %d, want 1", len(out.Nebraska))
	}

	neb := out.Nebraska[0]
	if neb.Maintainer != "maintainer1" {
		t.Errorf("Maintainer = %q, want %q", neb.Maintainer, "maintainer1")
	}
}

func TestReadLayoutWithNebraska(t *testing.T) {
	input := LayoutData{
		Width:  400,
		Height: 300,
		Blocks: []BlockData{},
		Nebraska: []NebraskaData{
			{
				Maintainer: "maintainer1",
				Score:      10.5,
				Packages: []NebraskaPackage{
					{Package: "pkg-a", Role: "owner", URL: "https://github.com/a"},
				},
			},
		},
	}

	data, _ := json.Marshal(input)
	_, meta, err := ReadLayout(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ReadLayout() error: %v", err)
	}

	if len(meta.Nebraska) != 1 {
		t.Fatalf("Nebraska count = %d, want 1", len(meta.Nebraska))
	}

	if meta.Nebraska[0].Maintainer != "maintainer1" {
		t.Errorf("Maintainer = %q, want %q", meta.Nebraska[0].Maintainer, "maintainer1")
	}
	if meta.Nebraska[0].Score != 10.5 {
		t.Errorf("Score = %v, want 10.5", meta.Nebraska[0].Score)
	}
	if len(meta.Nebraska[0].Packages) != 1 {
		t.Errorf("Packages count = %d, want 1", len(meta.Nebraska[0].Packages))
	}
	if meta.Nebraska[0].Packages[0].Role != feature.RoleOwner {
		t.Errorf("Role = %q, want %q", meta.Nebraska[0].Packages[0].Role, feature.RoleOwner)
	}
}

func TestWriteLayoutWithBlockMeta(t *testing.T) {
	g := dag.New(nil)
	g.AddNode(dag.Node{
		ID:  "pkg",
		Row: 0,
		Meta: dag.Metadata{
			"description":       "A test package",
			"repo_url":          "https://github.com/test/pkg",
			"repo_stars":        1000,
			"repo_last_commit":  "2024-01-01",
			"repo_last_release": "v1.0.0",
		},
	})

	l := layout.Layout{
		FrameWidth:  400,
		FrameHeight: 300,
		Blocks: map[string]layout.Block{
			"pkg": {NodeID: "pkg", Left: 0, Right: 100, Bottom: 0, Top: 50},
		},
	}

	data, err := WriteLayout(l, WithGraph(g))
	if err != nil {
		t.Fatalf("WriteLayout() error: %v", err)
	}

	var out LayoutData
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if len(out.Blocks) != 1 {
		t.Fatalf("Blocks count = %d, want 1", len(out.Blocks))
	}

	block := out.Blocks[0]
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
	input := LayoutData{
		Width:  400,
		Height: 300,
		Blocks: []BlockData{
			{ID: "a", Label: "Package A", URL: "https://example.com/a"},
			{ID: "b", Label: "Package B", Meta: &BlockMeta{Stars: 100}},
		},
	}

	data, _ := json.Marshal(input)
	blocks, err := ReadBlocksData(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ReadBlocksData() error: %v", err)
	}

	if len(blocks) != 2 {
		t.Fatalf("Blocks count = %d, want 2", len(blocks))
	}

	if blocks[0].ID != "a" || blocks[0].URL != "https://example.com/a" {
		t.Errorf("Block 0 = %+v, unexpected", blocks[0])
	}
	if blocks[1].ID != "b" || blocks[1].Meta == nil || blocks[1].Meta.Stars != 100 {
		t.Errorf("Block 1 = %+v, unexpected", blocks[1])
	}
}
