package php

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/deps"
)

const projectRoot = "__project__"

// ComposerJSON parses composer.json files. It extracts direct and dev
// dependencies and optionally resolves them via Packagist.
type ComposerJSON struct {
	resolver deps.Resolver
}

func (c *ComposerJSON) Type() string              { return "composer.json" }
func (c *ComposerJSON) IncludesTransitive() bool  { return c.resolver != nil }
func (c *ComposerJSON) Supports(name string) bool { return strings.EqualFold(name, "composer.json") }

func (c *ComposerJSON) Parse(path string, opts deps.Options) (*deps.ManifestResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var comp composerFile
	if err := json.Unmarshal(data, &comp); err != nil {
		return nil, err
	}

	directDeps := extractComposerDeps(comp)

	var g *dag.DAG
	if c.resolver != nil {
		g, err = c.resolve(context.Background(), directDeps, opts)
		if err != nil {
			return nil, err
		}
	} else {
		g = shallowComposerGraph(directDeps)
	}

	if comp.Name != "" {
		if root, ok := g.Node(projectRoot); ok {
			root.Meta["version"] = comp.Version
		}
	}

	return &deps.ManifestResult{
		Graph:              g,
		Type:               c.Type(),
		IncludesTransitive: c.resolver != nil,
		RootPackage:        comp.Name,
	}, nil
}

func (c *ComposerJSON) resolve(ctx context.Context, pkgs []string, opts deps.Options) (*dag.DAG, error) {
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

func extractComposerDeps(comp composerFile) []string {
	var deps []string
	for name := range comp.Require {
		if isPHPRequirement(name) {
			continue
		}
		deps = append(deps, name)
	}
	for name := range comp.RequireDev {
		if isPHPRequirement(name) {
			continue
		}
		deps = append(deps, name)
	}
	return deps
}

func isPHPRequirement(name string) bool {
	return name == "php" || strings.HasPrefix(name, "php-") || strings.HasPrefix(name, "ext-")
}

func shallowComposerGraph(pkgs []string) *dag.DAG {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: projectRoot, Meta: dag.Metadata{"virtual": true}})
	for _, pkg := range pkgs {
		_ = g.AddNode(dag.Node{ID: pkg})
		_ = g.AddEdge(dag.Edge{From: projectRoot, To: pkg})
	}
	return g
}

type composerFile struct {
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	Require    map[string]string `json:"require"`
	RequireDev map[string]string `json:"require-dev"`
}
