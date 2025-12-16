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

var languages = []*deps.Language{
	python.Language,
	rust.Language,
	javascript.Language,
	ruby.Language,
	php.Language,
	java.Language,
	golang.Language,
}

type parseOpts struct {
	maxDepth int
	maxNodes int
	enrich   bool
	refresh  bool
	output   string
}

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

	for _, lang := range languages {
		cmd.AddCommand(langCmd(lang, &opts))
	}

	return cmd
}

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
			return resolve(c.Context(), opts, res, args[1])
		},
	}
}

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

func smartParse(ctx context.Context, lang *deps.Language, opts *parseOpts, arg string) error {
	if lang.HasManifests() && looksLikeFile(arg) {
		return parseManifestAuto(ctx, lang, opts, arg)
	}
	res, err := lang.Resolver()
	if err != nil {
		return err
	}
	return resolve(ctx, opts, res, arg)
}

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

func parseManifest(ctx context.Context, opts *parseOpts, parser deps.ManifestParser, filePath string) error {
	logger := loggerFromContext(ctx)
	logger.Infof("Parsing %s (%s)", filePath, parser.Type())

	prog := newProgress(logger)
	result, err := parser.Parse(filePath, opts.resolveOptions(ctx))
	if err != nil {
		return err
	}
	g := result.Graph.(*dag.DAG)
	prog.done(fmt.Sprintf("Parsed %d packages with %d dependencies", g.NodeCount(), g.EdgeCount()))

	return writeGraph(g, opts.output, logger)
}

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

type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }

func openOutput(path string) (io.WriteCloser, error) {
	if path == "" {
		return nopCloser{os.Stdout}, nil
	}
	return os.Create(path)
}
