//go:build integration

package deps

import (
	"context"
	"testing"
	"time"
)

// mockIntegrationFetcher wraps a real fetcher for integration testing
type mockIntegrationFetcher struct {
	fetchCount int
}

func (f *mockIntegrationFetcher) Fetch(ctx context.Context, name string, refresh bool) (*Package, error) {
	f.fetchCount++
	// Return a simple package with no deps to test the resolver machinery
	return &Package{
		Name:         name,
		Version:      "1.0.0",
		Dependencies: nil,
	}, nil
}

func TestRegistryResolve_Integration(t *testing.T) {
	fetcher := &mockIntegrationFetcher{}
	registry := NewRegistry("test", fetcher)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := Options{
		MaxDepth: 5,
		MaxNodes: 100,
	}

	g, err := registry.Resolve(ctx, "test-package", opts)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if g == nil {
		t.Fatal("Resolve() returned nil DAG")
	}

	if g.NodeCount() == 0 {
		t.Error("DAG should have at least one node")
	}

	if fetcher.fetchCount == 0 {
		t.Error("fetcher should have been called")
	}
}

func TestRegistryResolveWithDeps_Integration(t *testing.T) {
	// Fetcher that returns packages with dependencies
	packages := map[string]*Package{
		"root": {
			Name:         "root",
			Version:      "1.0.0",
			Dependencies: []string{"dep-a", "dep-b"},
		},
		"dep-a": {
			Name:         "dep-a",
			Version:      "2.0.0",
			Dependencies: []string{"shared"},
		},
		"dep-b": {
			Name:         "dep-b",
			Version:      "3.0.0",
			Dependencies: []string{"shared"},
		},
		"shared": {
			Name:         "shared",
			Version:      "1.0.0",
			Dependencies: nil,
		},
	}

	fetcher := &mapFetcher{packages: packages}
	registry := NewRegistry("test", fetcher)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	g, err := registry.Resolve(ctx, "root", Options{MaxDepth: 10, MaxNodes: 100})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	// Should have all 4 packages
	if g.NodeCount() != 4 {
		t.Errorf("NodeCount() = %d, want 4", g.NodeCount())
	}

	// Check edges exist
	if g.EdgeCount() < 4 {
		t.Errorf("EdgeCount() = %d, want at least 4", g.EdgeCount())
	}
}

type mapFetcher struct {
	packages map[string]*Package
}

func (f *mapFetcher) Fetch(ctx context.Context, name string, refresh bool) (*Package, error) {
	if pkg, ok := f.packages[name]; ok {
		return pkg, nil
	}
	return &Package{Name: name, Version: "unknown"}, nil
}
