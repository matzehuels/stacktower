package cli

import (
	"testing"

	"github.com/matzehuels/stacktower/pkg/deps/languages"
)

func TestLooksLikeFile(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want bool
	}{
		{"txt extension", "requirements.txt", true},
		{"lock extension", "poetry.lock", true},
		{"toml extension", "pyproject.toml", true},
		{"xml extension", "pom.xml", true},
		{"go.mod", "go.mod", true},
		{"GO.MOD uppercase", "GO.MOD", true},
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

func TestNopCloser(t *testing.T) {
	nc := nopCloser{}
	if err := nc.Close(); err != nil {
		t.Errorf("nopCloser.Close() error: %v", err)
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
