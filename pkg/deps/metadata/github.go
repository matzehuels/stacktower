package metadata

import (
	"context"
	"time"

	"github.com/matzehuels/stacktower/pkg/deps"
	"github.com/matzehuels/stacktower/pkg/integrations/github"
)

type GitHub struct {
	client *github.Client
}

func NewGitHub(token string, cacheTTL time.Duration) (*GitHub, error) {
	c, err := github.NewClient(token, cacheTTL)
	if err != nil {
		return nil, err
	}
	return &GitHub{c}, nil
}

func (g *GitHub) Name() string { return "github" }

func (g *GitHub) Enrich(ctx context.Context, pkg *deps.PackageRef, refresh bool) (map[string]any, error) {
	owner, name, ok := github.ExtractURL(pkg.ProjectURLs, pkg.HomePage)
	if !ok && pkg.ManifestFile != "" {
		owner, name, ok = g.client.SearchPackageRepo(ctx, pkg.Name, pkg.ManifestFile)
	}
	if !ok {
		return nil, nil
	}

	m, err := g.client.Fetch(ctx, owner, name, refresh)
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		RepoURL:      m.RepoURL,
		RepoOwner:    m.Owner,
		RepoStars:    m.Stars,
		RepoArchived: m.Archived,
	}
	if m.Language != "" {
		result[RepoLanguage] = m.Language
	}
	if len(m.Topics) > 0 {
		result[RepoTopics] = m.Topics
	}
	if m.LastCommitAt != nil {
		result[RepoLastCommit] = m.LastCommitAt.Format("2006-01-02")
	}
	if m.LastReleaseAt != nil {
		result[RepoLastRelease] = m.LastReleaseAt.Format("2006-01-02")
	}
	if len(m.Contributors) > 0 {
		maintainers := make([]string, len(m.Contributors))
		for i, c := range m.Contributors {
			maintainers[i] = c.Login
		}
		result[RepoMaintainers] = maintainers
	}
	return result, nil
}
