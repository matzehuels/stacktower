// Package io provides serialization and deserialization for tower visualization layouts.
//
// This package is part of the three-stage visualization pipeline:
//
//	pkg/io         → DAG I/O (graph.json)
//	pkg/render/tower/io → Layout I/O (layout.json) ← THIS PACKAGE
//	pkg/render/tower/sink → Visual output (SVG/PNG/PDF)
//
// # Purpose
//
// The layout.json format captures computed block positions, enabling:
//
//   - Separation of layout computation from rendering
//   - Fast re-rendering to different formats without recomputation
//   - Round-trip persistence for reproducible visualizations
//   - Integration with external visualization tools
//
// # JSON Format
//
// The format stores everything needed to render the tower visualization:
//
//	{
//	  "width": 800,
//	  "height": 600,
//	  "margin_x": 40,
//	  "margin_y": 30,
//	  "viz_type": "tower",
//	  "style": "handdrawn",
//	  "seed": 42,
//	  "randomize": true,
//	  "merged": false,
//	  "rows": {"0": ["pkg-a", "pkg-b"], "1": ["pkg-c"]},
//	  "blocks": [
//	    {"id": "pkg-a", "label": "pkg-a", "x": 40, "y": 30, "width": 100, "height": 50, ...}
//	  ],
//	  "edges": [{"from": "pkg-a", "to": "pkg-b"}],
//	  "nebraska": [...]
//	}
//
// # Pipeline Usage
//
// After computing a layout:
//
//	l := layout.Build(g, 800, 600)
//	data, err := towerio.WriteLayout(l, towerio.WithGraph(g), towerio.WithStyle("handdrawn"))
//	storage.Store(ctx, jobID, "layout.json", bytes.NewReader(data))
//
// Before rendering to a format:
//
//	reader, _ := storage.Retrieve(ctx, "job-123/layout.json")
//	l, meta, err := towerio.ReadLayout(reader)
//	svg := sink.RenderSVG(l, sink.WithGraph(dag), ...)
//
// # LayoutMeta
//
// The [ReadLayout] function returns a [LayoutMeta] struct containing render options
// that were stored with the layout. Use these to configure the sink renderer:
//
//	l, meta, _ := towerio.ReadLayout(reader)
//	if meta.Style == "handdrawn" {
//	    opts = append(opts, sink.WithStyle(handdrawn.New(meta.Seed)))
//	}
package io
