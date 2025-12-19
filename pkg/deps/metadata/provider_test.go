package metadata

import (
	"context"
	"errors"
	"testing"

	"github.com/matzehuels/stacktower/pkg/deps"
)

type mockProvider struct {
	name string
	data map[string]any
	err  error
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Enrich(ctx context.Context, pkg *deps.PackageRef, refresh bool) (map[string]any, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.data, nil
}

func TestNewComposite(t *testing.T) {
	p1 := &mockProvider{name: "p1"}
	p2 := &mockProvider{name: "p2"}

	c := NewComposite(p1, p2)

	if c == nil {
		t.Fatal("NewComposite() returned nil")
	}
	if len(c.providers) != 2 {
		t.Errorf("providers count = %d, want 2", len(c.providers))
	}
}

func TestCompositeName(t *testing.T) {
	c := NewComposite()
	if got := c.Name(); got != "composite" {
		t.Errorf("Name() = %q, want %q", got, "composite")
	}
}

func TestCompositeEnrich(t *testing.T) {
	p1 := &mockProvider{
		name: "p1",
		data: map[string]any{"key1": "value1"},
	}
	p2 := &mockProvider{
		name: "p2",
		data: map[string]any{"key2": "value2"},
	}

	c := NewComposite(p1, p2)
	pkg := &deps.PackageRef{Name: "test-pkg"}

	result, err := c.Enrich(context.Background(), pkg, false)
	if err != nil {
		t.Fatalf("Enrich() error: %v", err)
	}

	if result["key1"] != "value1" {
		t.Errorf("result[key1] = %v, want %q", result["key1"], "value1")
	}
	if result["key2"] != "value2" {
		t.Errorf("result[key2] = %v, want %q", result["key2"], "value2")
	}
}

func TestCompositeEnrichMergesData(t *testing.T) {
	p1 := &mockProvider{
		name: "p1",
		data: map[string]any{"shared": "from-p1", "p1-only": "yes"},
	}
	p2 := &mockProvider{
		name: "p2",
		data: map[string]any{"shared": "from-p2", "p2-only": "yes"},
	}

	c := NewComposite(p1, p2)
	pkg := &deps.PackageRef{Name: "test-pkg"}

	result, err := c.Enrich(context.Background(), pkg, false)
	if err != nil {
		t.Fatalf("Enrich() error: %v", err)
	}

	// Later providers should override earlier ones
	if result["shared"] != "from-p2" {
		t.Errorf("result[shared] = %v, want %q (should be overwritten by p2)", result["shared"], "from-p2")
	}
	if result["p1-only"] != "yes" {
		t.Errorf("result[p1-only] = %v, want %q", result["p1-only"], "yes")
	}
	if result["p2-only"] != "yes" {
		t.Errorf("result[p2-only] = %v, want %q", result["p2-only"], "yes")
	}
}

func TestCompositeEnrichSkipsErrors(t *testing.T) {
	p1 := &mockProvider{
		name: "p1",
		err:  errors.New("provider error"),
	}
	p2 := &mockProvider{
		name: "p2",
		data: map[string]any{"key": "value"},
	}

	c := NewComposite(p1, p2)
	pkg := &deps.PackageRef{Name: "test-pkg"}

	result, err := c.Enrich(context.Background(), pkg, false)
	if err != nil {
		t.Fatalf("Enrich() should not return error when individual providers fail")
	}

	// p2's data should still be present
	if result["key"] != "value" {
		t.Errorf("result[key] = %v, want %q", result["key"], "value")
	}
}

func TestCompositeEnrichNoProviders(t *testing.T) {
	c := NewComposite()
	pkg := &deps.PackageRef{Name: "test-pkg"}

	result, err := c.Enrich(context.Background(), pkg, false)
	if err != nil {
		t.Fatalf("Enrich() error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Enrich() with no providers should return empty map, got %v", result)
	}
}
