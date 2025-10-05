package metadata

import (
	"context"
	"time"

	"stacktower/pkg/integrations/gitlab"
	"stacktower/pkg/source"
)

type GitLab struct {
	client *gitlab.Client
}

func NewGitLab(token string, cacheTTL time.Duration) (*GitLab, error) {
	c, err := gitlab.NewClient(token, cacheTTL)
	if err != nil {
		return nil, err
	}
	return &GitLab{c}, nil
}

func (g *GitLab) Name() string { return "gitlab" }

func (g *GitLab) Enrich(ctx context.Context, repo *source.RepoInfo, refresh bool) (map[string]any, error) {
	_, _, ok := gitlab.ExtractURL(repo.ProjectURLs, repo.HomePage)
	if !ok {
		return nil, nil
	}
	return nil, nil
}
