package artifact

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/matzehuels/stacktower/pkg/dag"
)

func TestLocalBackendBasics(t *testing.T) {
	// Create temp dir for test
	tmpDir, err := os.MkdirTemp("", "artifact-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create backend
	backend, err := NewLocalBackend(LocalBackendConfig{
		CacheDir: tmpDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()

	ctx := context.Background()

	// Test GetGraph on empty cache (should return false)
	_, found, err := backend.GetGraph(ctx, "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("expected not found for nonexistent graph")
	}

	// Test PutGraph and GetGraph
	g := dag.New(nil)
	g.AddNode(dag.Node{ID: "a"})
	g.AddNode(dag.Node{ID: "b"})
	g.AddEdge(dag.Edge{From: "a", To: "b"})

	hash := "test-graph-hash"
	if err := backend.PutGraph(ctx, hash, g, GraphTTL); err != nil {
		t.Fatal(err)
	}

	// Retrieve graph
	retrieved, found, err := backend.GetGraph(ctx, hash)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Error("expected to find cached graph")
	}
	if retrieved.NodeCount() != 2 {
		t.Errorf("expected 2 nodes, got %d", retrieved.NodeCount())
	}
}

func TestLocalBackendLayout(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "artifact-layout-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	backend, err := NewLocalBackend(LocalBackendConfig{
		CacheDir: tmpDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()

	ctx := context.Background()

	// Test GetLayout on empty cache
	_, found, err := backend.GetLayout(ctx, "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("expected not found for nonexistent layout")
	}

	// Test PutLayout and GetLayout
	layoutData := []byte(`{"viz_type": "tower", "blocks": []}`)
	hash := "test-layout-hash"

	if err := backend.PutLayout(ctx, hash, layoutData, LayoutTTL); err != nil {
		t.Fatal(err)
	}

	retrieved, found, err := backend.GetLayout(ctx, hash)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Error("expected to find cached layout")
	}
	if string(retrieved) != string(layoutData) {
		t.Error("layout data mismatch")
	}
}

func TestLocalBackendRender(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "artifact-render-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	backend, err := NewLocalBackend(LocalBackendConfig{
		CacheDir: tmpDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()

	ctx := context.Background()

	// Test GetRender on empty cache
	_, found, err := backend.GetRender(ctx, "nonexistent", "svg")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("expected not found for nonexistent render")
	}

	// Test PutRender and GetRender
	svgData := []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)
	hash := "test-render-hash"

	if err := backend.PutRender(ctx, hash, "svg", svgData, RenderTTL); err != nil {
		t.Fatal(err)
	}

	retrieved, found, err := backend.GetRender(ctx, hash, "svg")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Error("expected to find cached render")
	}
	if string(retrieved) != string(svgData) {
		t.Error("render data mismatch")
	}

	// Different format should not be found
	_, found, err = backend.GetRender(ctx, hash, "png")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("expected not found for different format")
	}
}

func TestHash(t *testing.T) {
	data1 := []byte("hello world")
	data2 := []byte("hello world")
	data3 := []byte("hello world!")

	hash1 := Hash(data1)
	hash2 := Hash(data2)
	hash3 := Hash(data3)

	if hash1 != hash2 {
		t.Error("identical data should produce identical hashes")
	}

	if hash1 == hash3 {
		t.Error("different data should produce different hashes")
	}
}

func TestHashJSON(t *testing.T) {
	obj1 := map[string]interface{}{
		"language": "python",
		"package":  "requests",
	}

	obj2 := map[string]interface{}{
		"language": "python",
		"package":  "requests",
	}

	obj3 := map[string]interface{}{
		"language": "python",
		"package":  "flask",
	}

	hash1 := HashJSON(obj1)
	hash2 := HashJSON(obj2)
	hash3 := HashJSON(obj3)

	if hash1 != hash2 {
		t.Error("identical objects should produce identical hashes")
	}

	if hash1 == hash3 {
		t.Error("different objects should produce different hashes")
	}
}

func TestNullBackend(t *testing.T) {
	ctx := context.Background()
	backend := NullBackend{}

	// GetGraph should always return false
	_, found, err := backend.GetGraph(ctx, "any")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("NullBackend should never find anything")
	}

	// PutGraph should succeed but not store
	g := dag.New(nil)
	if err := backend.PutGraph(ctx, "any", g, time.Hour); err != nil {
		t.Fatal(err)
	}

	// Still should not find it
	_, found, err = backend.GetGraph(ctx, "any")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("NullBackend should never find anything")
	}

	// Close should work
	if err := backend.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestLocalIndexExpiration(t *testing.T) {
	// Entry that expires in the future
	future := &localIndexEntry{
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if future.isExpired() {
		t.Error("entry should not be expired")
	}

	// Entry that expired in the past
	past := &localIndexEntry{
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	if !past.isExpired() {
		t.Error("entry should be expired")
	}
}
