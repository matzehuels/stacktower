package integrations

import (
	"regexp"
	"testing"
)

func TestExtractRepoURL(t *testing.T) {
	githubPattern := regexp.MustCompile(`https?://github\.com/([^/]+)/([^/]+)`)
	gitlabPattern := regexp.MustCompile(`https?://gitlab\.com/([^/]+)/([^/]+)`)

	tests := []struct {
		name        string
		pattern     *regexp.Regexp
		projectURLs map[string]string
		homepage    string
		wantOwner   string
		wantRepo    string
		wantOK      bool
	}{
		{
			name:        "github from project urls",
			pattern:     githubPattern,
			projectURLs: map[string]string{"Source": "https://github.com/foo/bar"},
			wantOwner:   "foo",
			wantRepo:    "bar",
			wantOK:      true,
		},
		{
			name:      "github from homepage",
			pattern:   githubPattern,
			homepage:  "http://github.com/baz/qux",
			wantOwner: "baz",
			wantRepo:  "qux",
			wantOK:    true,
		},
		{
			name:        "gitlab from project urls",
			pattern:     gitlabPattern,
			projectURLs: map[string]string{"Repository": "https://gitlab.com/acme/widget"},
			wantOwner:   "acme",
			wantRepo:    "widget",
			wantOK:      true,
		},
		{
			name:        "no match",
			pattern:     githubPattern,
			projectURLs: map[string]string{"Homepage": "https://example.com"},
			wantOK:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, ok := ExtractRepoURL(tt.pattern, tt.projectURLs, tt.homepage)
			if ok != tt.wantOK {
				t.Errorf("got ok=%v, want %v", ok, tt.wantOK)
			}
			if ok {
				if owner != tt.wantOwner {
					t.Errorf("got owner=%s, want %s", owner, tt.wantOwner)
				}
				if repo != tt.wantRepo {
					t.Errorf("got repo=%s, want %s", repo, tt.wantRepo)
				}
			}
		})
	}
}
