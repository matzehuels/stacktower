package rust

import (
	"context"
	"os"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/deps"
)

const projectRoot = "__project__"

type CargoToml struct {
	resolver deps.Resolver
}

func (c *CargoToml) Type() string              { return "Cargo.toml" }
func (c *CargoToml) IncludesTransitive() bool  { return c.resolver != nil }
func (c *CargoToml) Supports(name string) bool { return strings.EqualFold(name, "cargo.toml") }

func (c *CargoToml) Parse(path string, opts deps.Options) (*deps.ManifestResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cargo cargoFile
	if err := toml.Unmarshal(data, &cargo); err != nil {
		return nil, err
	}

	directDeps := extractCargoDeps(cargo)

	var g *dag.DAG
	if c.resolver != nil {
		g, err = c.resolve(context.Background(), directDeps, opts)
		if err != nil {
			return nil, err
		}
	} else {
		g = shallowCargoGraph(directDeps)
	}

	rootPackage := cargo.Package.Name
	if rootPackage != "" {
		if root, ok := g.Node(projectRoot); ok {
			root.Meta["version"] = cargo.Package.Version
		}
	}

	return &deps.ManifestResult{
		Graph:              g,
		Type:               c.Type(),
		IncludesTransitive: c.resolver != nil,
		RootPackage:        rootPackage,
	}, nil
}

func (c *CargoToml) resolve(ctx context.Context, pkgs []string, opts deps.Options) (*dag.DAG, error) {
	merged := dag.New(nil)
	_ = merged.AddNode(dag.Node{ID: projectRoot, Meta: dag.Metadata{"virtual": true}})

	for _, pkg := range pkgs {
		g, err := c.resolver.Resolve(ctx, pkg, opts)
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

func extractCargoDeps(cargo cargoFile) []string {
	var deps []string
	for name := range cargo.Dependencies {
		deps = append(deps, name)
	}
	for name := range cargo.DevDependencies {
		deps = append(deps, name)
	}
	for name := range cargo.BuildDependencies {
		deps = append(deps, name)
	}
	return deps
}

func shallowCargoGraph(pkgs []string) *dag.DAG {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: projectRoot, Meta: dag.Metadata{"virtual": true}})
	for _, pkg := range pkgs {
		_ = g.AddNode(dag.Node{ID: pkg})
		_ = g.AddEdge(dag.Edge{From: projectRoot, To: pkg})
	}
	return g
}

type cargoFile struct {
	Package struct {
		Name    string `toml:"name"`
		Version string `toml:"version"`
	} `toml:"package"`
	Dependencies      map[string]any `toml:"dependencies"`
	DevDependencies   map[string]any `toml:"dev-dependencies"`
	BuildDependencies map[string]any `toml:"build-dependencies"`
}
