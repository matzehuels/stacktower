// Package jobs defines the shared job payload types for the Stacktower API and Worker.
//
// This package is internal because it's a contract between the API and Worker
// components, not a public API for external consumption.
//
// The job system follows a three-stage pipeline:
//
//	PARSE → LAYOUT → VISUALIZE
//
// Each stage produces a stored artifact that can be used by subsequent stages:
//
//	Parse:     manifest/package → graph.json (DAG with nodes, edges, metadata)
//	Layout:    graph.json → layout.json (computed positions - the durable artifact)
//	Visualize: layout.json → svg/png/pdf (final visual output)
//
// # Job Types
//
// Individual stages:
//   - parse: Parse a package or manifest file into a dependency graph
//   - layout: Compute visualization layout from a graph
//   - visualize: Generate visual output from a layout
//
// Combined pipelines:
//   - render: Full pipeline (parse + layout + visualize) or shortcut (layout + visualize)
//
// # Layout as the Durable Artifact
//
// The layout.json is the key cacheable artifact. It fully determines the
// visualization and can be re-rendered with different styles without
// recomputing positions. For tower visualizations, it contains block
// positions and dimensions. For nodelink, it contains the DOT representation.
//
// # Visualization Types
//
// The system supports multiple visualization styles via the viz_type parameter:
//
//   - tower: Stacked block visualization (default)
//   - nodelink: Traditional node-and-edge graph
//
// Each viz_type has its own layout format and rendering logic.
//
// # Usage
//
// Submit a parse job:
//
//	payload := jobs.ParsePayload{
//	    Language: "python",
//	    Package:  "requests",
//	}
//
// Submit a layout job using the parse result:
//
//	payload := jobs.LayoutPayload{
//	    GraphPath: "job-123/graph.json",
//	    VizType:   "tower",
//	}
//
// Submit a visualize job using the layout result:
//
//	payload := jobs.VisualizePayload{
//	    LayoutPath: "job-123/layout.json",
//	    Formats:    []string{"svg", "png"},
//	}
//
// Or use render for the full pipeline:
//
//	payload := jobs.RenderPayload{
//	    Language: "python",
//	    Package:  "requests",
//	    VizType:  "tower",
//	    Formats:  []string{"svg"},
//	}
package jobs
