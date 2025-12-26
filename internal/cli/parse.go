package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/deps"
	"github.com/matzehuels/stacktower/pkg/core/deps/languages"
	"github.com/matzehuels/stacktower/pkg/infra"
	"github.com/matzehuels/stacktower/pkg/infra/common"
	"github.com/matzehuels/stacktower/pkg/integrations/github"
	pkgio "github.com/matzehuels/stacktower/pkg/io"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// parseOpts holds the command-line flags for the parse command.
type parseOpts struct {
	maxDepth  int    // maximum dependency tree depth
	maxNodes  int    // maximum total nodes to fetch
	enrich    bool   // whether to fetch GitHub metadata
	output    string // output file path (stdout if empty)
	name      string // override project name for manifest parsing
	normalize bool   // apply DAG normalization
	noCache   bool   // disable caching
}

// newParseCmd creates the parse command with language-specific subcommands.
func newParseCmd() *cobra.Command {
	opts := parseOpts{maxDepth: 10, maxNodes: 5000, enrich: true, normalize: true}

	cmd := &cobra.Command{
		Use:   "parse",
		Short: "Parse dependency graphs from package managers or manifest files",
		Long: `Parse dependency graphs from package managers or local manifest files.

The command auto-detects whether you're providing a package name or a manifest file.
Results are cached locally for faster subsequent runs.

Examples:
  stacktower parse python requests                        # Package from PyPI
  stacktower parse python poetry.lock                     # Manifest file
  stacktower parse python requests --no-cache             # Disable caching`,
	}

	cmd.PersistentFlags().IntVar(&opts.maxDepth, "max-depth", opts.maxDepth, "maximum dependency depth")
	cmd.PersistentFlags().IntVar(&opts.maxNodes, "max-nodes", opts.maxNodes, "maximum nodes to fetch")
	cmd.PersistentFlags().BoolVar(&opts.enrich, "enrich", opts.enrich, "enrich with GitHub metadata (requires GITHUB_TOKEN)")
	cmd.PersistentFlags().BoolVar(&opts.normalize, "normalize", opts.normalize, "apply DAG normalization")
	cmd.PersistentFlags().StringVarP(&opts.output, "output", "o", "", "output file (stdout if empty)")
	cmd.PersistentFlags().StringVarP(&opts.name, "name", "n", "", "project name (for manifest parsing)")
	cmd.PersistentFlags().BoolVar(&opts.noCache, "no-cache", false, "disable caching")

	for _, lang := range languages.All {
		cmd.AddCommand(newLangCmd(lang, &opts))
	}

	cmd.AddCommand(newParseGitHubCmd(&opts))

	return cmd
}

// newLangCmd creates a language-specific parse subcommand (e.g., "parse python").
func newLangCmd(lang *deps.Language, opts *parseOpts) *cobra.Command {
	return &cobra.Command{
		Use:   fmt.Sprintf("%s <package-or-file>", lang.Name),
		Short: fmt.Sprintf("Parse %s dependencies", lang.Name),
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return parseAuto(c.Context(), lang, opts, args[0])
		},
	}
}

// parseAuto auto-detects whether arg is a manifest file or package name.
func parseAuto(ctx context.Context, lang *deps.Language, opts *parseOpts, arg string) error {
	if lang.HasManifests() && looksLikeFile(arg) {
		return parseManifest(ctx, lang, opts, arg)
	}
	pkg := arg
	if lang.NormalizeName != nil {
		pkg = lang.NormalizeName(pkg)
	}
	return parsePackage(ctx, lang, opts, pkg)
}

// parsePackage parses a package using the pipeline service.
func parsePackage(ctx context.Context, lang *deps.Language, opts *parseOpts, pkg string) error {
	logger := common.LoggerFromContext(ctx)

	// Create pipeline service
	svc, cleanup, err := newPipelineService(opts.noCache, logger)
	if err != nil {
		return err
	}
	defer cleanup()

	// Build pipeline options
	pipelineOpts := pipeline.Options{
		Language:    lang.Name,
		Package:     pkg,
		MaxDepth:    opts.maxDepth,
		MaxNodes:    opts.maxNodes,
		Normalize:   opts.normalize,
		Enrich:      opts.enrich,
		Logger:      logger,
		GitHubToken: infra.LoadGitHubConfig().Token,
	}

	logger.Infof("Resolving %s/%s", lang.Name, pkg)
	prog := common.NewProgress(logger)

	// Parse via pipeline
	g, data, cacheHit, err := svc.Parse(ctx, pipelineOpts)
	if err != nil {
		return err
	}

	prog.Done(fmt.Sprintf("Resolved %d packages with %d dependencies (%s)",
		g.NodeCount(), g.EdgeCount(), formatCacheStatus(cacheHit)))

	return writeData(data, opts.output, logger)
}

// parseManifest parses a manifest file using the pipeline service.
func parseManifest(ctx context.Context, lang *deps.Language, opts *parseOpts, filePath string) error {
	logger := common.LoggerFromContext(ctx)

	// Read manifest content
	manifestContent, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	// Create pipeline service
	svc, cleanup, err := newPipelineService(opts.noCache, logger)
	if err != nil {
		return err
	}
	defer cleanup()

	// Build pipeline options
	pipelineOpts := pipeline.Options{
		Language:         lang.Name,
		Manifest:         string(manifestContent),
		ManifestFilename: filepath.Base(filePath),
		MaxDepth:         opts.maxDepth,
		MaxNodes:         opts.maxNodes,
		Normalize:        opts.normalize,
		Enrich:           opts.enrich,
		Logger:           logger,
		GitHubToken:      infra.LoadGitHubConfig().Token,
	}

	logger.Infof("Parsing %s", filepath.Base(filePath))
	prog := common.NewProgress(logger)

	// Parse via pipeline
	g, data, cacheHit, err := svc.Parse(ctx, pipelineOpts)
	if err != nil {
		return err
	}

	// Rename root node if name is specified
	name := opts.name
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	}
	if name != "" {
		_ = g.RenameNode("__project__", name)
		// Re-serialize since we renamed
		data, err = serializeGraph(g)
		if err != nil {
			return err
		}
	}

	prog.Done(fmt.Sprintf("Parsed %d packages with %d dependencies (%s)",
		g.NodeCount(), g.EdgeCount(), formatCacheStatus(cacheHit)))

	return writeData(data, opts.output, logger)
}

