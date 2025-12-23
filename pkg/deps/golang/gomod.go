package golang

import (
	"bufio"
	"context"
	"os"
	"strings"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/deps"
)

const projectRoot = "__project__"

// GoModParser parses go.mod files. It extracts direct dependencies and
// optionally resolves them via the Go Module Proxy if a [deps.Resolver]
// is provided.
type GoModParser struct {
	resolver deps.Resolver
}

func (p *GoModParser) Type() string              { return "go.mod" }
func (p *GoModParser) IncludesTransitive() bool  { return p.resolver != nil }
func (p *GoModParser) Supports(name string) bool { return name == "go.mod" }

func (p *GoModParser) Parse(path string, opts deps.Options) (*deps.ManifestResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	moduleName, directDeps := parseGoModFile(f)

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
		RootPackage:        moduleName,
	}, nil
}

func (p *GoModParser) resolve(ctx context.Context, pkgs []string, opts deps.Options) (*dag.DAG, error) {
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

func parseGoModFile(f *os.File) (moduleName string, deps []string) {
	seen := make(map[string]bool)
	inRequire := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Extract module name
		if strings.HasPrefix(line, "module ") {
			moduleName = strings.TrimPrefix(line, "module ")
			moduleName = strings.TrimSpace(moduleName)
			continue
		}

		// Handle require block
		if strings.HasPrefix(line, "require (") || line == "require(" {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}

		// Single-line require
		if strings.HasPrefix(line, "require ") && !strings.Contains(line, "(") {
			line = strings.TrimPrefix(line, "require ")
		} else if !inRequire {
			continue
		}

		// Parse module path from require line
		if dep := parseRequireLine(line); dep != "" && !seen[dep] {
			seen[dep] = true
			deps = append(deps, dep)
		}
	}

	return moduleName, deps
}

func parseRequireLine(line string) string {
	// Skip indirect dependencies
	if strings.Contains(line, "// indirect") {
		return ""
	}

	// Remove inline comments
	if idx := strings.Index(line, "//"); idx != -1 {
		line = line[:idx]
	}

	line = strings.TrimSpace(line)
	fields := strings.Fields(line)
	if len(fields) >= 1 {
		return fields[0]
	}
	return ""
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
