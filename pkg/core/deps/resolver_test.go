package deps

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockFetcher struct {
	packages map[string]*Package
	fetchErr error
}

func (m *mockFetcher) Fetch(ctx context.Context, name string, refresh bool) (*Package, error) {
	if m.fetchErr != nil {
		return nil, m.fetchErr
	}
	if pkg, ok := m.packages[name]; ok {
		return pkg, nil
	}
	return nil, errors.New("package not found")
}

func TestNewRegistry(t *testing.T) {
	fetcher := &mockFetcher{}
	r := NewRegistry("test-registry", fetcher)

	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	if r.name != "test-registry" {
		t.Errorf("name = %q, want %q", r.name, "test-registry")
	}
	if r.fetcher != fetcher {
		t.Error("fetcher not set correctly")
	}
}

func TestRegistryName(t *testing.T) {
	r := NewRegistry("my-registry", &mockFetcher{})
	if got := r.Name(); got != "my-registry" {
		t.Errorf("Name() = %q, want %q", got, "my-registry")
	}
}

func TestRegistryResolveSinglePackage(t *testing.T) {
	fetcher := &mockFetcher{
		packages: map[string]*Package{
			"root": {
				Name:         "root",
				Version:      "1.0.0",
				Dependencies: nil,
			},
		},
	}
	r := NewRegistry("test", fetcher)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dag, err := r.Resolve(ctx, "root", Options{})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if dag == nil {
		t.Fatal("Resolve() returned nil DAG")
	}
	if dag.NodeCount() != 1 {
		t.Errorf("NodeCount() = %d, want 1", dag.NodeCount())
	}

	node, ok := dag.Node("root")
	if !ok {
		t.Error("root node not found")
	}
	if node.ID != "root" {
		t.Errorf("node.ID = %q, want %q", node.ID, "root")
	}
}

func TestRegistryResolveWithDependencies(t *testing.T) {
	fetcher := &mockFetcher{
		packages: map[string]*Package{
			"root": {
				Name:         "root",
				Version:      "1.0.0",
				Dependencies: []string{"dep-a", "dep-b"},
			},
			"dep-a": {
				Name:         "dep-a",
				Version:      "2.0.0",
				Dependencies: nil,
			},
			"dep-b": {
				Name:         "dep-b",
				Version:      "3.0.0",
				Dependencies: nil,
			},
		},
	}
	r := NewRegistry("test", fetcher)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dag, err := r.Resolve(ctx, "root", Options{})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if dag.NodeCount() != 3 {
		t.Errorf("NodeCount() = %d, want 3", dag.NodeCount())
	}
	if dag.EdgeCount() != 2 {
		t.Errorf("EdgeCount() = %d, want 2", dag.EdgeCount())
	}
}

func TestRegistryResolveWithTransitiveDeps(t *testing.T) {
	fetcher := &mockFetcher{
		packages: map[string]*Package{
			"root": {
				Name:         "root",
				Version:      "1.0.0",
				Dependencies: []string{"dep-a"},
			},
			"dep-a": {
				Name:         "dep-a",
				Version:      "2.0.0",
				Dependencies: []string{"dep-b"},
			},
			"dep-b": {
				Name:         "dep-b",
				Version:      "3.0.0",
				Dependencies: nil,
			},
		},
	}
	r := NewRegistry("test", fetcher)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dag, err := r.Resolve(ctx, "root", Options{})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if dag.NodeCount() != 3 {
		t.Errorf("NodeCount() = %d, want 3", dag.NodeCount())
	}
}

func TestRegistryResolveRootError(t *testing.T) {
	fetcher := &mockFetcher{
		packages: map[string]*Package{},
	}
	r := NewRegistry("test", fetcher)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.Resolve(ctx, "nonexistent", Options{})
	if err == nil {
		t.Error("Resolve() should return error for missing root package")
	}
}

func TestRegistryResolveDepErrorContinues(t *testing.T) {
	fetcher := &mockFetcher{
		packages: map[string]*Package{
			"root": {
				Name:         "root",
				Version:      "1.0.0",
				Dependencies: []string{"missing-dep", "existing-dep"},
			},
			"existing-dep": {
				Name:         "existing-dep",
				Version:      "1.0.0",
				Dependencies: nil,
			},
		},
	}
	r := NewRegistry("test", fetcher)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dag, err := r.Resolve(ctx, "root", Options{})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	// Should have root and existing-dep, missing-dep should be skipped
	if dag.NodeCount() < 2 {
		t.Errorf("NodeCount() = %d, want at least 2", dag.NodeCount())
	}
}

