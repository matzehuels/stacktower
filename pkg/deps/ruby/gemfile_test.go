package ruby

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/deps"
)

func TestGemfile_Supports(t *testing.T) {
	parser := &Gemfile{}

	tests := []struct {
		filename string
		want     bool
	}{
		{"Gemfile", true},
		{"gemfile", false},
		{"Gemfile.lock", false},
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

func TestGemfile_Parse(t *testing.T) {
	dir := t.TempDir()
	gemfile := filepath.Join(dir, "Gemfile")
	content := `source 'https://rubygems.org'

# Web framework
gem 'rails', '~> 7.0'
gem 'puma', '>= 5.0'

group :development, :test do
  gem 'rspec-rails'
  gem 'factory_bot_rails'
end
`

	if err := os.WriteFile(gemfile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	parser := &Gemfile{}
	result, err := parser.Parse(gemfile, deps.Options{})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	g := result.Graph.(*dag.DAG)

	// project root + 4 gems
	if got := g.NodeCount(); got != 5 {
		t.Errorf("NodeCount = %d, want 5", got)
	}

	for _, dep := range []string{"rails", "puma", "rspec-rails", "factory_bot_rails"} {
		if _, ok := g.Node(dep); !ok {
			t.Errorf("expected node %q not found", dep)
		}
	}
}

func TestGemfile_Type(t *testing.T) {
	parser := &Gemfile{}
	if got := parser.Type(); got != "Gemfile" {
		t.Errorf("Type() = %q, want %q", got, "Gemfile")
	}
}

func TestGemfile_IncludesTransitive(t *testing.T) {
	parser := &Gemfile{}
	if parser.IncludesTransitive() {
		t.Error("IncludesTransitive() = true, want false (no resolver)")
	}
}

func TestParseGemfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Gemfile")
	content := `gem 'rails'
gem "puma"
gem 'rails'  # duplicate should be ignored
# gem 'commented_out'
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	f, _ := os.Open(path)
	defer f.Close()

	gems := parseGemfile(f)

	if len(gems) != 2 {
		t.Errorf("expected 2 gems, got %d: %v", len(gems), gems)
	}
}
