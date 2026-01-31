package deps

import (
	"testing"
)

type mockManifestParserForDetect struct {
	typeName     string
	supportsFunc func(string) bool
}

func (m *mockManifestParserForDetect) Type() string { return m.typeName }
func (m *mockManifestParserForDetect) Supports(filename string) bool {
	if m.supportsFunc != nil {
		return m.supportsFunc(filename)
	}
	return false
}
func (m *mockManifestParserForDetect) IncludesTransitive() bool { return false }
func (m *mockManifestParserForDetect) Parse(path string, opts Options) (*ManifestResult, error) {
	return &ManifestResult{}, nil
}

func TestDetectManifest(t *testing.T) {
	poetry := &mockManifestParserForDetect{
		typeName: "poetry",
		supportsFunc: func(f string) bool {
			return f == "pyproject.toml"
		},
	}
	requirements := &mockManifestParserForDetect{
		typeName: "requirements",
		supportsFunc: func(f string) bool {
			return f == "requirements.txt"
		},
	}

	tests := []struct {
		name     string
		path     string
		parsers  []ManifestParser
		wantType string
		wantErr  bool
	}{
		{
			name:     "matches poetry",
			path:     "/some/path/pyproject.toml",
			parsers:  []ManifestParser{poetry, requirements},
			wantType: "poetry",
			wantErr:  false,
		},
		{
			name:     "matches requirements",
			path:     "/project/requirements.txt",
			parsers:  []ManifestParser{poetry, requirements},
			wantType: "requirements",
			wantErr:  false,
		},
		{
			name:    "no match",
			path:    "/project/unknown.yaml",
			parsers: []ManifestParser{poetry, requirements},
			wantErr: true,
		},
		{
			name:    "no parsers",
			path:    "/project/anything.txt",
			parsers: []ManifestParser{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := DetectManifest(tt.path, tt.parsers...)
			if tt.wantErr {
				if err == nil {
					t.Error("DetectManifest() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("DetectManifest() unexpected error: %v", err)
			}
			if parser.Type() != tt.wantType {
				t.Errorf("DetectManifest().Type() = %q, want %q", parser.Type(), tt.wantType)
			}
		})
	}
}

func TestDetectManifestFirstMatch(t *testing.T) {
	// Test that first matching parser is returned
	p1 := &mockManifestParserForDetect{
		typeName: "first",
		supportsFunc: func(f string) bool {
			return f == "test.txt"
		},
	}
	p2 := &mockManifestParserForDetect{
		typeName: "second",
		supportsFunc: func(f string) bool {
			return f == "test.txt"
		},
	}

	parser, err := DetectManifest("/path/test.txt", p1, p2)
	if err != nil {
		t.Fatalf("DetectManifest() error: %v", err)
	}
	if parser.Type() != "first" {
		t.Errorf("DetectManifest() should return first matching parser, got %q", parser.Type())
	}
}
