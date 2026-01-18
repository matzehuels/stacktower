package feature

import (
	"cmp"
	"slices"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/deps/metadata"
)

type Role string

const (
	RoleOwner      Role = "owner"
	RoleLead       Role = "lead"
	RoleMaintainer Role = "maintainer"
)

type PackageRole struct {
	Package string
	Role    Role
	URL     string
	Depth   int
}

type NebraskaRanking struct {
	Maintainer string
	Score      float64
	Packages   []PackageRole
}

const (
	ownerWeight      = 3.0
	leadWeight       = 1.5
	maintainerWeight = 1.0
)

// RankNebraska identifies the most influential maintainers in the dependency
// graph using the Nebraska ranking algorithm. Maintainers are scored based
// on the "depth" of their packages in the tower (i.e., how many things
// depend on them).
func RankNebraska(g *dag.DAG, topN int) []NebraskaRanking {
	scores := make(map[string]float64)
	packages := make(map[string][]PackageRole)
	bestRole := make(map[string]Role)
	minRow := findMinRow(g)

	for _, n := range g.Nodes() {
		if n.IsSynthetic() || g.InDegree(n.ID) == 0 {
			continue
		}

		roles := getMaintainerRoles(n)
		if len(roles) == 0 {
			continue
		}

		depth := n.Row - minRow
		share := float64(depth) / float64(len(roles))

		for maintainer, role := range roles {
			scores[maintainer] += share * roleWeight(role)

			if !hasPackage(packages[maintainer], n.ID) {
				url, _ := n.Meta[metadata.RepoURL].(string)
				packages[maintainer] = append(packages[maintainer], PackageRole{
					Package: n.ID,
					Role:    role,
					URL:     url,
					Depth:   depth,
				})
			}

			if roleRank(role) < roleRank(bestRole[maintainer]) {
				bestRole[maintainer] = role
			}
		}
	}

	rankings := make([]NebraskaRanking, 0, len(scores))
	for m, score := range scores {
		pkgs := packages[m]
		slices.SortFunc(pkgs, func(a, b PackageRole) int {
			if c := cmp.Compare(roleRank(a.Role), roleRank(b.Role)); c != 0 {
				return c
			}
			if c := cmp.Compare(b.Depth, a.Depth); c != 0 {
				return c
			}
			return cmp.Compare(a.Package, b.Package)
		})
		rankings = append(rankings, NebraskaRanking{
			Maintainer: m,
			Score:      score,
			Packages:   pkgs,
		})
	}

	slices.SortFunc(rankings, func(a, b NebraskaRanking) int {
		if c := cmp.Compare(b.Score, a.Score); c != 0 {
			return c
		}
		if c := cmp.Compare(roleRank(bestRole[a.Maintainer]), roleRank(bestRole[b.Maintainer])); c != 0 {
			return c
		}
		return cmp.Compare(a.Maintainer, b.Maintainer)
	})

	if len(rankings) > topN {
		return rankings[:topN]
	}
	return rankings
}

func roleRank(r Role) int {
	switch r {
	case RoleOwner:
		return 0
	case RoleLead:
		return 1
	case RoleMaintainer:
		return 2
	default:
		return 3
	}
}

func roleWeight(r Role) float64 {
	switch r {
	case RoleOwner:
		return ownerWeight
	case RoleLead:
		return leadWeight
	default:
		return maintainerWeight
	}
}

func findMinRow(g *dag.DAG) int {
	minRow := -1
	for _, n := range g.Nodes() {
		if !n.IsSynthetic() && (minRow < 0 || n.Row < minRow) {
			minRow = n.Row
		}
	}
	return max(0, minRow)
}

func getMaintainerRoles(n *dag.Node) map[string]Role {
	if n.Meta == nil {
		return nil
	}

	owner, _ := n.Meta[metadata.RepoOwner].(string)
	maintainers := getStringSlice(n.Meta[metadata.RepoMaintainers])

	if len(maintainers) == 0 && owner != "" {
		return map[string]Role{owner: RoleOwner}
	}

	roles := make(map[string]Role, len(maintainers))
	leadAssigned := false

	for _, m := range maintainers {
		switch {
		case m == owner:
			roles[m] = RoleOwner
		case !leadAssigned:
			roles[m] = RoleLead
			leadAssigned = true
		default:
			roles[m] = RoleMaintainer
		}
	}
	return roles
}

func getStringSlice(v any) []string {
	switch v := v.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func hasPackage(pkgs []PackageRole, id string) bool {
	return slices.ContainsFunc(pkgs, func(p PackageRole) bool { return p.Package == id })
}
