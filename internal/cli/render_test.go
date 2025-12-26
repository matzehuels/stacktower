package cli

import (
	"testing"
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
		{"valid multiple", []string{"svg", "pdf", "png"}, false},
		{"invalid format", []string{"invalid"}, true},
		{"json not allowed", []string{"json"}, true}, // json is layout output, not render
		{"mixed valid invalid", []string{"svg", "invalid"}, true},
		{"empty slice", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFormats(tt.formats)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFormats(%v) error = %v, wantErr %v", tt.formats, err, tt.wantErr)
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
			err := validateStyle(tt.style)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateStyle(%q) error = %v, wantErr %v", tt.style, err, tt.wantErr)
			}
		})
	}
}

func TestValidFormatsMap(t *testing.T) {
	expected := map[string]bool{
		"svg": true,
		"pdf": true,
		"png": true,
	}

	for k, v := range expected {
		if validFormats[k] != v {
			t.Errorf("validFormats[%q] = %v, want %v", k, validFormats[k], v)
		}
	}

	// json is NOT a valid render format (it's layout output)
	if validFormats["json"] {
		t.Error("validFormats[json] should be false")
	}

	if validFormats["invalid"] {
		t.Error("validFormats[invalid] should be false")
	}
}

func TestStyleConstants(t *testing.T) {
	if styleSimple != "simple" {
		t.Errorf("styleSimple = %q, want %q", styleSimple, "simple")
	}
	if styleHanddrawn != "handdrawn" {
		t.Errorf("styleHanddrawn = %q, want %q", styleHanddrawn, "handdrawn")
	}
}

func TestDefaultConstants(t *testing.T) {
	if defaultWidth != 800 {
		t.Errorf("defaultWidth = %v, want 800", defaultWidth)
	}
	if defaultHeight != 600 {
		t.Errorf("defaultHeight = %v, want 600", defaultHeight)
	}
	if defaultSeed != 42 {
		t.Errorf("defaultSeed = %v, want 42", defaultSeed)
	}
}
