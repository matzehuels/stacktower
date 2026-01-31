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

	"github.com/matzehuels/stacktower/pkg/core/deps"
	"github.com/matzehuels/stacktower/pkg/core/deps/languages"
	"github.com/matzehuels/stacktower/pkg/integrations/github"
)

// Default timeout for GitHub operations.
const defaultGitHubTimeout = 5 * time.Minute

// parseGitHubCommand creates the github subcommand for parsing from GitHub repos.
func (c *CLI) parseGitHubCommand(flags *parseFlags) *cobra.Command {
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
			return c.runParseGitHub(cmd.Context(), args, flags, publicOnly, timeout)
		},
	}

	cmd.Flags().BoolVar(&publicOnly, "public-only", false, "show only public repositories")
	cmd.Flags().DurationVar(&timeout, "timeout", defaultGitHubTimeout, "timeout for GitHub operations")

	return cmd
}

func (c *CLI) runParseGitHub(ctx context.Context, args []string, flags *parseFlags, publicOnly bool, timeout time.Duration) error {
	sess, err := loadGitHubSession(ctx)
	if err != nil {
		printWarning("Not logged in to GitHub. Starting login flow...")
		printNewline()
		sess, err = c.runGitHubLogin(ctx)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
	}

	c.Logger.Debug("Authenticated as", "user", sess.User.Login)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := github.NewContentClient(sess.AccessToken)

	var owner, repo string
	var manifests []github.ManifestFile
	var selectedManifest github.ManifestFile

	if len(args) == 1 {
		var err error
		owner, repo, err = github.ParseRepoRef(args[0])
		if err != nil {
			return err
		}
		printInfo("Repository: %s", StyleHighlight.Render(owner+"/"+repo))
	} else {
		spinner := newSpinner("Fetching and scanning repositories...")
		spinner.Start()
		manifestPatterns := deps.SupportedManifests(languages.All)
		rwm, err := client.ScanReposForManifests(ctx, manifestPatterns, publicOnly)
		spinner.Stop()
		if err != nil {
			return fmt.Errorf("scan repos: %w", err)
		}

		if len(rwm) == 0 {
			printError("No repositories found")
			return fmt.Errorf("no repositories found")
		}

		printSuccess("Found %d repositories with manifests", len(rwm))
		printNewline()

		m := NewRepoListModel(rwm)
		p := tea.NewProgram(m)
		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		fm, ok := finalModel.(RepoListModel)
		if !ok || fm.Selected == nil {
			printDetail("No selection made")
			return nil
		}

		parts := strings.SplitN(fm.Selected.Repo.Repo.FullName, "/", 2)
		owner, repo = parts[0], parts[1]
		manifests = fm.Selected.Repo.Manifests
	}

	if selectedManifest.Name == "" {
		if len(manifests) == 0 && len(args) == 1 {
			spinner := newSpinner(fmt.Sprintf("Scanning %s/%s for manifests...", owner, repo))
			spinner.Start()
			var err error
			manifests, err = client.DetectManifests(ctx, owner, repo, deps.SupportedManifests(languages.All))
			spinner.Stop()
			if err != nil {
				return fmt.Errorf("detect manifests: %w", err)
			}
		}

		if len(manifests) == 0 {
			printError("No manifest files found in %s/%s", owner, repo)
			return fmt.Errorf("no manifest files found in %s/%s", owner, repo)
		}

		if len(manifests) == 1 {
			selectedManifest = manifests[0]
			printInfo("Found: %s (%s)", StyleHighlight.Render(selectedManifest.Name), selectedManifest.Language)
		} else {
			printInfo("Found %d manifest files", len(manifests))
			printNewline()
			mm := NewManifestListModel(manifests)
			mp := tea.NewProgram(mm)
			mfinalModel, err := mp.Run()
			if err != nil {
				return err
			}

			mfm, ok := mfinalModel.(ManifestListModel)
			if !ok || mfm.Selected == nil {
				printDetail("No manifest selected")
				return nil
			}
			selectedManifest = *mfm.Selected
		}
	}

	spinner := newSpinner(fmt.Sprintf("Fetching %s...", selectedManifest.Path))
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

	// Set defaults from selection
	if flags.name == "" {
		flags.name = repo
	}
	if flags.output == "" {
		flags.output = repo + ".json"
	}

	printNewline()

	return c.parseManifest(ctx, lang, flags, tmpFile)
}
