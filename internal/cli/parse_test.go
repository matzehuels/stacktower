package cli

import (
	"testing"

	"github.com/matzehuels/stacktower/pkg/core/deps/languages"
)

func TestLooksLikeFile(t *testing.T) {
	// Test cases based on actual language definitions in pkg/core/deps.
	// Only manifests defined in Language.ManifestAliases are recognized.
	tests := []struct {
		name string
		arg  string
		want bool
	}{
		// Recognized manifest files (from language definitions)
		{"requirements.txt", "requirements.txt", true},
		{"poetry.lock", "poetry.lock", true},
		{"pom.xml", "pom.xml", true},
		{"go.mod", "go.mod", true},
		{"package.json", "package.json", true},
		{"Cargo.toml", "Cargo.toml", true},
		{"cargo.toml lowercase", "cargo.toml", true},
		{"Gemfile", "Gemfile", true},
		{"composer.json", "composer.json", true},

		// Not in language definitions (even if reasonable)
		// pyproject.toml is not in Python's ManifestAliases
		{"pyproject.toml not defined", "pyproject.toml", false},
		// Case sensitivity: go.mod is defined, but GO.MOD is not
		{"GO.MOD uppercase not matched", "GO.MOD", false},

		// Package names (not files)
		{"package name", "requests", false},
		{"package with version", "requests==2.0", false},
		{"package with dash", "my-package", false},
		{"package with underscore", "my_package", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := looksLikeFile(tt.arg); got != tt.want {
				t.Errorf("looksLikeFile(%q) = %v, want %v", tt.arg, got, tt.want)
			}
		})
	}
}

func TestLanguagesRegistered(t *testing.T) {
	if len(languages.All) == 0 {
		t.Error("languages.All slice should not be empty")
	}

	// Check that all expected languages are present
	expectedLangs := []string{"python", "rust", "javascript", "ruby", "php", "java", "go"}
	langNames := make(map[string]bool)
	for _, lang := range languages.All {
		langNames[lang.Name] = true
	}

	for _, expected := range expectedLangs {
		if !langNames[expected] {
			t.Errorf("languages.All missing %q", expected)
		}
	}
}
