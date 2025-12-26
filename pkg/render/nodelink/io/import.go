package io

import (
	"encoding/json"
	"fmt"
	"io"
)

// ReadLayout deserializes a nodelink layout from JSON.
//
// Returns the layout data and metadata for configuring renderers.
func ReadLayout(r io.Reader) (*LayoutData, *LayoutMeta, error) {
	var data LayoutData
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return nil, nil, fmt.Errorf("decode layout: %w", err)
	}

	// Validate viz type
	if data.VizType != "" && data.VizType != VizType {
		return nil, nil, fmt.Errorf("unexpected viz_type: %q (expected %q)", data.VizType, VizType)
	}

	meta := &LayoutMeta{
		VizType: VizType,
		Engine:  data.Engine,
		Style:   data.Style,
	}

	return &data, meta, nil
}
