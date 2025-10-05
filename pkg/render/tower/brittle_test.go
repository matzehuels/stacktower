package tower

import (
	"testing"
	"time"

	"stacktower/pkg/dag"
)

func TestIsBrittle(t *testing.T) {
	threeYearsAgo := time.Now().AddDate(-3, 0, 0).Format("2006-01-02")
	eighteenMonthsAgo := time.Now().AddDate(-1, -6, 0).Format("2006-01-02")
	fourteenMonthsAgo := time.Now().AddDate(-1, -2, 0).Format("2006-01-02")
	threeMonthsAgo := time.Now().AddDate(0, -3, 0).Format("2006-01-02")
	oneMonthAgo := time.Now().AddDate(0, -1, 0).Format("2006-01-02")

	cases := []struct {
		name string
		node *dag.Node
		want bool
	}{
		{"nil node", nil, false},
		{"no metadata", &dag.Node{ID: "pkg"}, false},
		{"empty metadata", &dag.Node{ID: "pkg", Meta: dag.Metadata{}}, false},
		{
			"archived",
			&dag.Node{ID: "pkg", Meta: dag.Metadata{"repo_archived": true}},
			true,
		},
		{
			"abandoned (3 years stale)",
			&dag.Node{ID: "pkg", Meta: dag.Metadata{
				"repo_last_commit": threeYearsAgo,
				"repo_stars":       5000,
				"repo_maintainers": []string{"a", "b", "c", "d", "e"},
			}},
			true,
		},
		{
			"bus factor (single maintainer + stale)",
			&dag.Node{ID: "pkg", Meta: dag.Metadata{
				"repo_last_commit": eighteenMonthsAgo,
				"repo_stars":       10000,
				"repo_maintainers": []string{"solo-dev"},
			}},
			true,
		},
		{
			"stagnant with low stars",
			&dag.Node{ID: "pkg", Meta: dag.Metadata{
				"repo_last_commit": fourteenMonthsAgo,
				"repo_stars":       50,
				"repo_maintainers": []string{"a", "b", "c"},
			}},
			true,
		},
		{
			"stagnant with few maintainers",
			&dag.Node{ID: "pkg", Meta: dag.Metadata{
				"repo_last_commit": fourteenMonthsAgo,
				"repo_stars":       5000,
				"repo_maintainers": []string{"a", "b"},
			}},
			true,
		},
		{
			"healthy (active, many stars, many maintainers)",
			&dag.Node{ID: "pkg", Meta: dag.Metadata{
				"repo_last_commit": threeMonthsAgo,
				"repo_stars":       5000,
				"repo_maintainers": []string{"a", "b", "c", "d", "e"},
			}},
			false,
		},
		{
			"active trumps low stars",
			&dag.Node{ID: "pkg", Meta: dag.Metadata{
				"repo_last_commit": oneMonthAgo,
				"repo_stars":       20,
				"repo_maintainers": []string{"a"},
			}},
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsBrittle(tc.node); got != tc.want {
				t.Errorf("IsBrittle() = %v, want %v", got, tc.want)
			}
		})
	}
}