// looksLikeFile returns true if arg appears to be a file path.
func looksLikeFile(arg string) bool {
	if _, err := os.Stat(arg); err == nil {
		return true
	}
	lower := strings.ToLower(arg)
	return strings.HasSuffix(lower, ".txt") ||
		strings.HasSuffix(lower, ".lock") ||
		strings.HasSuffix(lower, ".toml") ||
		strings.HasSuffix(lower, ".xml") ||
		lower == "go.mod" ||
		lower == "pom.xml"
}

// =============================================================================
// Output helpers
// =============================================================================

// writeData writes raw data to the specified path (or stdout if empty).
func writeData(data []byte, path string, logger interface{ Infof(string, ...any) }) error {
	out, err := openOutput(path)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := out.Write(data); err != nil {
		return err
	}
	if path != "" {
		logger.Infof("Wrote to %s", path)
	}
	return nil
}

// serializeGraph converts a DAG to JSON bytes.
func serializeGraph(g *dag.DAG) ([]byte, error) {
	var buf strings.Builder
	if err := pkgio.WriteJSON(g, &buf); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

// nopCloser wraps an io.Writer with a no-op Close method.
type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }

// openOutput returns a WriteCloser for the given path.
func openOutput(path string) (io.WriteCloser, error) {
	if path == "" {
		return nopCloser{os.Stdout}, nil
	}
	return os.Create(path)
}

// =============================================================================
// GitHub integration
// =============================================================================

// newParseGitHubCmd creates the github subcommand for parsing from GitHub repos.
func newParseGitHubCmd(opts *parseOpts) *cobra.Command {
	var publicOnly bool

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
			return runParseGitHub(cmd.Context(), args, opts, publicOnly)
		},
	}

	cmd.Flags().BoolVar(&publicOnly, "public-only", false, "show only public repositories")

	return cmd
}

func runParseGitHub(ctx context.Context, args []string, opts *parseOpts, publicOnly bool) error {
	logger := common.LoggerFromContext(ctx)

	sess, err := LoadGitHubSession()
	if err != nil {
		fmt.Println("Not logged in to GitHub. Starting login flow...")
		sess, err = runGitHubLogin(ctx)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
	}

	logger.Debug("Authenticated as", "user", sess.User.Login)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
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
		fmt.Printf("Repository: %s/%s\n", owner, repo)
	} else {
		fmt.Println("Fetching and scanning repositories...")
		manifestPatterns := deps.SupportedManifests(languages.All)
		rwm, err := client.ScanReposForManifests(ctx, manifestPatterns, publicOnly)
		if err != nil {
			return fmt.Errorf("scan repos: %w", err)
		}

		if len(rwm) == 0 {
			return fmt.Errorf("no repositories found")
		}

		m := NewRepoListModel(rwm, true)
		p := tea.NewProgram(m)
		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		fm, ok := finalModel.(RepoListModel)
		if !ok || fm.Selected == nil {
			fmt.Println("No selection made.")
			return nil
		}

		parts := strings.SplitN(fm.Selected.Repo.Repo.FullName, "/", 2)
		owner, repo = parts[0], parts[1]
		manifests = fm.Selected.Repo.Manifests
	}

	if selectedManifest.Name == "" {
		if len(manifests) == 0 && len(args) == 1 {
			fmt.Printf("Scanning %s/%s for manifests...\n", owner, repo)
			var err error
			manifests, err = client.DetectManifests(ctx, owner, repo, deps.SupportedManifests(languages.All))
			if err != nil {
				return fmt.Errorf("detect manifests: %w", err)
			}
		}

		if len(manifests) == 0 {
			return fmt.Errorf("no manifest files found in %s/%s", owner, repo)
		}

		if len(manifests) == 1 {
			selectedManifest = manifests[0]
			fmt.Printf("Found: %s (%s)\n", selectedManifest.Name, selectedManifest.Language)
		} else {
			fmt.Printf("Found %d manifest files:\n", len(manifests))
			mm := NewManifestListModel(manifests)
			mp := tea.NewProgram(mm)
			mfinalModel, err := mp.Run()
			if err != nil {
				return err
			}

			mfm, ok := mfinalModel.(ManifestListModel)
			if !ok || mfm.Selected == nil {
				fmt.Println("No manifest selected.")
				return nil
			}
			selectedManifest = *mfm.Selected
		}
	}

	fmt.Printf("Fetching %s...\n", selectedManifest.Path)
	content, err := client.FetchFileRaw(ctx, owner, repo, selectedManifest.Path)
	if err != nil {
		return fmt.Errorf("fetch manifest: %w", err)
	}

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

	if opts.name == "" {
		opts.name = repo
	}

	if opts.output == "" {
		opts.output = repo + ".json"
	}

	fmt.Printf("Parsing %s dependencies...\n", selectedManifest.Language)

	if err := parseManifest(ctx, lang, opts, tmpFile); err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	fmt.Printf("\nDone! Graph saved to: %s\n", opts.output)
	fmt.Printf("Next: stacktower render %s\n", opts.output)

	return nil
}
