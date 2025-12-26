package golang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/deps"
)

func TestGoModParser_Supports(t *testing.T) {
	parser := &GoModParser{}

	tests := []struct {
		filename string
		want     bool
	}{
		{"go.mod", true},
		{"Go.mod", false},
		{"go.sum", false},
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

func TestGoModParser_Parse(t *testing.T) {
	dir := t.TempDir()
	goModFile := filepath.Join(dir, "go.mod")
	content := `module github.com/example/myapp

go 1.21

require (
	github.com/gin-gonic/gin v1.9.0
	github.com/spf13/cobra v1.7.0
	golang.org/x/sync v0.3.0 // indirect
)

require github.com/stretchr/testify v1.8.0
`

	if err := os.WriteFile(goModFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	parser := &GoModParser{} // No resolver = shallow parse
	result, err := parser.Parse(goModFile, deps.Options{})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	g := result.Graph.(*dag.DAG)

	// Should have project root + 3 direct dependencies
	if got := g.NodeCount(); got != 4 {
		t.Errorf("NodeCount = %d, want 4", got)
	}

	// Check that direct deps are included
	for _, dep := range []string{"github.com/gin-gonic/gin", "github.com/spf13/cobra", "github.com/stretchr/testify"} {
		if _, ok := g.Node(dep); !ok {
			t.Errorf("expected node %q not found", dep)
		}
	}

	// Check that indirect deps are excluded
	if _, ok := g.Node("golang.org/x/sync"); ok {
		t.Error("unexpected node golang.org/x/sync (should be filtered as indirect)")
	}

	// Verify root package
	if result.RootPackage != "github.com/example/myapp" {
		t.Errorf("RootPackage = %q, want %q", result.RootPackage, "github.com/example/myapp")
	}
}

func TestGoModParser_Type(t *testing.T) {
	parser := &GoModParser{}
	if got := parser.Type(); got != "go.mod" {
		t.Errorf("Type() = %q, want %q", got, "go.mod")
	}
}

func TestGoModParser_IncludesTransitive(t *testing.T) {
	parser := &GoModParser{}
	if parser.IncludesTransitive() {
		t.Error("IncludesTransitive() = true, want false (no resolver)")
	}
}

func TestParseGoModFile(t *testing.T) {
	content := `module github.com/example/app

go 1.21

require (
	github.com/gin-gonic/gin v1.9.0
	github.com/gin-gonic/gin v1.9.0
	golang.org/x/sync v0.3.0 // indirect
)
`
	dir := t.TempDir()
	path := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	f, _ := os.Open(path)
	defer f.Close()

	mod, deps := parseGoModFile(f)

	if mod != "github.com/example/app" {
		t.Errorf("module = %q, want github.com/example/app", mod)
	}

	// Should dedupe and filter indirect
	if len(deps) != 1 {
		t.Errorf("expected 1 dep, got %d: %v", len(deps), deps)
	}
}

func TestParseRequireLine_Variations(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"github.com/pkg/errors v0.9.1", "github.com/pkg/errors"},
		{"golang.org/x/sync v0.3.0 // indirect", ""},
		{"  github.com/spf13/cobra v1.7.0  ", "github.com/spf13/cobra"},
		{"github.com/example/pkg v1.0.0 // some other comment", "github.com/example/pkg"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		name := tt.line
		if name == "" {
			name = "empty"
		} else if strings.TrimSpace(name) == "" {
			name = "whitespace"
		}
		t.Run(name, func(t *testing.T) {
			if got := parseRequireLine(tt.line); got != tt.want {
				t.Errorf("parseRequireLine(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}
