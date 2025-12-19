package golang

import (
	"testing"
	"time"

	"github.com/matzehuels/stacktower/pkg/deps"
)

func TestLanguageDefinition(t *testing.T) {
	if Language == nil {
		t.Fatal("Language should not be nil")
	}

	if Language.Name != "go" {
		t.Errorf("Name = %q, want %q", Language.Name, "go")
	}

	if Language.DefaultRegistry != "goproxy" {
		t.Errorf("DefaultRegistry = %q, want %q", Language.DefaultRegistry, "goproxy")
	}
}

func TestLanguageRegistryAliases(t *testing.T) {
	tests := []struct {
		alias string
		want  string
	}{
		{"proxy", "goproxy"},
		{"go", "goproxy"},
		{"goproxy", "goproxy"},
	}

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			got := Language.RegistryAliases[tt.alias]
			if tt.alias == "goproxy" {
				// Not in aliases, should be handled as default
				return
			}
			if got != tt.want {
				t.Errorf("RegistryAliases[%q] = %q, want %q", tt.alias, got, tt.want)
			}
		})
	}
}

func TestLanguageManifestTypes(t *testing.T) {
	if len(Language.ManifestTypes) == 0 {
		t.Error("ManifestTypes should not be empty")
	}

	found := false
	for _, mt := range Language.ManifestTypes {
		if mt == "gomod" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ManifestTypes should contain 'gomod'")
	}
}

func TestLanguageManifestAliases(t *testing.T) {
	if Language.ManifestAliases["go.mod"] != "gomod" {
		t.Errorf("ManifestAliases[go.mod] = %q, want %q",
			Language.ManifestAliases["go.mod"], "gomod")
	}
}

func TestNewManifest(t *testing.T) {
	tests := []struct {
		name    string
		wantNil bool
	}{
		{"gomod", false},
		{"unknown", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := newManifest(tt.name, nil)
			if tt.wantNil && parser != nil {
				t.Error("newManifest() should return nil for unknown manifest type")
			}
			if !tt.wantNil && parser == nil {
				t.Error("newManifest() should not return nil for known manifest type")
			}
		})
	}
}

func TestManifestParsers(t *testing.T) {
	parsers := manifestParsers(nil)

	if len(parsers) == 0 {
		t.Error("manifestParsers() should return at least one parser")
	}

	// Check that GoModParser is included
	found := false
	for _, p := range parsers {
		if _, ok := p.(*GoModParser); ok {
			found = true
			break
		}
	}
	if !found {
		t.Error("manifestParsers() should include GoModParser")
	}
}

func TestLanguageHasManifests(t *testing.T) {
	if !Language.HasManifests() {
		t.Error("HasManifests() should return true for Go language")
	}
}

func TestLanguageManifest(t *testing.T) {
	// Test getting manifest parser via Language interface
	parser, ok := Language.Manifest("gomod", nil)
	if !ok {
		t.Error("Manifest(gomod) should return true")
	}
	if parser == nil {
		t.Error("Manifest(gomod) should return non-nil parser")
	}

	// Test alias
	parser, ok = Language.Manifest("go.mod", nil)
	if !ok {
		t.Error("Manifest(go.mod) should return true (via alias)")
	}
	if parser == nil {
		t.Error("Manifest(go.mod) should return non-nil parser")
	}
}

func TestNewResolverFunctionExists(t *testing.T) {
	if Language.NewResolver == nil {
		t.Error("NewResolver function should not be nil")
	}
}

// Integration test - only run if network is available
func TestNewResolverCreatesResolver(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	resolver, err := newResolver(time.Hour)
	if err != nil {
		t.Fatalf("newResolver() error: %v", err)
	}

	if resolver == nil {
		t.Error("newResolver() should return non-nil resolver")
	}

	if resolver.Name() != "goproxy" {
		t.Errorf("resolver.Name() = %q, want %q", resolver.Name(), "goproxy")
	}
}

func TestLanguageResolverMethod(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	resolver, err := Language.Resolver()
	if err != nil {
		t.Fatalf("Resolver() error: %v", err)
	}

	if resolver == nil {
		t.Error("Resolver() should return non-nil")
	}
}

func TestLanguageRegistryMethod(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Test default registry
	resolver, err := Language.Registry("goproxy")
	if err != nil {
		t.Fatalf("Registry(goproxy) error: %v", err)
	}
	if resolver == nil {
		t.Error("Registry(goproxy) should return non-nil")
	}

	// Test alias
	resolver, err = Language.Registry("proxy")
	if err != nil {
		t.Fatalf("Registry(proxy) error: %v", err)
	}
	if resolver == nil {
		t.Error("Registry(proxy) should return non-nil")
	}

	// Test unknown registry
	_, err = Language.Registry("unknown")
	if err == nil {
		t.Error("Registry(unknown) should return error")
	}
}

// Test that fetcher implements deps.Fetcher
func TestFetcherImplementsInterface(t *testing.T) {
	var _ deps.Fetcher = fetcher{}
}
