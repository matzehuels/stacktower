package tower

import (
	"testing"

	"stacktower/pkg/dag"
)

func TestRankNebraska_DeepDepsScoreHigher(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "app", Row: 0, Meta: dag.Metadata{
		"repo_maintainers": []string{"alice"},
	}})
	_ = g.AddNode(dag.Node{ID: "lib", Row: 1, Meta: dag.Metadata{
		"repo_maintainers": []string{"bob"},
	}})
	_ = g.AddNode(dag.Node{ID: "deep", Row: 2, Meta: dag.Metadata{
		"repo_maintainers": []string{"nebraska-guy"},
	}})
	_ = g.AddEdge(dag.Edge{From: "app", To: "lib"})
	_ = g.AddEdge(dag.Edge{From: "lib", To: "deep"})

	rankings := RankNebraska(g, 5)

	if len(rankings) != 2 {
		t.Fatalf("expected 2 rankings, got %d", len(rankings))
	}
	if rankings[0].Maintainer != "nebraska-guy" {
		t.Errorf("expected nebraska-guy to rank first, got %s", rankings[0].Maintainer)
	}
	if rankings[0].Score <= rankings[1].Score {
		t.Error("deeper dependency should have higher score")
	}
}

func TestRankNebraska_LeadScoresHigherThanMaintainer(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "root", Row: 0, Meta: dag.Metadata{}})
	_ = g.AddNode(dag.Node{ID: "pkg", Row: 1, Meta: dag.Metadata{
		"repo_maintainers": []string{"alice", "bob"},
	}})
	_ = g.AddEdge(dag.Edge{From: "root", To: "pkg"})

	rankings := RankNebraska(g, 5)

	if len(rankings) != 2 {
		t.Fatalf("expected 2 rankings, got %d", len(rankings))
	}
	if rankings[0].Maintainer != "alice" {
		t.Errorf("expected alice (lead) to rank first, got %s", rankings[0].Maintainer)
	}
	if rankings[0].Score <= rankings[1].Score {
		t.Error("lead should have higher score than maintainer")
	}
}

func TestRankNebraska_AggregatesAcrossPackages(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "root", Row: 0, Meta: dag.Metadata{
		"repo_maintainers": []string{"prolific-dev"},
	}})
	_ = g.AddNode(dag.Node{ID: "mid", Row: 1, Meta: dag.Metadata{
		"repo_maintainers": []string{"prolific-dev"},
	}})
	_ = g.AddNode(dag.Node{ID: "deep", Row: 2, Meta: dag.Metadata{
		"repo_maintainers": []string{"prolific-dev"},
	}})
	_ = g.AddEdge(dag.Edge{From: "root", To: "mid"})
	_ = g.AddEdge(dag.Edge{From: "mid", To: "deep"})

	rankings := RankNebraska(g, 5)

	if len(rankings) != 1 {
		t.Fatalf("expected 1 ranking, got %d", len(rankings))
	}
	if len(rankings[0].Packages) != 2 {
		t.Errorf("expected 2 packages (root excluded), got %d", len(rankings[0].Packages))
	}
}

func TestRankNebraska_FallsBackToOwner(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "root", Row: 0, Meta: dag.Metadata{}})
	_ = g.AddNode(dag.Node{ID: "dep", Row: 1, Meta: dag.Metadata{
		"repo_owner": "owner-only",
	}})
	_ = g.AddEdge(dag.Edge{From: "root", To: "dep"})

	rankings := RankNebraska(g, 5)

	if len(rankings) != 1 {
		t.Fatalf("expected 1 ranking, got %d", len(rankings))
	}
	if rankings[0].Maintainer != "owner-only" {
		t.Errorf("expected owner-only, got %s", rankings[0].Maintainer)
	}
	if rankings[0].Packages[0].Role != RoleOwner {
		t.Errorf("expected owner role, got %s", rankings[0].Packages[0].Role)
	}
}

func TestRankNebraska_AssignsRoles(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "root", Row: 0, Meta: dag.Metadata{}})
	_ = g.AddNode(dag.Node{ID: "pkg", Row: 1, Meta: dag.Metadata{
		"repo_owner":       "alice",
		"repo_maintainers": []string{"alice", "bob", "carol"},
	}})
	_ = g.AddEdge(dag.Edge{From: "root", To: "pkg"})

	rankings := RankNebraska(g, 5)

	roles := make(map[string]Role)
	for _, r := range rankings {
		roles[r.Maintainer] = r.Packages[0].Role
	}

	if roles["alice"] != RoleOwner {
		t.Errorf("alice should be owner, got %s", roles["alice"])
	}
	if roles["bob"] != RoleLead {
		t.Errorf("bob should be lead (first non-owner), got %s", roles["bob"])
	}
	if roles["carol"] != RoleMaintainer {
		t.Errorf("carol should be maintainer, got %s", roles["carol"])
	}

	if rankings[0].Maintainer != "alice" {
		t.Errorf("owner should rank first, got %s", rankings[0].Maintainer)
	}
	if rankings[1].Maintainer != "bob" {
		t.Errorf("lead should rank second, got %s", rankings[1].Maintainer)
	}
}

func TestRankNebraska_SkipsSubdividers(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "root", Row: 0, Meta: dag.Metadata{}})
	_ = g.AddNode(dag.Node{ID: "pkg", Row: 1, Meta: dag.Metadata{
		"repo_maintainers": []string{"real"},
	}})
	_ = g.AddNode(dag.Node{ID: "pkg_sub", Row: 2, Kind: dag.NodeKindSubdivider, MasterID: "pkg", Meta: dag.Metadata{
		"repo_maintainers": []string{"fake"},
	}})
	_ = g.AddEdge(dag.Edge{From: "root", To: "pkg"})

	rankings := RankNebraska(g, 5)

	if len(rankings) != 1 {
		t.Fatalf("expected 1 ranking, got %d", len(rankings))
	}
	if rankings[0].Maintainer != "real" {
		t.Errorf("expected real, got %s", rankings[0].Maintainer)
	}
}

func TestRankNebraska_RespectsTopN(t *testing.T) {
	g := dag.New(nil)
	// a=row0 (root, excluded), f=row5 (deepest)
	ids := []string{"a", "b", "c", "d", "e", "f"}
	for i, name := range ids {
		_ = g.AddNode(dag.Node{ID: name, Row: i, Meta: dag.Metadata{
			"repo_maintainers": []string{name + "-maintainer"},
		}})
		if i > 0 {
			_ = g.AddEdge(dag.Edge{From: ids[i-1], To: name})
		}
	}

	rankings := RankNebraska(g, 3)

	if len(rankings) != 3 {
		t.Errorf("expected 3 rankings, got %d", len(rankings))
	}
	// Deepest (row 5 = "f") should be first, as depth = row - minRow = 5 - 0 = 5
	if rankings[0].Maintainer != "f-maintainer" {
		t.Errorf("expected f-maintainer first, got %s", rankings[0].Maintainer)
	}
}

func TestRankNebraska_EmptyGraph(t *testing.T) {
	g := dag.New(nil)
	rankings := RankNebraska(g, 5)
	if len(rankings) != 0 {
		t.Errorf("expected empty rankings, got %d", len(rankings))
	}
}

func TestRankNebraska_NoMetadata(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "pkg", Row: 0})

	rankings := RankNebraska(g, 5)
	if len(rankings) != 0 {
		t.Errorf("expected empty rankings, got %d", len(rankings))
	}
}
