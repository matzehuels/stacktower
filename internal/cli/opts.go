package cli

import "github.com/matzehuels/stacktower/pkg/pipeline"

// =============================================================================
// CLI Command Options
// Each command struct embeds pipeline.Options and adds CLI-specific fields.
// This eliminates duplication between CLI and API while keeping CLI concerns separate.
// =============================================================================

// CLIOpts contains CLI-specific options shared across commands.
type CLIOpts struct {
	Output       string // output file path (stdout if empty)
	NoCache      bool   // disable caching
	OrderTimeout int    // timeout in seconds for optimal ordering search (CLI-specific)
}

// DefaultOrderTimeout is the default timeout for optimal ordering search.
// 60 seconds provides enough time for most dependency graphs (<100 nodes)
// to find an optimal or near-optimal ordering while keeping the CLI responsive.
// Users can increase this via --ordering-timeout for larger graphs.
const DefaultOrderTimeout = 60

// DefaultCLIOpts returns CLIOpts with sensible defaults.
func DefaultCLIOpts() CLIOpts {
	return CLIOpts{
		OrderTimeout: DefaultOrderTimeout,
	}
}

// ParseCmdOpts combines pipeline options with CLI-specific options for parsing.
type ParseCmdOpts struct {
	pipeline.Options
	CLIOpts
	Name string // override project name for manifest parsing (CLI-specific)
}

// DefaultParseCmdOpts returns ParseCmdOpts with sensible defaults.
func DefaultParseCmdOpts() ParseCmdOpts {
	opts := ParseCmdOpts{
		CLIOpts: DefaultCLIOpts(),
	}
	// Set parse-specific defaults
	opts.MaxDepth = pipeline.DefaultMaxDepth
	opts.MaxNodes = pipeline.DefaultMaxNodes
	opts.Enrich = true
	opts.Normalize = true
	return opts
}

// LayoutCmdOpts combines pipeline options with CLI-specific options for layout.
type LayoutCmdOpts struct {
	pipeline.Options
	CLIOpts
}

// DefaultLayoutCmdOpts returns LayoutCmdOpts with sensible defaults.
func DefaultLayoutCmdOpts() LayoutCmdOpts {
	opts := LayoutCmdOpts{
		CLIOpts: DefaultCLIOpts(),
	}
	// Set layout-specific defaults
	opts.VizType = pipeline.DefaultVizType
	opts.Width = pipeline.DefaultWidth
	opts.Height = pipeline.DefaultHeight
	opts.Ordering = pipeline.DefaultOrdering
	opts.Seed = pipeline.DefaultSeed
	opts.Randomize = true
	opts.Merge = true
	opts.Normalize = true
	return opts
}

// NeedsOptimalOrderer returns true if the ordering algorithm requires the optimal orderer.
func (o *LayoutCmdOpts) NeedsOptimalOrderer() bool {
	return o.Ordering == pipeline.DefaultOrdering || o.Ordering == ""
}

// VisualizeCmdOpts combines pipeline options with CLI-specific options for visualization.
type VisualizeCmdOpts struct {
	pipeline.Options
	CLIOpts
}

// DefaultVisualizeCmdOpts returns VisualizeCmdOpts with sensible defaults.
func DefaultVisualizeCmdOpts() VisualizeCmdOpts {
	opts := VisualizeCmdOpts{
		CLIOpts: DefaultCLIOpts(),
	}
	// Set visualize-specific defaults
	opts.Style = pipeline.DefaultStyle
	opts.Popups = true
	return opts
}

// RenderCmdOpts combines pipeline options with CLI-specific options for full render.
type RenderCmdOpts struct {
	pipeline.Options
	CLIOpts
}

// DefaultRenderCmdOpts returns RenderCmdOpts with sensible defaults.
func DefaultRenderCmdOpts() RenderCmdOpts {
	opts := RenderCmdOpts{
		CLIOpts: DefaultCLIOpts(),
	}
	// Set layout defaults
	opts.VizType = pipeline.DefaultVizType
	opts.Width = pipeline.DefaultWidth
	opts.Height = pipeline.DefaultHeight
	opts.Ordering = pipeline.DefaultOrdering
	opts.Seed = pipeline.DefaultSeed
	opts.Randomize = true
	opts.Merge = true
	opts.Normalize = true
	// Set render defaults
	opts.Style = pipeline.DefaultStyle
	opts.Popups = true
	return opts
}

// NeedsOptimalOrderer returns true if the ordering algorithm requires the optimal orderer.
func (o *RenderCmdOpts) NeedsOptimalOrderer() bool {
	return o.Ordering == pipeline.DefaultOrdering || o.Ordering == ""
}
