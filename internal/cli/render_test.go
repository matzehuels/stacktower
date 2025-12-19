package cli

import (
	"testing"
)

func TestParseVizTypes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty defaults to tower", "", []string{"tower"}},
		{"single type", "tower", []string{"tower"}},
		{"multiple types", "tower,nodelink", []string{"tower", "nodelink"}},
		{"nodelink only", "nodelink", []string{"nodelink"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseVizTypes(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseVizTypes(%q) length = %d, want %d", tt.input, len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("parseVizTypes(%q)[%d] = %q, want %q", tt.input, i, v, tt.want[i])
				}
			}
		})
	}
}

func TestParseFormats(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty defaults to svg", "", []string{"svg"}},
		{"single format", "svg", []string{"svg"}},
		{"multiple formats", "svg,json,pdf", []string{"svg", "json", "pdf"}},
		{"json only", "json", []string{"json"}},
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
		{"valid json", []string{"json"}, false},
		{"valid pdf", []string{"pdf"}, false},
		{"valid png", []string{"png"}, false},
		{"valid multiple", []string{"svg", "json", "pdf", "png"}, false},
		{"invalid format", []string{"invalid"}, true},
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

func TestBasePath(t *testing.T) {
	tests := []struct {
		name   string
		output string
		input  string
		want   string
	}{
		{"empty output uses input", "", "/path/to/file.json", "/path/to/file"},
		{"output without format ext", "/out/file", "/in/file.json", "/out/file"},
		{"output with svg ext", "/out/file.svg", "/in/file.json", "/out/file"},
		{"output with json ext", "/out/file.json", "/in/file.json", "/out/file"},
		{"output with pdf ext", "/out/file.pdf", "/in/file.json", "/out/file"},
		{"output with png ext", "/out/file.png", "/in/file.json", "/out/file"},
		{"output with unknown ext", "/out/file.txt", "/in/file.json", "/out/file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := basePath(tt.output, tt.input); got != tt.want {
				t.Errorf("basePath(%q, %q) = %q, want %q", tt.output, tt.input, got, tt.want)
			}
		})
	}
}

func TestValidFormatsMap(t *testing.T) {
	expected := map[string]bool{
		"svg":  true,
		"json": true,
		"pdf":  true,
		"png":  true,
	}

	for k, v := range expected {
		if validFormats[k] != v {
			t.Errorf("validFormats[%q] = %v, want %v", k, validFormats[k], v)
		}
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
