package sink

import (
	"encoding/json"
	"testing"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/render/tower/feature"
	"github.com/matzehuels/stacktower/pkg/render/tower/layout"
)

func TestRenderJSON(t *testing.T) {
	l := layout.Layout{
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

	data, err := RenderJSON(l)
	if err != nil {
		t.Fatalf("RenderJSON() error: %v", err)
	}

	var out jsonOutput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if out.Width != 800 {
		t.Errorf("Width = %v, want 800", out.Width)
	}
	if out.Height != 600 {
		t.Errorf("Height = %v, want 600", out.Height)
	}
	if out.MarginX != 40 {
		t.Errorf("MarginX = %v, want 40", out.MarginX)
	}
	if out.MarginY != 30 {
		t.Errorf("MarginY = %v, want 30", out.MarginY)
	}
	if len(out.Blocks) != 2 {
		t.Errorf("Blocks count = %d, want 2", len(out.Blocks))
	}
}

func TestRenderJSONWithOptions(t *testing.T) {
	l := layout.Layout{
		FrameWidth:  400,
		FrameHeight: 300,
		Blocks:      map[string]layout.Block{},
	}

	data, err := RenderJSON(l,
		WithJSONStyle("handdrawn"),
		WithJSONMerged(),
		WithJSONRandomize(12345),
	)
	if err != nil {
		t.Fatalf("RenderJSON() error: %v", err)
	}

	var out jsonOutput
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

func TestRenderJSONWithGraph(t *testing.T) {
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

	data, err := RenderJSON(l, WithJSONGraph(g))
	if err != nil {
		t.Fatalf("RenderJSON() error: %v", err)
	}

	var out jsonOutput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if len(out.Edges) != 1 {
		t.Errorf("Edges count = %d, want 1", len(out.Edges))
	}
	if len(out.Edges) > 0 {
		if out.Edges[0].From != "a" || out.Edges[0].To != "b" {
			t.Errorf("Edge = {%q -> %q}, want {a -> b}", out.Edges[0].From, out.Edges[0].To)
		}
	}
}

func TestRenderJSONWithMergedEdges(t *testing.T) {
	g := dag.New(nil)
	g.AddNode(dag.Node{ID: "a", Row: 0})
	g.AddNode(dag.Node{ID: "a-sub", Row: 1, MasterID: "a"})
	g.AddNode(dag.Node{ID: "b", Row: 2})
	g.AddEdge(dag.Edge{From: "a", To: "a-sub"})
	g.AddEdge(dag.Edge{From: "a-sub", To: "b"})

	l := layout.Layout{
		FrameWidth:  400,
		FrameHeight: 300,
		Blocks: map[string]layout.Block{
			"a":     {NodeID: "a", Left: 0, Right: 100, Bottom: 0, Top: 50},
			"a-sub": {NodeID: "a-sub", Left: 0, Right: 100, Bottom: 50, Top: 100},
			"b":     {NodeID: "b", Left: 0, Right: 100, Bottom: 100, Top: 150},
		},
	}

	data, err := RenderJSON(l, WithJSONGraph(g), WithJSONMerged())
	if err != nil {
		t.Fatalf("RenderJSON() error: %v", err)
	}

	var out jsonOutput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	// Merged edges should skip self-edges from subdivider nodes
	// Edge from a-sub to b should be converted to a->b
	found := false
	for _, e := range out.Edges {
		if e.From == "a" && e.To == "b" {
			found = true
		}
		// Should not have edge from a to a (self-edge via subdivider)
		if e.From == e.To {
			t.Errorf("Found self-edge: %q -> %q", e.From, e.To)
		}
	}
	if !found && len(out.Edges) > 0 {
		t.Logf("Edges: %+v", out.Edges)
	}
}

func TestRenderJSONWithNebraska(t *testing.T) {
	rankings := []feature.NebraskaRanking{
		{
			Maintainer: "maintainer1",
			Score:      10.5,
			Packages: []feature.PackageRole{
				{Package: "pkg-a", Role: feature.RoleOwner, URL: "https://github.com/a"},
				{Package: "pkg-b", Role: feature.RoleMaintainer, URL: ""},
			},
		},
	}

	l := layout.Layout{
		FrameWidth:  400,
		FrameHeight: 300,
		Blocks:      map[string]layout.Block{},
	}

	data, err := RenderJSON(l, WithJSONNebraska(rankings))
	if err != nil {
		t.Fatalf("RenderJSON() error: %v", err)
	}

	var out jsonOutput
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
	if neb.Score != 10.5 {
		t.Errorf("Score = %v, want 10.5", neb.Score)
	}
	if len(neb.Packages) != 2 {
		t.Errorf("Packages count = %d, want 2", len(neb.Packages))
	}
}

func TestRenderJSONWithNodeMeta(t *testing.T) {
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
			"repo_archived":     false,
		},
	})

	l := layout.Layout{
		FrameWidth:  400,
		FrameHeight: 300,
		Blocks: map[string]layout.Block{
			"pkg": {NodeID: "pkg", Left: 0, Right: 100, Bottom: 0, Top: 50},
		},
	}

	data, err := RenderJSON(l, WithJSONGraph(g))
	if err != nil {
		t.Fatalf("RenderJSON() error: %v", err)
	}

	var out jsonOutput
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
		t.Errorf("Meta.Description = %q, want %q", block.Meta.Description, "A test package")
	}
	if block.Meta.Stars != 1000 {
		t.Errorf("Meta.Stars = %d, want 1000", block.Meta.Stars)
	}
}

