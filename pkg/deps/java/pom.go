package java

import (
	"context"
	"encoding/xml"
	"os"
	"strings"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/deps"
)

const projectRoot = "__project__"

type POMParser struct {
	resolver deps.Resolver
}

func (p *POMParser) Type() string              { return "pom.xml" }
func (p *POMParser) IncludesTransitive() bool  { return p.resolver != nil }
func (p *POMParser) Supports(name string) bool { return name == "pom.xml" }

func (p *POMParser) Parse(path string, opts deps.Options) (*deps.ManifestResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pom pomProject
	if err := xml.Unmarshal(data, &pom); err != nil {
		return nil, err
	}

	directDeps := extractDependencies(&pom)

	var g *dag.DAG
	if p.resolver != nil {
		g, err = p.resolve(context.Background(), directDeps, opts)
		if err != nil {
			return nil, err
		}
	} else {
		g = shallow(directDeps)
	}

	return &deps.ManifestResult{
		Graph:              g,
		Type:               p.Type(),
		IncludesTransitive: p.resolver != nil,
		RootPackage:        pom.GroupID + ":" + pom.ArtifactID,
	}, nil
}

func (p *POMParser) resolve(ctx context.Context, pkgs []string, opts deps.Options) (*dag.DAG, error) {
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

func extractDependencies(pom *pomProject) []string {
	var deps []string
	seen := make(map[string]bool)

	for _, dep := range pom.Dependencies {
		// Skip test and provided scope dependencies
		if dep.Scope == "test" || dep.Scope == "provided" || dep.Optional == "true" {
			continue
		}
		// Skip dependencies with unresolved Maven properties
		if strings.HasPrefix(dep.GroupID, "${") || strings.HasPrefix(dep.ArtifactID, "${") {
			continue
		}
		coord := dep.GroupID + ":" + dep.ArtifactID
		if !seen[coord] {
			seen[coord] = true
			deps = append(deps, coord)
		}
	}
	return deps
}

func shallow(pkgs []string) *dag.DAG {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: projectRoot, Meta: dag.Metadata{"virtual": true}})
	for _, pkg := range pkgs {
		_ = g.AddNode(dag.Node{ID: pkg})
		_ = g.AddEdge(dag.Edge{From: projectRoot, To: pkg})
	}
	return g
}

type pomProject struct {
	GroupID      string          `xml:"groupId"`
	ArtifactID   string          `xml:"artifactId"`
	Version      string          `xml:"version"`
	Name         string          `xml:"name"`
	Description  string          `xml:"description"`
	URL          string          `xml:"url"`
	Dependencies []pomDependency `xml:"dependencies>dependency"`
	Parent       *pomParent      `xml:"parent"`
}

type pomParent struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
}

type pomDependency struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
	Scope      string `xml:"scope"`
	Optional   string `xml:"optional"`
}