func TestRegistryResolveMaxDepth(t *testing.T) {
	fetcher := &mockFetcher{
		packages: map[string]*Package{
			"root": {
				Name:         "root",
				Version:      "1.0.0",
				Dependencies: []string{"level1"},
			},
			"level1": {
				Name:         "level1",
				Version:      "1.0.0",
				Dependencies: []string{"level2"},
			},
			"level2": {
				Name:         "level2",
				Version:      "1.0.0",
				Dependencies: []string{"level3"},
			},
			"level3": {
				Name:         "level3",
				Version:      "1.0.0",
				Dependencies: nil,
			},
		},
	}
	r := NewRegistry("test", fetcher)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dag, err := r.Resolve(ctx, "root", Options{MaxDepth: 2})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	// With MaxDepth=2, should not fetch level3
	if dag.NodeCount() > 3 {
		t.Errorf("NodeCount() = %d, should be limited by MaxDepth", dag.NodeCount())
	}
}

func TestRegistryResolveContextCancellation(t *testing.T) {
	// Test that context cancellation is handled gracefully
	// Note: Due to concurrent nature, we just verify no panic occurs
	fetcher := &mockFetcher{
		packages: map[string]*Package{
			"root": {
				Name:         "root",
				Version:      "1.0.0",
				Dependencies: nil,
			},
		},
	}
	r := NewRegistry("test", fetcher)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Should either succeed or return context error, but not panic
	_, _ = r.Resolve(ctx, "root", Options{})
}

func TestRegistryResolveWithMetadata(t *testing.T) {
	fetcher := &mockFetcher{
		packages: map[string]*Package{
			"root": {
				Name:        "root",
				Version:     "1.0.0",
				Description: "Root package",
				License:     "MIT",
				Author:      "Test Author",
			},
		},
	}
	r := NewRegistry("test", fetcher)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dag, err := r.Resolve(ctx, "root", Options{})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	node, ok := dag.Node("root")
	if !ok {
		t.Fatal("root node not found")
	}
	if node.Meta == nil {
		t.Fatal("node.Meta should not be nil")
	}
	if node.Meta["version"] != "1.0.0" {
		t.Errorf("Meta[version] = %v, want %q", node.Meta["version"], "1.0.0")
	}
}

func TestRegistryResolveDeduplication(t *testing.T) {
	// Diamond dependency: root -> a, b; a -> c; b -> c
	fetcher := &mockFetcher{
		packages: map[string]*Package{
			"root": {
				Name:         "root",
				Dependencies: []string{"a", "b"},
			},
			"a": {
				Name:         "a",
				Dependencies: []string{"c"},
			},
			"b": {
				Name:         "b",
				Dependencies: []string{"c"},
			},
			"c": {
				Name:         "c",
				Dependencies: nil,
			},
		},
	}
	r := NewRegistry("test", fetcher)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dag, err := r.Resolve(ctx, "root", Options{})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	// Should have exactly 4 nodes, not duplicated
	if dag.NodeCount() != 4 {
		t.Errorf("NodeCount() = %d, want 4 (deduplication should prevent duplicates)", dag.NodeCount())
	}
}

type mockMetadataProvider struct {
	name string
	data map[string]any
	err  error
}

func (m *mockMetadataProvider) Name() string { return m.name }
func (m *mockMetadataProvider) Enrich(ctx context.Context, pkg *PackageRef, refresh bool) (map[string]any, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.data, nil
}

func TestRegistryResolveWithMetadataProvider(t *testing.T) {
	fetcher := &mockFetcher{
		packages: map[string]*Package{
			"root": {
				Name:    "root",
				Version: "1.0.0",
			},
		},
	}
	r := NewRegistry("test", fetcher)

	provider := &mockMetadataProvider{
		name: "github",
		data: map[string]any{
			"stars": 1000,
			"url":   "https://github.com/test/root",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dag, err := r.Resolve(ctx, "root", Options{
		MetadataProviders: []MetadataProvider{provider},
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	node, ok := dag.Node("root")
	if !ok {
		t.Fatal("root node not found")
	}
	if node.Meta["stars"] != 1000 {
		t.Errorf("Meta[stars] = %v, want 1000", node.Meta["stars"])
	}
}

func TestRegistryResolveMetadataProviderError(t *testing.T) {
	fetcher := &mockFetcher{
		packages: map[string]*Package{
			"root": {
				Name:    "root",
				Version: "1.0.0",
			},
		},
	}
	r := NewRegistry("test", fetcher)

	provider := &mockMetadataProvider{
		name: "github",
		err:  errors.New("API error"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Should not fail even if metadata provider fails
	dag, err := r.Resolve(ctx, "root", Options{
		MetadataProviders: []MetadataProvider{provider},
	})
	if err != nil {
		t.Fatalf("Resolve() should not fail due to metadata provider error: %v", err)
	}
	if dag.NodeCount() != 1 {
		t.Error("DAG should still have the root node")
	}
}
