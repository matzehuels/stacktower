package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/internal/cli/term"
	"github.com/matzehuels/stacktower/pkg/core/deps"
	"github.com/matzehuels/stacktower/pkg/core/deps/languages"
	"github.com/matzehuels/stacktower/pkg/infra"
	"github.com/matzehuels/stacktower/pkg/integrations/github"
)

// Default timeout for GitHub operations.
const defaultGitHubTimeout = 5 * time.Minute

// newParseGitHubCmd creates the github subcommand for parsing from GitHub repos.
func newParseGitHubCmd(opts *ParseCmdOpts) *cobra.Command {
	var (
		publicOnly bool
		timeout    time.Duration
	)

	cmd := &cobra.Command{
		Use:   "github [owner/repo]",
		Short: "Parse dependencies from a GitHub repository",
		Long: `Interactive workflow to parse dependencies from a GitHub repository.

If not logged in, prompts you to authenticate with GitHub first.
If no repository is specified, shows an interactive list to select one.
Then lets you select a manifest file from the repository.

Examples:
  stacktower parse github                       # Interactive selection
  stacktower parse github owner/repo            # Select manifest from repo
  stacktower parse github owner/repo -o out.json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runParseGitHub(cmd.Context(), args, opts, publicOnly, timeout)
		},
	}

	cmd.Flags().BoolVar(&publicOnly, "public-only", false, "show only public repositories")
	cmd.Flags().DurationVar(&timeout, "timeout", defaultGitHubTimeout, "timeout for GitHub operations")

	return cmd
}

func runParseGitHub(ctx context.Context, args []string, opts *ParseCmdOpts, publicOnly bool, timeout time.Duration) error {
	logger := infra.LoggerFromContext(ctx)

	sess, err := LoadGitHubSession(ctx)
	if err != nil {
		term.PrintWarning("Not logged in to GitHub. Starting login flow...")
		term.PrintNewline()
		sess, err = runGitHubLogin(ctx)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
	}

	logger.Debug("Authenticated as", "user", sess.User.Login)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := github.NewContentClient(sess.AccessToken)

	var owner, repo string
	var manifests []github.ManifestFile
	var selectedManifest github.ManifestFile

	if len(args) == 1 {
		parts := strings.SplitN(args[0], "/", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid repo format, use owner/repo")
		}
		owner, repo = parts[0], parts[1]
		term.PrintInfo("Repository: %s", term.StyleHighlight.Render(owner+"/"+repo))
	} else {
		spinner := term.NewSpinner("Fetching and scanning repositories...")
		spinner.Start()
		manifestPatterns := deps.SupportedManifests(languages.All)
		rwm, err := client.ScanReposForManifests(ctx, manifestPatterns, publicOnly)
		spinner.Stop()
		if err != nil {
			return fmt.Errorf("scan repos: %w", err)
		}

		if len(rwm) == 0 {
			term.PrintError("No repositories found")
			return fmt.Errorf("no repositories found")
		}

		term.PrintSuccess("Found %d repositories with manifests", len(rwm))
		term.PrintNewline()

		m := term.NewRepoListModel(rwm, true)
		p := tea.NewProgram(m)
		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		fm, ok := finalModel.(term.RepoListModel)
		if !ok || fm.Selected == nil {
			term.PrintDetail("No selection made")
			return nil
		}

		parts := strings.SplitN(fm.Selected.Repo.Repo.FullName, "/", 2)
		owner, repo = parts[0], parts[1]
		manifests = fm.Selected.Repo.Manifests
	}

	if selectedManifest.Name == "" {
		if len(manifests) == 0 && len(args) == 1 {
			spinner := term.NewSpinner(fmt.Sprintf("Scanning %s/%s for manifests...", owner, repo))
			spinner.Start()
			var err error
			manifests, err = client.DetectManifests(ctx, owner, repo, deps.SupportedManifests(languages.All))
			spinner.Stop()
			if err != nil {
				return fmt.Errorf("detect manifests: %w", err)
			}
		}

		if len(manifests) == 0 {
			term.PrintError("No manifest files found in %s/%s", owner, repo)
			return fmt.Errorf("no manifest files found in %s/%s", owner, repo)
		}

		if len(manifests) == 1 {
			selectedManifest = manifests[0]
			term.PrintInfo("Found: %s (%s)", term.StyleHighlight.Render(selectedManifest.Name), selectedManifest.Language)
		} else {
			term.PrintInfo("Found %d manifest files", len(manifests))
			term.PrintNewline()
			mm := term.NewManifestListModel(manifests)
			mp := tea.NewProgram(mm)
			mfinalModel, err := mp.Run()
			if err != nil {
				return err
			}

			mfm, ok := mfinalModel.(term.ManifestListModel)
			if !ok || mfm.Selected == nil {
				term.PrintDetail("No manifest selected")
				return nil
			}
			selectedManifest = *mfm.Selected
		}
	}

	spinner := term.NewSpinner(fmt.Sprintf("Fetching %s...", selectedManifest.Path))
	spinner.Start()
	content, err := client.FetchFileRaw(ctx, owner, repo, selectedManifest.Path)
	if err != nil {
		spinner.StopWithError("Failed to fetch manifest")
		return fmt.Errorf("fetch manifest: %w", err)
	}
	spinner.Stop()

	tmpDir, err := os.MkdirTemp("", "stacktower-github-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, selectedManifest.Name)
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	lang := languages.Find(selectedManifest.Language)
	if lang == nil {
		return fmt.Errorf("unsupported language: %s", selectedManifest.Language)
	}

	if opts.Name == "" {
		opts.Name = repo
	}

	if opts.Output == "" {
		opts.Output = repo + ".json"
	}

	term.PrintNewline()

	return runParseManifest(ctx, lang, opts, tmpFile)
}
