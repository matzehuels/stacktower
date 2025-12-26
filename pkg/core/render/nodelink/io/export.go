package io

import (
	"encoding/json"
	"fmt"
)

// WriteOption configures layout serialization.
type WriteOption func(*writeConfig)

type writeConfig struct {
	engine string
	style  string
}

// WithEngine sets the graphviz engine name in the output.
func WithEngine(engine string) WriteOption {
	return func(c *writeConfig) {
		c.engine = engine
	}
}

// WithStyle sets the style name in the output.
func WithStyle(style string) WriteOption {
	return func(c *writeConfig) {
		c.style = style
	}
}

// WriteLayout serializes nodelink layout data to JSON.
//
// This is a placeholder - the actual implementation would take
// graphviz output and convert it to our LayoutData format.
func WriteLayout(data *LayoutData, opts ...WriteOption) ([]byte, error) {
	if data == nil {
		return nil, fmt.Errorf("layout data is nil")
	}

	cfg := &writeConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Apply config to data
	data.VizType = VizType
	if cfg.engine != "" {
		data.Engine = cfg.engine
	}
	if cfg.style != "" {
		data.Style = cfg.style
	}

	return json.MarshalIndent(data, "", "  ")
}
