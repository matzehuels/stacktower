package cli

import "github.com/matzehuels/stacktower/pkg/pipeline"

// =============================================================================
// CLI Command Options - Embedding Pattern
// =============================================================================
//
// This file defines option structs for CLI commands using Go's struct embedding.
// The pattern enables code sharing between CLI and API while keeping concerns separate.
//
// # Architecture
//
// Each command-specific options struct embeds two types:
//   - pipeline.Options: Core rendering options shared with API (pkg/pipeline/pipeline.go)
//   - CommonOptions: CLI-specific options (output paths, caching, timeouts)
//
// Example:
//
//	type RenderCmdOpts struct {
//	    pipeline.Options  // Shared: Language, VizType, Formats, etc.
//	    CommonOptions     // CLI-only: Output, NoCache, OrderTimeout
//	}
//
// # Benefits
//
//  1. Single source of truth: pipeline.Options defines all rendering params once
//  2. CLI flexibility: CommonOptions adds CLI-specific behavior without polluting API
//  3. Type safety: Each command gets its own typed struct with appropriate defaults
//  4. Flag binding: Cobra flags can bind directly to embedded fields (opts.MaxDepth)
//
// # Adding a New Command
//
//  1. Create a new *CmdOpts struct embedding pipeline.Options and CommonOptions
//  2. Create a Default*CmdOpts() function that sets appropriate defaults
//  3. In cmd_*.go, use the opts struct with Cobra flag binding
//
// # Default Values
//
// Pipeline defaults are defined in pkg/pipeline/pipeline.go (DefaultMaxDepth, etc.)
// CLI defaults call those constants to stay in sync.
// =============================================================================

// CommonOptions contains CLI-specific options shared across commands.
// These options are distinct from pipeline.Options and handle CLI concerns
// like output paths, caching preferences, and timeout configuration.
type CommonOptions struct {
	Output       string // output file path (stdout if empty)
	NoCache      bool   // disable caching
	OrderTimeout int    // timeout in seconds for optimal ordering search
}

// DefaultOrderTimeout is the default timeout for optimal ordering search.
// 60 seconds provides enough time for most dependency graphs (<100 nodes)
// to find an optimal or near-optimal ordering while keeping the CLI responsive.
// Users can increase this via --ordering-timeout for larger graphs.
const DefaultOrderTimeout = 60

// DefaultCommonOptions returns CommonOptions with sensible defaults.
func DefaultCommonOptions() CommonOptions {
	return CommonOptions{
		OrderTimeout: DefaultOrderTimeout,
	}
}

// ParseCmdOpts combines pipeline options with CLI-specific options for parsing.
type ParseCmdOpts struct {
	pipeline.Options
	CommonOptions
	Name string // override project name for manifest parsing
}

// DefaultParseCmdOpts returns ParseCmdOpts with sensible defaults.
func DefaultParseCmdOpts() ParseCmdOpts {
	opts := ParseCmdOpts{
		CommonOptions: DefaultCommonOptions(),
	}
	// Set parse-specific defaults
	opts.MaxDepth = pipeline.DefaultMaxDepth
	opts.MaxNodes = pipeline.DefaultMaxNodes
	opts.SkipEnrich = false
	opts.Normalize = true
	return opts
}

// LayoutCmdOpts combines pipeline options with CLI-specific options for layout.
type LayoutCmdOpts struct {
	pipeline.Options
	CommonOptions
}

// DefaultLayoutCmdOpts returns LayoutCmdOpts with sensible defaults.
func DefaultLayoutCmdOpts() LayoutCmdOpts {
	opts := LayoutCmdOpts{
		CommonOptions: DefaultCommonOptions(),
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
	CommonOptions
}

// DefaultVisualizeCmdOpts returns VisualizeCmdOpts with sensible defaults.
func DefaultVisualizeCmdOpts() VisualizeCmdOpts {
	opts := VisualizeCmdOpts{
		CommonOptions: DefaultCommonOptions(),
	}
	// Set visualize-specific defaults
	opts.Style = pipeline.DefaultStyle
	opts.Popups = true
	return opts
}

// RenderCmdOpts combines pipeline options with CLI-specific options for full render.
type RenderCmdOpts struct {
	pipeline.Options
	CommonOptions
}

// DefaultRenderCmdOpts returns RenderCmdOpts with sensible defaults.
func DefaultRenderCmdOpts() RenderCmdOpts {
	opts := RenderCmdOpts{
		CommonOptions: DefaultCommonOptions(),
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
