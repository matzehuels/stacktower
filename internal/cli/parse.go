package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	pkgio "github.com/matzehuels/stacktower/pkg/io"
	"github.com/matzehuels/stacktower/pkg/source"
	"github.com/matzehuels/stacktower/pkg/source/javascript"
	"github.com/matzehuels/stacktower/pkg/source/metadata"
	"github.com/matzehuels/stacktower/pkg/source/python"
	"github.com/matzehuels/stacktower/pkg/source/rust"
	"github.com/matzehuels/stacktower/pkg/source/ruby"
)

type parseOpts struct {
	maxDepth int
	maxNodes int
	enrich   bool
	refresh  bool
	output   string
}

type parserFactory func() (source.Parser, error)

func newParseCmd() *cobra.Command {
	opts := parseOpts{maxDepth: 10, maxNodes: 5000}

	cmd := &cobra.Command{
		Use:   "parse",
		Short: "Parse dependency graphs from package managers",
		Long:  `Parse dependency graphs from package managers (PyPI, crates.io, npm) and output as JSON.`,
	}

	cmd.PersistentFlags().IntVar(&opts.maxDepth, "max-depth", opts.maxDepth, "maximum dependency depth")
	cmd.PersistentFlags().IntVar(&opts.maxNodes, "max-nodes", opts.maxNodes, "maximum nodes to fetch")
	cmd.PersistentFlags().BoolVar(&opts.enrich, "enrich", false, "enrich with repository metadata")
	cmd.PersistentFlags().BoolVar(&opts.refresh, "refresh", false, "bypass cache")
	cmd.PersistentFlags().StringVarP(&opts.output, "output", "o", "", "output file (stdout if empty)")

	cmd.AddCommand(newParserCmd("python <package>", "Parse Python package dependencies from PyPI",
		func() (source.Parser, error) { return python.NewParser(source.DefaultCacheTTL) }, &opts))
	cmd.AddCommand(newParserCmd("rust <crate>", "Parse Rust crate dependencies from crates.io",
		func() (source.Parser, error) { return rust.NewParser(source.DefaultCacheTTL) }, &opts))
	cmd.AddCommand(newParserCmd("javascript <package>", "Parse JavaScript package dependencies from npm",
		func() (source.Parser, error) { return javascript.NewParser(source.DefaultCacheTTL) }, &opts))
	cmd.AddCommand(newParserCmd("ruby <gem>", "Parse Ruby gem dependencies from RubyGems",
		func() (source.Parser, error) { return ruby.NewParser(source.DefaultCacheTTL) }, &opts))

	return cmd
}

func newParserCmd(use, short string, factory parserFactory, opts *parseOpts) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := factory()
			if err != nil {
				return err
			}
			return runParse(cmd.Context(), p, args[0], opts)
		},
	}
}

func runParse(ctx context.Context, p source.Parser, pkg string, opts *parseOpts) error {
	logger := loggerFromContext(ctx)
	logger.Infof("Parsing %s dependencies", pkg)

	providers, err := buildMetadataProviders(opts.enrich)
	if err != nil {
		logger.Warnf("Metadata enrichment disabled: %v", err)
	} else if len(providers) > 0 {
		logger.Debugf("Metadata enrichment enabled (%d providers)", len(providers))
	}

	srcOpts := source.Options{
		MaxDepth:          opts.maxDepth,
		MaxNodes:          opts.maxNodes,
		MetadataProviders: providers,
		Refresh:           opts.refresh,
		CacheTTL:          source.DefaultCacheTTL,
		Logger:            func(msg string, args ...any) { logger.Warnf(msg, args...) },
	}

	logger.Info("Resolving dependency graph")
	prog := newProgress(logger)
	g, err := p.Parse(ctx, pkg, srcOpts)
	if err != nil {
		return err
	}
	prog.done(fmt.Sprintf("Resolved %d packages with %d dependencies", g.NodeCount(), g.EdgeCount()))

	out, err := openOutput(opts.output)
	if err != nil {
		return err
	}
	defer out.Close()

	if err := pkgio.WriteJSON(g, out); err != nil {
		return err
	}

	if opts.output != "" {
		logger.Infof("Wrote graph to %s", opts.output)
	}
	return nil
}

func buildMetadataProviders(enrich bool) ([]source.MetadataProvider, error) {
	if !enrich {
		return nil, nil
	}

	var providers []source.MetadataProvider
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		gh, err := metadata.NewGitHub(tok, source.DefaultCacheTTL)
		if err != nil {
			return nil, fmt.Errorf("github: %w", err)
		}
		providers = append(providers, gh)
	}
	if tok := os.Getenv("GITLAB_TOKEN"); tok != "" {
		gl, err := metadata.NewGitLab(tok, source.DefaultCacheTTL)
		if err != nil {
			return nil, fmt.Errorf("gitlab: %w", err)
		}
		providers = append(providers, gl)
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("no tokens found (GITHUB_TOKEN, GITLAB_TOKEN)")
	}
	return providers, nil
}

type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }

func openOutput(path string) (io.WriteCloser, error) {
	if path == "" {
		return nopCloser{os.Stdout}, nil
	}
	return os.Create(path)
}
