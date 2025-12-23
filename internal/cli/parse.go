package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/deps"
	"github.com/matzehuels/stacktower/pkg/deps/golang"
	"github.com/matzehuels/stacktower/pkg/deps/java"
	"github.com/matzehuels/stacktower/pkg/deps/javascript"
	"github.com/matzehuels/stacktower/pkg/deps/metadata"
	"github.com/matzehuels/stacktower/pkg/deps/php"
	"github.com/matzehuels/stacktower/pkg/deps/python"
	"github.com/matzehuels/stacktower/pkg/deps/ruby"
	"github.com/matzehuels/stacktower/pkg/deps/rust"
	pkgio "github.com/matzehuels/stacktower/pkg/io"
)

// languages is the list of supported package ecosystems.
// Each language provides resolvers for package registries and manifest parsers.
var languages = []*deps.Language{
	python.Language,
	rust.Language,
	javascript.Language,
	ruby.Language,
	php.Language,
	java.Language,
	golang.Language,
}

// parseOpts holds the command-line flags for the parse command.
// These options control dependency resolution depth, caching, and metadata enrichment.
type parseOpts struct {
	maxDepth int    // maximum dependency tree depth
	maxNodes int    // maximum total nodes to fetch
	enrich   bool   // whether to fetch GitHub metadata
	refresh  bool   // bypass HTTP cache
	output   string // output file path (stdout if empty)
	name     string // override project name for manifest parsing
}

// resolveOptions converts parseOpts into deps.Options for the resolver.
// If metadata enrichment fails (e.g., missing GITHUB_TOKEN), a warning is logged
// and enrichment is disabled rather than failing the entire operation.
func (o *parseOpts) resolveOptions(ctx context.Context) deps.Options {
	logger := loggerFromContext(ctx)
	providers, err := metadataProviders(o.enrich)
	if err != nil {
		logger.Warnf("Metadata enrichment disabled: %v", err)
	}
	return deps.Options{
		MaxDepth:          o.maxDepth,
		MaxNodes:          o.maxNodes,
		Refresh:           o.refresh,
		CacheTTL:          deps.DefaultCacheTTL,
		MetadataProviders: providers,
		Logger:            func(msg string, args ...any) { logger.Warnf(msg, args...) },
	}
}

// newParseCmd creates the parse command with language-specific subcommands.
// It supports parsing from package registries (e.g., "parse python requests")
// or from local manifest files (e.g., "parse python poetry.lock").
//
// Default options:
//   - maxDepth: 10 levels of transitive dependencies
//   - maxNodes: 5000 packages maximum
//   - enrich: true (fetch GitHub metadata if GITHUB_TOKEN is set)
func newParseCmd() *cobra.Command {
	opts := parseOpts{maxDepth: 10, maxNodes: 5000, enrich: true}

	cmd := &cobra.Command{
		Use:   "parse",
		Short: "Parse dependency graphs from package managers or manifest files",
		Long: `Parse dependency graphs from package managers or local manifest files.

The command auto-detects whether you're providing a package name or a manifest file.

Examples:
  stacktower parse python requests                        # Package from PyPI
  stacktower parse python poetry.lock                     # Manifest file
  stacktower parse python registry pypi fastapi           # Explicit registry  
  stacktower parse python manifest poetry my_poetry.lock  # Explicit manifest type`,
	}

	cmd.PersistentFlags().IntVar(&opts.maxDepth, "max-depth", opts.maxDepth, "maximum dependency depth")
	cmd.PersistentFlags().IntVar(&opts.maxNodes, "max-nodes", opts.maxNodes, "maximum nodes to fetch")
	cmd.PersistentFlags().BoolVar(&opts.enrich, "enrich", opts.enrich, "enrich with GitHub metadata (requires GITHUB_TOKEN)")
	cmd.PersistentFlags().BoolVar(&opts.refresh, "refresh", false, "bypass cache")
	cmd.PersistentFlags().StringVarP(&opts.output, "output", "o", "", "output file (stdout if empty)")
	cmd.PersistentFlags().StringVarP(&opts.name, "name", "n", "", "project name (for manifest parsing, overrides auto-detection)")

	for _, lang := range languages {
		cmd.AddCommand(langCmd(lang, &opts))
	}

	return cmd
}

// langCmd creates a language-specific parse subcommand (e.g., "parse python").
// The command auto-detects whether the argument is a package name or manifest file
// using smartParse.
func langCmd(lang *deps.Language, opts *parseOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%s <package-or-file>", lang.Name),
		Short: fmt.Sprintf("Parse %s dependencies", lang.Name),
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return smartParse(c.Context(), lang, opts, args[0])
		},
	}

	cmd.AddCommand(registryCmd(lang, opts))
	if lang.HasManifests() {
		cmd.AddCommand(manifestCmd(lang, opts))
	}

	return cmd
}

// registryCmd creates the "registry" subcommand for explicit registry selection.
// Example: "parse python registry pypi requests" to force PyPI even if the default changes.
func registryCmd(lang *deps.Language, opts *parseOpts) *cobra.Command {
	return &cobra.Command{
		Use:   fmt.Sprintf("registry <%s> <package>", lang.DefaultRegistry),
		Short: fmt.Sprintf("Parse %s package from specific registry", lang.Name),
		Args:  cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			res, err := lang.Registry(args[0])
			if err != nil {
				return err
			}
			pkg := args[1]
			// Normalize package name if the language supports it
			if lang.NormalizeName != nil {
				pkg = lang.NormalizeName(pkg)
			}
			return resolve(c.Context(), opts, res, pkg)
		},
	}
}

