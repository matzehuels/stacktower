package python

import (
	"bufio"
	"context"
	"os"
	"regexp"
	"strings"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/deps"
)

var depNameRE = regexp.MustCompile(`^([a-zA-Z0-9][-a-zA-Z0-9._]*)`)

type Requirements struct {
	resolver deps.Resolver
}

func (r *Requirements) Type() string             { return "requirements.txt" }
func (r *Requirements) IncludesTransitive() bool { return r.resolver != nil }

func (r *Requirements) Supports(name string) bool {
	return name == "requirements.txt" ||
		(strings.HasPrefix(name, "requirements") && strings.HasSuffix(name, ".txt"))
}

func (r *Requirements) Parse(path string, opts deps.Options) (*deps.ManifestResult, error) {
	pkgs, err := parseFile(path)
	if err != nil {
		return nil, err
	}

	var g *dag.DAG
	if r.resolver != nil {
		g, err = r.resolve(context.Background(), pkgs, opts)
		if err != nil {
			return nil, err
		}
	} else {
		g = shallow(pkgs)
	}

	return &deps.ManifestResult{
		Graph:              g,
		Type:               r.Type(),
		IncludesTransitive: r.resolver != nil,
	}, nil
}

func (r *Requirements) resolve(ctx context.Context, pkgs []string, opts deps.Options) (*dag.DAG, error) {
	merged := dag.New(nil)
	_ = merged.AddNode(dag.Node{ID: projectRoot, Meta: dag.Metadata{"virtual": true}})

	for _, pkg := range pkgs {
		g, err := r.resolver.Resolve(ctx, pkg, opts)
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

func parseFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	seen := make(map[string]bool)
	var result []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line[0] == '#' || line[0] == '-' {
			continue
		}
		if strings.Contains(line, "://") || strings.HasPrefix(line, "git+") {
			continue
		}
		if m := depNameRE.FindStringSubmatch(line); len(m) > 1 {
			name := normalize(m[1])
			if !seen[name] {
				seen[name] = true
				result = append(result, name)
			}
		}
	}

	return result, scanner.Err()
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
