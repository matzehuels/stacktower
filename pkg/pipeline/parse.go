package pipeline

import (
	"context"
	"fmt"
	"os"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	dagtransform "github.com/matzehuels/stacktower/pkg/core/dag/transform"
	"github.com/matzehuels/stacktower/pkg/core/deps"
	"github.com/matzehuels/stacktower/pkg/core/deps/languages"
	"github.com/matzehuels/stacktower/pkg/core/deps/metadata"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
)

// Parse resolves dependencies for a package or manifest.
func Parse(ctx context.Context, backend storage.Backend, opts Options) (*dag.DAG, error) {
	lang := languages.Find(opts.Language)
	if lang == nil {
		return nil, fmt.Errorf("unsupported language: %s", opts.Language)
	}

	resolveOpts := buildResolveOptions(backend, opts)

	var g *dag.DAG
	var err error

	if opts.Manifest != "" {
		g, err = parseManifest(ctx, backend, lang, opts, resolveOpts)
	} else {
		g, err = resolvePackage(ctx, backend, lang, opts.Package, resolveOpts)
	}

	if err != nil {
		return nil, err
	}

	if opts.Normalize {
		dagtransform.Normalize(g)
	}

	return g, nil
}

// buildResolveOptions creates deps.Options from pipeline options.
func buildResolveOptions(backend storage.Backend, opts Options) deps.Options {
	resolveOpts := deps.Options{
		MaxDepth: opts.MaxDepth,
		MaxNodes: opts.MaxNodes,
		Refresh:  opts.Refresh,
		CacheTTL: deps.DefaultCacheTTL,
	}

	// Set up logger callback
	if opts.Logger != nil {
		resolveOpts.Logger = func(format string, args ...any) {
			opts.Logger.Warnf(format, args...)
		}
	}

	// Set up metadata providers
	token := opts.GitHubToken
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if opts.ShouldEnrich() && token != "" {
		gh := metadata.NewGitHub(backend, token, deps.DefaultCacheTTL)
		resolveOpts.MetadataProviders = []deps.MetadataProvider{gh}
	}

	return resolveOpts
}

// resolvePackage resolves dependencies from a package registry.
func resolvePackage(ctx context.Context, backend storage.Backend, lang *deps.Language, pkg string, opts deps.Options) (*dag.DAG, error) {
	resolver, err := lang.Resolver(backend)
	if err != nil {
		return nil, fmt.Errorf("get resolver: %w", err)
	}

	// Normalize package name if the language supports it
	if lang.NormalizeName != nil {
		pkg = lang.NormalizeName(pkg)
	}

	g, err := resolver.Resolve(ctx, pkg, opts)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", pkg, err)
	}

	return g, nil
}

// parseManifest parses dependencies from a manifest file or content.
func parseManifest(ctx context.Context, backend storage.Backend, lang *deps.Language, opts Options, resolveOpts deps.Options) (*dag.DAG, error) {
	resolver, err := lang.Resolver(backend)
	if err != nil {
		return nil, fmt.Errorf("get resolver: %w", err)
	}

	parser, ok := lang.Manifest(opts.ManifestFilename, resolver)
	if !ok {
		return nil, fmt.Errorf("no parser for manifest: %s", opts.ManifestFilename)
	}

	// If manifest content is provided, write to temp file
	var filePath string
	if opts.Manifest != "" {
		tmpDir, err := os.MkdirTemp("", "stacktower-*")
		if err != nil {
			return nil, fmt.Errorf("create temp dir: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		filePath = tmpDir + "/" + opts.ManifestFilename
		if err := os.WriteFile(filePath, []byte(opts.Manifest), 0644); err != nil {
			return nil, fmt.Errorf("write temp file: %w", err)
		}
	} else {
		filePath = opts.ManifestFilename
	}

	result, err := parser.Parse(filePath, resolveOpts)
	if err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	g, ok := result.Graph.(*dag.DAG)
	if !ok {
		return nil, fmt.Errorf("unexpected graph type: %T", result.Graph)
	}

	return g, nil
}
