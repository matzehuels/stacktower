package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/internal/cli/term"
	"github.com/matzehuels/stacktower/pkg/config"
	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/deps"
	"github.com/matzehuels/stacktower/pkg/core/deps/languages"
	"github.com/matzehuels/stacktower/pkg/dto"
)

// NewParseCmd creates the parse command with language-specific subcommands.
func NewParseCmd() *cobra.Command {
	opts := DefaultParseCmdOpts()

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

	cmd.PersistentFlags().IntVar(&opts.MaxDepth, "max-depth", opts.MaxDepth, "maximum dependency depth")
	cmd.PersistentFlags().IntVar(&opts.MaxNodes, "max-nodes", opts.MaxNodes, "maximum nodes to fetch")
	cmd.PersistentFlags().BoolVar(&opts.SkipEnrich, "skip-enrich", opts.SkipEnrich, "skip metadata enrichment (GitHub descriptions, etc.)")
	cmd.PersistentFlags().StringVarP(&opts.Output, "output", "o", "", "output file (stdout if empty)")
	cmd.PersistentFlags().StringVarP(&opts.Name, "name", "n", "", "project name (for manifest parsing)")
	cmd.PersistentFlags().BoolVar(&opts.NoCache, "no-cache", false, "disable caching")

	for _, lang := range languages.All {
		cmd.AddCommand(newLangCmd(lang, &opts))
	}

	cmd.AddCommand(newParseGitHubCmd(&opts))

	return cmd
}

// newLangCmd creates a language-specific parse subcommand (e.g., "parse python").
func newLangCmd(lang *deps.Language, opts *ParseCmdOpts) *cobra.Command {
	return &cobra.Command{
		Use:   fmt.Sprintf("%s <package-or-file>", lang.Name),
		Short: fmt.Sprintf("Parse %s dependencies", lang.Name),
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runParse(c.Context(), lang, opts, args[0])
		},
	}
}

// runParse auto-detects whether arg is a manifest file or package name.
func runParse(ctx context.Context, lang *deps.Language, opts *ParseCmdOpts, arg string) error {
	if lang.HasManifests() && looksLikeFile(arg) {
		return runParseManifest(ctx, lang, opts, arg)
	}
	pkg := arg
	if lang.NormalizeName != nil {
		pkg = lang.NormalizeName(pkg)
	}
	return runParsePackage(ctx, lang, opts, pkg)
}

// runParsePackage parses a package using the pipeline service.
func runParsePackage(ctx context.Context, lang *deps.Language, opts *ParseCmdOpts, pkg string) error {
	logger := loggerFromContext(ctx)

	runner, err := NewRunner(opts.NoCache, logger)
	if err != nil {
		return fmt.Errorf("initialize runner: %w", err)
	}
	defer runner.Close()

	opts.Language = lang.Name
	opts.Package = pkg
	opts.Logger = logger
	opts.GitHubToken = config.LoadGitHubConfig().Token

	spinner := term.NewSpinner(fmt.Sprintf("Resolving %s/%s...", lang.Name, pkg))
	spinner.Start()

	g, cacheHit, err := runner.ParseWithCacheInfo(ctx, opts.Options)
	if err != nil {
		spinner.StopWithError("Failed to resolve dependencies")
		return fmt.Errorf("resolve %s/%s: %w", lang.Name, pkg, err)
	}
	spinner.Stop()

	return finishParse(g, opts, lang.Name, pkg, cacheHit)
}

// runParseManifest parses a manifest file using the pipeline service.
func runParseManifest(ctx context.Context, lang *deps.Language, opts *ParseCmdOpts, filePath string) error {
	logger := loggerFromContext(ctx)

	manifestContent, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	runner, err := NewRunner(opts.NoCache, logger)
	if err != nil {
		return fmt.Errorf("initialize runner: %w", err)
	}
	defer runner.Close()

	opts.Language = lang.Name
	opts.Manifest = string(manifestContent)
	opts.ManifestFilename = filepath.Base(filePath)
	opts.Logger = logger
	opts.GitHubToken = config.LoadGitHubConfig().Token

	spinner := term.NewSpinner(fmt.Sprintf("Parsing %s...", filepath.Base(filePath)))
	spinner.Start()

	g, cacheHit, err := runner.ParseWithCacheInfo(ctx, opts.Options)
	if err != nil {
		spinner.StopWithError("Failed to parse manifest")
		return fmt.Errorf("parse %s: %w", filepath.Base(filePath), err)
	}
	spinner.Stop()

	// Rename root node if name is specified
	name := opts.Name
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	}
	if name != "" {
		_ = g.RenameNode("__project__", name)
	}

	return finishParse(g, opts, lang.Name, filepath.Base(filePath), cacheHit)
}

// finishParse writes output and prints summary for a parsed graph.
func finishParse(g *dag.DAG, opts *ParseCmdOpts, langName, source string, cacheHit bool) error {
	// Write output
	if opts.Output == "" {
		if err := dto.WriteGraph(g, os.Stdout); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	} else {
		if err := dto.WriteGraphFile(g, opts.Output); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	}

	// Show success summary
	summary := term.NewSuccessSummary("Dependencies resolved")
	summary.AddKeyValue("Source", source)
	summary.AddKeyValue("Language", langName)
	if opts.Output != "" {
		summary.AddFile(opts.Output)
	}
	summary.Print()
	term.PrintStats(g.NodeCount(), g.EdgeCount(), cacheHit)

	if opts.Output != "" {
		term.PrintNewline()
		term.PrintNextStep("Render", "stacktower render "+opts.Output)
	}
	return nil
}

// looksLikeFile returns true if arg appears to be a file path.
// It checks if the file exists or if the filename matches a known manifest pattern.
func looksLikeFile(arg string) bool {
	// Check if file actually exists
	if _, err := os.Stat(arg); err == nil {
		return true
	}
	// Check against supported manifest patterns from language definitions
	base := filepath.Base(arg)
	return deps.IsManifestSupported(base, languages.All)
}