// manifestCmd creates the "manifest" subcommand for explicit manifest type selection.
// Example: "parse python manifest poetry poetry.lock" to force Poetry format.
func manifestCmd(lang *deps.Language, opts *parseOpts) *cobra.Command {
	return &cobra.Command{
		Use:   fmt.Sprintf("manifest <%s> <file>", strings.Join(lang.ManifestTypes, "|")),
		Short: fmt.Sprintf("Parse %s manifest file with explicit type", lang.Name),
		Args:  cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			res, err := lang.Resolver()
			if err != nil {
				return err
			}
			parser, ok := lang.Manifest(args[0], res)
			if !ok {
				return fmt.Errorf("unknown manifest type: %s (available: %s)", args[0], strings.Join(lang.ManifestTypes, ", "))
			}
			return parseManifest(c.Context(), opts, parser, args[1])
		},
	}
}

// smartParse auto-detects whether arg is a manifest file or package name.
// If looksLikeFile returns true and the language supports manifests, it parses as a file.
// Otherwise, it resolves as a package from the default registry.
func smartParse(ctx context.Context, lang *deps.Language, opts *parseOpts, arg string) error {
	if lang.HasManifests() && looksLikeFile(arg) {
		return parseManifestAuto(ctx, lang, opts, arg)
	}
	res, err := lang.Resolver()
	if err != nil {
		return err
	}
	// Normalize package name if the language supports it (e.g., Maven coordinate normalization)
	if lang.NormalizeName != nil {
		arg = lang.NormalizeName(arg)
	}
	return resolve(ctx, opts, res, arg)
}

// parseManifestAuto attempts to detect the manifest file type and parse it.
// It returns an error if detection fails or if the file format is unsupported.
func parseManifestAuto(ctx context.Context, lang *deps.Language, opts *parseOpts, filePath string) error {
	res, err := lang.Resolver()
	if err != nil {
		return fmt.Errorf("resolver: %w", err)
	}
	parser, err := deps.DetectManifest(filePath, lang.ManifestParsers(res)...)
	if err != nil {
		return fmt.Errorf("%w\n\nSupported: %s", err, strings.Join(lang.ManifestTypes, ", "))
	}
	return parseManifest(ctx, opts, parser, filePath)
}

// resolve fetches the dependency graph for pkg from the given resolver.
// It logs progress and writes the resulting graph as JSON to opts.output (or stdout).
func resolve(ctx context.Context, opts *parseOpts, res deps.Resolver, pkg string) error {
	logger := loggerFromContext(ctx)
	logger.Infof("Resolving %s from %s", pkg, res.Name())

	prog := newProgress(logger)
	g, err := res.Resolve(ctx, pkg, opts.resolveOptions(ctx))
	if err != nil {
		return err
	}
	prog.done(fmt.Sprintf("Resolved %d packages with %d dependencies", g.NodeCount(), g.EdgeCount()))

	return writeGraph(g, opts.output, logger)
}

// parseManifest parses a manifest file and writes the resulting dependency graph.
// If opts.name is set, it renames the root node from "__project__" to the specified name.
func parseManifest(ctx context.Context, opts *parseOpts, parser deps.ManifestParser, filePath string) error {
	logger := loggerFromContext(ctx)
	logger.Infof("Parsing %s (%s)", filePath, parser.Type())

	prog := newProgress(logger)
	result, err := parser.Parse(filePath, opts.resolveOptions(ctx))
	if err != nil {
		return err
	}
	g := result.Graph.(*dag.DAG)

	name := opts.name
	if name == "" {
		name = result.RootPackage
	}
	if name != "" {
		_ = g.RenameNode("__project__", name)
	}

	prog.done(fmt.Sprintf("Parsed %d packages with %d dependencies", g.NodeCount(), g.EdgeCount()))

	return writeGraph(g, opts.output, logger)
}

// looksLikeFile returns true if arg appears to be a file path rather than a package name.
// It checks if the file exists or has a known manifest extension (.txt, .lock, .toml, .xml).
// Known manifest files like "go.mod" and "pom.xml" always return true.
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

// writeGraph serializes g as JSON to the specified path (or stdout if empty).
// The logger is notified on success with the output path.
func writeGraph(g *dag.DAG, path string, logger interface{ Infof(string, ...any) }) error {
	out, err := openOutput(path)
	if err != nil {
		return err
	}
	defer out.Close()

	if err := pkgio.WriteJSON(g, out); err != nil {
		return err
	}
	if path != "" {
		logger.Infof("Wrote graph to %s", path)
	}
	return nil
}

// metadataProviders returns GitHub metadata providers if enrich is true.
// It requires the GITHUB_TOKEN environment variable.
// If the token is missing or invalid, an error is returned.
func metadataProviders(enrich bool) ([]deps.MetadataProvider, error) {
	if !enrich {
		return nil, nil
	}
	tok := os.Getenv("GITHUB_TOKEN")
	if tok == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN not set (set it or use --enrich=false)")
	}
	gh, err := metadata.NewGitHub(tok, deps.DefaultCacheTTL)
	if err != nil {
		return nil, fmt.Errorf("github: %w", err)
	}
	return []deps.MetadataProvider{gh}, nil
}

// nopCloser wraps an io.Writer with a no-op Close method.
// It is used to make os.Stdout compatible with io.WriteCloser.
type nopCloser struct{ io.Writer }

// Close implements io.Closer with a no-op.
func (nopCloser) Close() error { return nil }

// openOutput returns a WriteCloser for the given path.
// If path is empty, it returns os.Stdout wrapped in nopCloser.
// Otherwise, it creates the file at path, overwriting if it exists.
func openOutput(path string) (io.WriteCloser, error) {
	if path == "" {
		return nopCloser{os.Stdout}, nil
	}
	return os.Create(path)
}
