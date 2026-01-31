package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/deps"
	"github.com/matzehuels/stacktower/pkg/core/deps/languages"
	"github.com/matzehuels/stacktower/pkg/graph"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// parseFlags holds parse command options.
type parseFlags struct {
	pipeline.Options
	output  string
	noCache bool
	name    string // project name override for manifest parsing
}

// parseCommand creates the parse command with language-specific subcommands.
func (c *CLI) parseCommand() *cobra.Command {
	flags := parseFlags{
		Options: pipeline.Options{
			MaxDepth: pipeline.DefaultMaxDepth,
			MaxNodes: pipeline.DefaultMaxNodes,
		},
	}

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

	cmd.PersistentFlags().IntVar(&flags.MaxDepth, "max-depth", flags.MaxDepth, "maximum dependency depth")
	cmd.PersistentFlags().IntVar(&flags.MaxNodes, "max-nodes", flags.MaxNodes, "maximum nodes to fetch")
	cmd.PersistentFlags().BoolVar(&flags.SkipEnrich, "skip-enrich", flags.SkipEnrich, "skip metadata enrichment")
	cmd.PersistentFlags().StringVarP(&flags.output, "output", "o", "", "output file (stdout if empty)")
	cmd.PersistentFlags().StringVarP(&flags.name, "name", "n", "", "project name (for manifest parsing)")
	cmd.PersistentFlags().BoolVar(&flags.noCache, "no-cache", false, "disable caching")

	for _, lang := range languages.All {
		cmd.AddCommand(c.langCommand(lang, &flags))
	}

	cmd.AddCommand(c.parseGitHubCommand(&flags))

	return cmd
}

// langCommand creates a language-specific parse subcommand.
func (c *CLI) langCommand(lang *deps.Language, flags *parseFlags) *cobra.Command {
	return &cobra.Command{
		Use:   fmt.Sprintf("%s <package-or-file>", lang.Name),
		Short: fmt.Sprintf("Parse %s dependencies", lang.Name),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runParse(cmd.Context(), lang, flags, args[0])
		},
	}
}

// runParse auto-detects whether arg is a manifest file or package name.
func (c *CLI) runParse(ctx context.Context, lang *deps.Language, flags *parseFlags, arg string) error {
	if lang.HasManifests() && looksLikeFile(arg) {
		return c.parseManifest(ctx, lang, flags, arg)
	}
	pkg := arg
	if lang.NormalizeName != nil {
		pkg = lang.NormalizeName(pkg)
	}
	return c.parsePackage(ctx, lang, flags, pkg)
}

// parsePackage parses a package using the pipeline service.
func (c *CLI) parsePackage(ctx context.Context, lang *deps.Language, flags *parseFlags, pkg string) error {
	if err := validatePackageName(pkg); err != nil {
		return err
	}

	runner, err := c.newRunner(flags.noCache)
	if err != nil {
		return fmt.Errorf("initialize runner: %w", err)
	}
	defer runner.Close()

	opts := flags.Options
	opts.Language = lang.Name
	opts.Package = pkg
	opts.Logger = c.Logger
	opts.GitHubToken = os.Getenv("GITHUB_TOKEN")

	spinner := newSpinnerWithContext(ctx, fmt.Sprintf("Resolving %s/%s...", lang.Name, pkg))
	spinner.Start()

	g, cacheHit, err := runner.ParseWithCacheInfo(ctx, opts)
	if err != nil {
		spinner.StopWithError("Failed to resolve dependencies")
		return fmt.Errorf("resolve %s/%s: %w", lang.Name, pkg, err)
	}
	spinner.Stop()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	return finishParse(g, flags.output, lang.Name, pkg, cacheHit)
}

// parseManifest parses a manifest file using the pipeline service.
func (c *CLI) parseManifest(ctx context.Context, lang *deps.Language, flags *parseFlags, filePath string) error {
	manifestContent, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	runner, err := c.newRunner(flags.noCache)
	if err != nil {
		return fmt.Errorf("initialize runner: %w", err)
	}
	defer runner.Close()

	opts := flags.Options
	opts.Language = lang.Name
	opts.Manifest = string(manifestContent)
	opts.ManifestFilename = filepath.Base(filePath)
	opts.Logger = c.Logger
	opts.GitHubToken = os.Getenv("GITHUB_TOKEN")

	spinner := newSpinnerWithContext(ctx, fmt.Sprintf("Parsing %s...", filepath.Base(filePath)))
	spinner.Start()

	g, cacheHit, err := runner.ParseWithCacheInfo(ctx, opts)
	if err != nil {
		spinner.StopWithError("Failed to parse manifest")
		return fmt.Errorf("parse %s: %w", filepath.Base(filePath), err)
	}
	spinner.Stop()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Rename root node
	name := flags.name
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	}
	if name != "" {
		_ = g.RenameNode(graph.ProjectRootNodeID, name)
	}

	return finishParse(g, flags.output, lang.Name, filepath.Base(filePath), cacheHit)
}

// finishParse writes output and prints summary.
func finishParse(g *dag.DAG, output, langName, source string, cacheHit bool) error {
	if output == "" {
		if err := graph.WriteGraph(g, os.Stdout); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	} else {
		if err := graph.WriteGraphFile(g, output); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	}

	printSuccess("Dependencies resolved")
	printKeyValue("Source", source)
	printKeyValue("Language", langName)
	if output != "" {
		printFile(output)
	}
	printStats(g.NodeCount(), g.EdgeCount(), cacheHit)

	if output != "" {
		printNewline()
		printNextStep("Render", "stacktower render "+output)
	}
	return nil
}

// looksLikeFile returns true if arg appears to be a file path.
func looksLikeFile(arg string) bool {
	if _, err := os.Stat(arg); err == nil {
		return true
	}
	base := filepath.Base(arg)
	return deps.IsManifestSupported(base, languages.All)
}

// validatePackageName performs basic security validation on package names.
func validatePackageName(name string) error {
	if name == "" {
		return fmt.Errorf("package name cannot be empty")
	}
	if len(name) > 256 {
		return fmt.Errorf("package name too long (max 256 characters)")
	}

	dangerousPatterns := []string{"..", "//", "\x00", "\\"}
	for _, pattern := range dangerousPatterns {
		if strings.Contains(name, pattern) {
			return fmt.Errorf("invalid package name: contains %q", pattern)
		}
	}

	for _, r := range name {
		if r < 32 || r == 127 {
			return fmt.Errorf("invalid package name: contains control characters")
		}
	}

	return nil
}
