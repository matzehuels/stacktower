package javascript

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/deps"
)

func TestPackageJSON_Supports(t *testing.T) {
	parser := &PackageJSON{}

	tests := []struct {
		filename string
		want     bool
	}{
		{"package.json", true},
		{"Package.json", true},
		{"PACKAGE.JSON", true},
		{"Cargo.toml", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			if got := parser.Supports(tt.filename); got != tt.want {
				t.Errorf("Supports(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestPackageJSON_Parse(t *testing.T) {
	dir := t.TempDir()
	pkgFile := filepath.Join(dir, "package.json")
	content := `{
  "name": "my-package",
  "version": "1.0.0",
  "dependencies": {
    "express": "^4.18.0",
    "lodash": "^4.17.21"
  },
  "devDependencies": {
    "jest": "^29.0.0"
  }
}`

	if err := os.WriteFile(pkgFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	parser := &PackageJSON{}
	result, err := parser.Parse(pkgFile, deps.Options{})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	g := result.Graph.(*dag.DAG)

	// project root + my-package + 3 deps
	if got := g.NodeCount(); got != 5 {
		t.Errorf("NodeCount = %d, want 5", got)
	}

	for _, dep := range []string{"express", "lodash", "jest"} {
		if _, ok := g.Node(dep); !ok {
			t.Errorf("expected node %q not found", dep)
		}
	}

	if result.RootPackage != "my-package" {
		t.Errorf("RootPackage = %q, want %q", result.RootPackage, "my-package")
	}
}

func TestPackageJSON_Type(t *testing.T) {
	parser := &PackageJSON{}
	if got := parser.Type(); got != "package.json" {
		t.Errorf("Type() = %q, want %q", got, "package.json")
	}
}

func TestPackageJSON_IncludesTransitive(t *testing.T) {
	parser := &PackageJSON{}
	if parser.IncludesTransitive() {
		t.Error("IncludesTransitive() = true, want false (no resolver)")
	}
}
