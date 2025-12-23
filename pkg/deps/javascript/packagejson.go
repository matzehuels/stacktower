package javascript

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/deps"
)

const projectRoot = "__project__"

// PackageJSON parses package.json files. It extracts dependencies,
// devDependencies, and peerDependencies.
type PackageJSON struct {
	resolver deps.Resolver
}

func (p *PackageJSON) Type() string              { return "package.json" }
func (p *PackageJSON) IncludesTransitive() bool  { return p.resolver != nil }
func (p *PackageJSON) Supports(name string) bool { return strings.EqualFold(name, "package.json") }

func (p *PackageJSON) Parse(path string, opts deps.Options) (*deps.ManifestResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pkg packageFile
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	directDeps := extractPackageDeps(pkg)

	var g *dag.DAG
	if p.resolver != nil {
		g, err = p.resolve(context.Background(), directDeps, opts)
		if err != nil {
			return nil, err
		}
	} else {
		g = shallowPackageGraph(directDeps)
	}

	if pkg.Name != "" {
		if root, ok := g.Node(projectRoot); ok {
			root.Meta["version"] = pkg.Version
		}
	}

	return &deps.ManifestResult{
		Graph:              g,
		Type:               p.Type(),
		IncludesTransitive: p.resolver != nil,
		RootPackage:        pkg.Name,
	}, nil
}

func (p *PackageJSON) resolve(ctx context.Context, pkgs []string, opts deps.Options) (*dag.DAG, error) {
	merged := dag.New(nil)
	_ = merged.AddNode(dag.Node{ID: projectRoot, Meta: dag.Metadata{"virtual": true}})

	for _, pkg := range pkgs {
		g, err := p.resolver.Resolve(ctx, pkg, opts)
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

func extractPackageDeps(pkg packageFile) []string {
	var deps []string
	for name := range pkg.Dependencies {
		deps = append(deps, name)
	}
	for name := range pkg.DevDependencies {
		deps = append(deps, name)
	}
	for name := range pkg.PeerDependencies {
		deps = append(deps, name)
	}
	return deps
}

func shallowPackageGraph(pkgs []string) *dag.DAG {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: projectRoot, Meta: dag.Metadata{"virtual": true}})
	for _, pkg := range pkgs {
		_ = g.AddNode(dag.Node{ID: pkg})
		_ = g.AddEdge(dag.Edge{From: projectRoot, To: pkg})
	}
	return g
}

type packageFile struct {
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	Dependencies     map[string]string `json:"dependencies"`
	DevDependencies  map[string]string `json:"devDependencies"`
	PeerDependencies map[string]string `json:"peerDependencies"`
}
