package cli

import (
	"testing"

	"github.com/matzehuels/stacktower/pkg/pipeline"
)

func TestParseFormats(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty defaults to svg", "", []string{"svg"}},
		{"single format", "svg", []string{"svg"}},
		{"multiple formats", "svg,pdf,png", []string{"svg", "pdf", "png"}},
		{"pdf only", "pdf", []string{"pdf"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFormats(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseFormats(%q) length = %d, want %d", tt.input, len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("parseFormats(%q)[%d] = %q, want %q", tt.input, i, v, tt.want[i])
				}
			}
		})
	}
}

func TestValidateFormats(t *testing.T) {
	tests := []struct {
		name    string
		formats []string
		wantErr bool
	}{
		{"valid svg", []string{"svg"}, false},
		{"valid pdf", []string{"pdf"}, false},
		{"valid png", []string{"png"}, false},
		{"valid json", []string{"json"}, false}, // json is a valid format for graph data export
		{"valid multiple", []string{"svg", "pdf", "png"}, false},
		{"valid all", []string{"svg", "pdf", "png", "json"}, false},
		{"invalid format", []string{"invalid"}, true},
		{"mixed valid invalid", []string{"svg", "invalid"}, true},
		{"empty slice", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pipeline.ValidateFormats(tt.formats)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFormats(%v) error = %v, wantErr %v", tt.formats, err, tt.wantErr)
			}
		})
	}
}

func TestValidateStyle(t *testing.T) {
	tests := []struct {
		name    string
		style   string
		wantErr bool
	}{
		{"simple", "simple", false},
		{"handdrawn", "handdrawn", false},
		{"invalid", "invalid", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pipeline.ValidateStyle(tt.style)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStyle(%q) error = %v, wantErr %v", tt.style, err, tt.wantErr)
			}
		})
	}
}

func TestValidFormatsMap(t *testing.T) {
	expected := map[string]bool{
		"svg":  true,
		"pdf":  true,
		"png":  true,
		"json": true, // json is valid for graph data export
	}

	for k, v := range expected {
		if pipeline.ValidFormats[k] != v {
			t.Errorf("ValidFormats[%q] = %v, want %v", k, pipeline.ValidFormats[k], v)
		}
	}

	if pipeline.ValidFormats["invalid"] {
		t.Error("ValidFormats[invalid] should be false")
	}
}

func TestStyleConstants(t *testing.T) {
	if pipeline.StyleSimple != "simple" {
		t.Errorf("pipeline.StyleSimple = %q, want %q", pipeline.StyleSimple, "simple")
	}
	if pipeline.StyleHanddrawn != "handdrawn" {
		t.Errorf("pipeline.StyleHanddrawn = %q, want %q", pipeline.StyleHanddrawn, "handdrawn")
	}
}

func TestDefaultConstants(t *testing.T) {
	if pipeline.DefaultWidth != 800 {
		t.Errorf("pipeline.DefaultWidth = %v, want 800", pipeline.DefaultWidth)
	}
	if pipeline.DefaultHeight != 600 {
		t.Errorf("pipeline.DefaultHeight = %v, want 600", pipeline.DefaultHeight)
	}
	if pipeline.DefaultSeed != 42 {
		t.Errorf("pipeline.DefaultSeed = %v, want 42", pipeline.DefaultSeed)
	}
}