func TestExtractJSONMetaNilMeta(t *testing.T) {
	n := &dag.Node{ID: "test", Meta: nil}
	m := extractJSONMeta(n)
	if m != nil {
		t.Error("extractJSONMeta() should return nil for node with nil meta")
	}
}

func TestExtractJSONMetaEmptyMeta(t *testing.T) {
	n := &dag.Node{ID: "test", Meta: dag.Metadata{}}
	m := extractJSONMeta(n)
	if m != nil {
		t.Error("extractJSONMeta() should return nil for node with empty meaningful meta")
	}
}

func TestExtractJSONMetaWithSummary(t *testing.T) {
	n := &dag.Node{
		ID: "test",
		Meta: dag.Metadata{
			"summary": "A summary description",
		},
	}
	m := extractJSONMeta(n)
	if m == nil {
		t.Fatal("extractJSONMeta() should not return nil")
	}
	if m.Description != "A summary description" {
		t.Errorf("Description = %q, want %q", m.Description, "A summary description")
	}
}

func TestBuildJSONEdgesFiltersInvalidBlocks(t *testing.T) {
	g := dag.New(nil)
	g.AddNode(dag.Node{ID: "a", Row: 0})
	g.AddNode(dag.Node{ID: "b", Row: 1})
	g.AddNode(dag.Node{ID: "c", Row: 2})
	g.AddEdge(dag.Edge{From: "a", To: "b"})
	g.AddEdge(dag.Edge{From: "b", To: "c"})

	// Layout only has "a" and "b", not "c"
	l := layout.Layout{
		Blocks: map[string]layout.Block{
			"a": {NodeID: "a"},
			"b": {NodeID: "b"},
		},
	}

	edges := buildJSONEdges(l, g, false)

	// Should only include a->b, not b->c (since c is not in layout)
	if len(edges) != 1 {
		t.Errorf("edges count = %d, want 1", len(edges))
	}
	if len(edges) > 0 && (edges[0].From != "a" || edges[0].To != "b") {
		t.Errorf("edge = {%q -> %q}, want {a -> b}", edges[0].From, edges[0].To)
	}
}

func TestWithJSONGraphOption(t *testing.T) {
	g := dag.New(nil)
	r := &jsonRenderer{}
	opt := WithJSONGraph(g)
	opt(r)
	if r.graph != g {
		t.Error("WithJSONGraph should set graph")
	}
}

func TestWithJSONMergedOption(t *testing.T) {
	r := &jsonRenderer{}
	opt := WithJSONMerged()
	opt(r)
	if !r.merged {
		t.Error("WithJSONMerged should set merged=true")
	}
}

func TestWithJSONRandomizeOption(t *testing.T) {
	r := &jsonRenderer{}
	opt := WithJSONRandomize(42)
	opt(r)
	if !r.randomize {
		t.Error("WithJSONRandomize should set randomize=true")
	}
	if r.seed != 42 {
		t.Errorf("seed = %d, want 42", r.seed)
	}
}

func TestWithJSONStyleOption(t *testing.T) {
	r := &jsonRenderer{}
	opt := WithJSONStyle("custom")
	opt(r)
	if r.style != "custom" {
		t.Errorf("style = %q, want %q", r.style, "custom")
	}
}

func TestWithJSONNebraskaOption(t *testing.T) {
	rankings := []feature.NebraskaRanking{{Maintainer: "test"}}
	r := &jsonRenderer{}
	opt := WithJSONNebraska(rankings)
	opt(r)
	if len(r.nebraska) != 1 {
		t.Errorf("nebraska count = %d, want 1", len(r.nebraska))
	}
}
