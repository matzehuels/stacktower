package python

import (
	"context"
	"maps"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/deps"
)

const projectRoot = "__project__"

// PoetryLock parses poetry.lock files. It provides a full transitive closure
// of the dependency graph without needing to contact a registry.
type PoetryLock struct{}

func (p *PoetryLock) Type() string              { return "poetry.lock" }
func (p *PoetryLock) IncludesTransitive() bool  { return true }
func (p *PoetryLock) Supports(name string) bool { return name == "poetry.lock" }

func (p *PoetryLock) Parse(path string, opts deps.Options) (*deps.ManifestResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lock lockFile
	if err := toml.Unmarshal(data, &lock); err != nil {
		return nil, err
	}

	g := buildGraph(lock.Packages)
	enrich(context.Background(), g, opts)

	return &deps.ManifestResult{
		Graph:              g,
		Type:               p.Type(),
		IncludesTransitive: true,
		RootPackage:        extractPyprojectName(filepath.Dir(path)),
	}, nil
}

func extractPyprojectName(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	if err != nil {
		return ""
	}
	var pyproject struct {
		Tool struct {
			Poetry struct {
				Name string `toml:"name"`
			} `toml:"poetry"`
		} `toml:"tool"`
		Project struct {
			Name string `toml:"name"`
		} `toml:"project"`
	}
	if err := toml.Unmarshal(data, &pyproject); err != nil {
		return ""
	}
	if pyproject.Tool.Poetry.Name != "" {
		return pyproject.Tool.Poetry.Name
	}
	return pyproject.Project.Name
}

type lockFile struct {
	Packages []lockPackage `toml:"package"`
}

type lockPackage struct {
	Name         string         `toml:"name"`
	Version      string         `toml:"version"`
	Description  string         `toml:"description"`
	Category     string         `toml:"category"`
	Dependencies map[string]any `toml:"dependencies"`
}

func buildGraph(packages []lockPackage) *dag.DAG {
	g := dag.New(nil)
	pkgs := make(map[string]bool, len(packages))

	for _, pkg := range packages {
		name := normalize(pkg.Name)
		pkgs[name] = true
		meta := dag.Metadata{"version": pkg.Version}
		if pkg.Description != "" {
			meta["description"] = pkg.Description
		}
		if pkg.Category != "" {
			meta["category"] = pkg.Category
		}
		_ = g.AddNode(dag.Node{ID: name, Meta: meta})
	}

	incoming := make(map[string]bool)
	for _, pkg := range packages {
		from := normalize(pkg.Name)
		for dep := range pkg.Dependencies {
			to := normalize(dep)
			if pkgs[to] {
				_ = g.AddEdge(dag.Edge{From: from, To: to})
				incoming[to] = true
			}
		}
	}

	_ = g.AddNode(dag.Node{ID: projectRoot, Meta: dag.Metadata{"virtual": true}})
	for _, pkg := range packages {
		name := normalize(pkg.Name)
		if !incoming[name] {
			_ = g.AddEdge(dag.Edge{From: projectRoot, To: name})
		}
	}

	return g
}

func enrich(ctx context.Context, g *dag.DAG, opts deps.Options) {
	if len(opts.MetadataProviders) == 0 {
		return
	}
	for _, node := range g.Nodes() {
		if node.ID == projectRoot {
			continue
		}
		version, _ := node.Meta["version"].(string)
		ref := &deps.PackageRef{
			Name:         node.ID,
			Version:      version,
			ManifestFile: "pyproject.toml",
		}
		for _, p := range opts.MetadataProviders {
			if m, err := p.Enrich(ctx, ref, opts.Refresh); err == nil {
				maps.Copy(node.Meta, m)
			} else {
				opts.Logger("enrich failed: %s: %v", node.ID, err)
			}
		}
	}
}
