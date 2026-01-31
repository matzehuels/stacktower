package deps

import (
	"testing"
	"time"
)

func TestOptionsWithDefaults(t *testing.T) {
	tests := []struct {
		name string
		opts Options
		want Options
	}{
		{
			name: "all zeros use defaults",
			opts: Options{},
			want: Options{
				MaxDepth: DefaultMaxDepth,
				MaxNodes: DefaultMaxNodes,
				CacheTTL: DefaultCacheTTL,
			},
		},
		{
			name: "preserves non-zero values",
			opts: Options{
				MaxDepth: 10,
				MaxNodes: 100,
				CacheTTL: time.Hour,
			},
			want: Options{
				MaxDepth: 10,
				MaxNodes: 100,
				CacheTTL: time.Hour,
			},
		},
		{
			name: "negative values use defaults",
			opts: Options{
				MaxDepth: -1,
				MaxNodes: -5,
				CacheTTL: -time.Hour,
			},
			want: Options{
				MaxDepth: DefaultMaxDepth,
				MaxNodes: DefaultMaxNodes,
				CacheTTL: DefaultCacheTTL,
			},
		},
		{
			name: "partial defaults",
			opts: Options{
				MaxDepth: 5,
				MaxNodes: 0,
				CacheTTL: time.Minute,
			},
			want: Options{
				MaxDepth: 5,
				MaxNodes: DefaultMaxNodes,
				CacheTTL: time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.opts.WithDefaults()
			if got.MaxDepth != tt.want.MaxDepth {
				t.Errorf("MaxDepth = %d, want %d", got.MaxDepth, tt.want.MaxDepth)
			}
			if got.MaxNodes != tt.want.MaxNodes {
				t.Errorf("MaxNodes = %d, want %d", got.MaxNodes, tt.want.MaxNodes)
			}
			if got.CacheTTL != tt.want.CacheTTL {
				t.Errorf("CacheTTL = %v, want %v", got.CacheTTL, tt.want.CacheTTL)
			}
			if got.Logger == nil {
				t.Error("Logger should not be nil after WithDefaults")
			}
		})
	}
}

func TestOptionsWithDefaultsPreservesLogger(t *testing.T) {
	called := false
	logger := func(string, ...any) { called = true }

	opts := Options{Logger: logger}.WithDefaults()
	opts.Logger("test")

	if !called {
		t.Error("custom logger should be preserved")
	}
}

func TestPackageMetadata(t *testing.T) {
	tests := []struct {
		name string
		pkg  Package
		want map[string]any
	}{
		{
			name: "version only",
			pkg:  Package{Version: "1.0.0"},
			want: map[string]any{"version": "1.0.0"},
		},
		{
			name: "all fields",
			pkg: Package{
				Version:     "2.0.0",
				Description: "A test package",
				License:     "MIT",
				Author:      "Test Author",
				Downloads:   1000,
			},
			want: map[string]any{
				"version":     "2.0.0",
				"description": "A test package",
				"license":     "MIT",
				"author":      "Test Author",
				"downloads":   1000,
			},
		},
		{
			name: "empty optional fields excluded",
			pkg: Package{
				Version:     "1.0.0",
				Description: "",
				License:     "",
				Author:      "",
				Downloads:   0,
			},
			want: map[string]any{"version": "1.0.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pkg.Metadata()

			if len(got) != len(tt.want) {
				t.Errorf("Metadata() length = %d, want %d", len(got), len(tt.want))
			}

			for k, wantV := range tt.want {
				if gotV, ok := got[k]; !ok {
					t.Errorf("Metadata() missing key %q", k)
				} else if gotV != wantV {
					t.Errorf("Metadata()[%q] = %v, want %v", k, gotV, wantV)
				}
			}
		})
	}
}

func TestPackageRef(t *testing.T) {
	tests := []struct {
		name string
		pkg  Package
		want PackageRef
	}{
		{
			name: "basic package",
			pkg: Package{
				Name:    "test-pkg",
				Version: "1.0.0",
			},
			want: PackageRef{
				Name:        "test-pkg",
				Version:     "1.0.0",
				ProjectURLs: map[string]string{},
			},
		},
		{
			name: "with repository and homepage",
			pkg: Package{
				Name:       "test-pkg",
				Version:    "2.0.0",
				Repository: "https://github.com/test/repo",
				HomePage:   "https://test.example.com",
			},
			want: PackageRef{
				Name:    "test-pkg",
				Version: "2.0.0",
				ProjectURLs: map[string]string{
					"repository": "https://github.com/test/repo",
					"homepage":   "https://test.example.com",
				},
				HomePage: "https://test.example.com",
			},
		},
		{
			name: "with existing project urls",
			pkg: Package{
				Name:        "test-pkg",
				Version:     "3.0.0",
				ProjectURLs: map[string]string{"docs": "https://docs.example.com"},
				Repository:  "https://github.com/test/repo",
			},
			want: PackageRef{
				Name:    "test-pkg",
				Version: "3.0.0",
				ProjectURLs: map[string]string{
					"docs":       "https://docs.example.com",
					"repository": "https://github.com/test/repo",
				},
			},
		},
		{
			name: "with manifest file",
			pkg: Package{
				Name:         "test-pkg",
				Version:      "1.0.0",
				ManifestFile: "requirements.txt",
			},
			want: PackageRef{
				Name:         "test-pkg",
				Version:      "1.0.0",
				ProjectURLs:  map[string]string{},
				ManifestFile: "requirements.txt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pkg.Ref()

			if got.Name != tt.want.Name {
				t.Errorf("Ref().Name = %q, want %q", got.Name, tt.want.Name)
			}
			if got.Version != tt.want.Version {
				t.Errorf("Ref().Version = %q, want %q", got.Version, tt.want.Version)
			}
			if got.HomePage != tt.want.HomePage {
				t.Errorf("Ref().HomePage = %q, want %q", got.HomePage, tt.want.HomePage)
			}
			if got.ManifestFile != tt.want.ManifestFile {
				t.Errorf("Ref().ManifestFile = %q, want %q", got.ManifestFile, tt.want.ManifestFile)
			}

			if len(got.ProjectURLs) != len(tt.want.ProjectURLs) {
				t.Errorf("Ref().ProjectURLs length = %d, want %d", len(got.ProjectURLs), len(tt.want.ProjectURLs))
			}
			for k, wantV := range tt.want.ProjectURLs {
				if gotV, ok := got.ProjectURLs[k]; !ok {
					t.Errorf("Ref().ProjectURLs missing key %q", k)
				} else if gotV != wantV {
					t.Errorf("Ref().ProjectURLs[%q] = %q, want %q", k, gotV, wantV)
				}
			}
		})
	}
}

func TestPackageRefDoesNotModifyOriginal(t *testing.T) {
	original := map[string]string{"key": "value"}
	pkg := Package{
		Name:        "test",
		Version:     "1.0",
		ProjectURLs: original,
		Repository:  "https://github.com/test/repo",
	}

	ref := pkg.Ref()
	ref.ProjectURLs["new"] = "added"

	if _, exists := original["new"]; exists {
		t.Error("Ref() should clone ProjectURLs, not reference original")
	}
}
