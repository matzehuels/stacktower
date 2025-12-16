package ruby

import (
	"bufio"
	"context"
	"os"
	"regexp"
	"strings"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/deps"
)

const projectRoot = "__project__"

type Gemfile struct {
	resolver deps.Resolver
}

func (g *Gemfile) Type() string              { return "Gemfile" }
func (g *Gemfile) IncludesTransitive() bool  { return g.resolver != nil }
func (g *Gemfile) Supports(name string) bool { return name == "Gemfile" }

func (gf *Gemfile) Parse(path string, opts deps.Options) (*deps.ManifestResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	directDeps := parseGemfile(f)

	var g *dag.DAG
	if gf.resolver != nil {
		g, err = gf.resolve(context.Background(), directDeps, opts)
		if err != nil {
			return nil, err
		}
	} else {
		g = shallowGemGraph(directDeps)
	}

	return &deps.ManifestResult{
		Graph:              g,
		Type:               gf.Type(),
		IncludesTransitive: gf.resolver != nil,
	}, nil
}

func (gf *Gemfile) resolve(ctx context.Context, pkgs []string, opts deps.Options) (*dag.DAG, error) {
	merged := dag.New(nil)
	_ = merged.AddNode(dag.Node{ID: projectRoot, Meta: dag.Metadata{"virtual": true}})

	for _, pkg := range pkgs {
		g, err := gf.resolver.Resolve(ctx, pkg, opts)
		if err != nil {
			opts.Logger("resolve failed: %s: %v", pkg, err)
			_ = merged.AddNode(dag.Node{ID: pkg})
			_ = merged.AddEdge(dag.Edge{From: projectRoot, To: pkg})
			continue
		}
		for _, n := range g.Nodes() {
			_ = merged.AddNode(dag.Node{ID: n.ID, Meta: n.Meta})
		}
		for _, e := range g.Edges() {
			_ = merged.AddEdge(dag.Edge{From: e.From, To: e.To})
		}
		_ = merged.AddEdge(dag.Edge{From: projectRoot, To: pkg})
	}

	return merged, nil
}

var gemPattern = regexp.MustCompile(`^\s*gem\s+['"]([^'"]+)['"]`)

func parseGemfile(f *os.File) []string {
	var gems []string
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		if match := gemPattern.FindStringSubmatch(line); len(match) > 1 {
			name := match[1]
			if !seen[name] {
				seen[name] = true
				gems = append(gems, name)
			}
		}
	}

	return gems
}

func shallowGemGraph(pkgs []string) *dag.DAG {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: projectRoot, Meta: dag.Metadata{"virtual": true}})
	for _, pkg := range pkgs {
		_ = g.AddNode(dag.Node{ID: pkg})
		_ = g.AddEdge(dag.Edge{From: projectRoot, To: pkg})
	}
	return g
}
