package rust

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/deps"
)

func TestCargoToml_Supports(t *testing.T) {
	parser := &CargoToml{}

	tests := []struct {
		filename string
		want     bool
	}{
		{"Cargo.toml", true},
		{"cargo.toml", true},
		{"CARGO.TOML", true},
		{"package.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			if got := parser.Supports(tt.filename); got != tt.want {
				t.Errorf("Supports(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestCargoToml_Parse(t *testing.T) {
	dir := t.TempDir()
	cargoFile := filepath.Join(dir, "Cargo.toml")
	content := `[package]
name = "my-crate"
version = "0.1.0"

[dependencies]
serde = "1.0"
tokio = { version = "1.0", features = ["full"] }

[dev-dependencies]
pretty_assertions = "1.0"
`

	if err := os.WriteFile(cargoFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	parser := &CargoToml{}
	result, err := parser.Parse(cargoFile, deps.Options{})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	g := result.Graph.(*dag.DAG)

	if got := g.NodeCount(); got != 4 {
		t.Errorf("NodeCount = %d, want 4", got)
	}

	for _, dep := range []string{"serde", "tokio", "pretty_assertions"} {
		if _, ok := g.Node(dep); !ok {
			t.Errorf("expected node %q not found", dep)
		}
	}

	if result.RootPackage != "my-crate" {
		t.Errorf("RootPackage = %q, want %q", result.RootPackage, "my-crate")
	}

	// Verify version metadata is on the root node
	if root, ok := g.Node("__project__"); ok {
		if root.Meta["version"] != "0.1.0" {
			t.Errorf("root node version = %v, want 0.1.0", root.Meta["version"])
		}
	} else {
		t.Error("__project__ node not found")
	}
}

func TestCargoToml_Type(t *testing.T) {
	parser := &CargoToml{}
	if got := parser.Type(); got != "Cargo.toml" {
		t.Errorf("Type() = %q, want %q", got, "Cargo.toml")
	}
}

func TestCargoToml_IncludesTransitive(t *testing.T) {
	parser := &CargoToml{}
	if parser.IncludesTransitive() {
		t.Error("IncludesTransitive() = true, want false (no resolver)")
	}
}
