package source

import (
	"context"
	"maps"
	"sync"
	"time"

	"stacktower/pkg/dag"
)

const (
	DefaultMaxDepth = 50
	DefaultMaxNodes = 5000
	DefaultCacheTTL = 24 * time.Hour
	numWorkers      = 20
)

type Parser interface {
	Parse(ctx context.Context, pkg string, opts Options) (*dag.DAG, error)
}

type MetadataProvider interface {
	Name() string
	Enrich(ctx context.Context, repo *RepoInfo, refresh bool) (map[string]any, error)
}

type RepoInfo struct {
	Name         string
	Version      string
	ProjectURLs  map[string]string
	HomePage     string
	ManifestFile string // e.g., "Cargo.toml", "pyproject.toml", "package.json"
}

type Options struct {
	MaxDepth          int
	MaxNodes          int
	CacheTTL          time.Duration
	Refresh           bool
	MetadataProviders []MetadataProvider
	Logger            func(string, ...any)
}

func (o Options) withDefaults() Options {
	if o.MaxDepth == 0 {
		o.MaxDepth = DefaultMaxDepth
	}
	if o.MaxNodes == 0 {
		o.MaxNodes = DefaultMaxNodes
	}
	if o.CacheTTL == 0 {
		o.CacheTTL = DefaultCacheTTL
	}
	if o.Logger == nil {
		o.Logger = func(string, ...any) {}
	}
	return o
}

type PackageInfo interface {
	GetName() string
	GetVersion() string
	GetDependencies() []string
	ToMetadata() map[string]any
	ToRepoInfo() *RepoInfo
}

type fetchFunc[T PackageInfo] func(ctx context.Context, name string, refresh bool) (T, error)

type job struct {
	name  string
	depth int
}

type result[T PackageInfo] struct {
	name  string
	info  T
	depth int
	err   error
}

func Parse[T PackageInfo](ctx context.Context, root string, opts Options, fetch fetchFunc[T]) (*dag.DAG, error) {
	opts = opts.withDefaults()

	p := &parser[T]{
		ctx:     ctx,
		opts:    opts,
		fetch:   fetch,
		g:       dag.New(nil),
		visited: make(map[string]bool),
		meta:    make(map[string]map[string]any),
		jobs:    make(chan job, numWorkers),
		results: make(chan result[T], numWorkers),
	}

	return p.parse(root)
}

type parser[T PackageInfo] struct {
	ctx   context.Context
	opts  Options
	fetch fetchFunc[T]

	g       *dag.DAG
	visited map[string]bool
	meta    map[string]map[string]any

	jobs    chan job
	results chan result[T]

	mu     sync.Mutex
	active int
}

func (p *parser[T]) parse(root string) (*dag.DAG, error) {
	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.worker()
		}()
	}

	go func() {
		wg.Wait()
		close(p.results)
	}()

	p.submit(job{name: root, depth: 0})

	rootErr := p.processResults(root)
	close(p.jobs)

	if rootErr != nil {
		return nil, rootErr
	}

	p.applyMetadata()
	return p.g, nil
}

func (p *parser[T]) worker() {
	for j := range p.jobs {
		if p.ctx.Err() != nil {
			return
		}
		info, err := p.fetch(p.ctx, j.name, p.opts.Refresh)
		p.results <- result[T]{name: j.name, info: info, depth: j.depth, err: err}
	}
}

func (p *parser[T]) submit(j job) {
	p.mu.Lock()
	if p.visited[j.name] {
		p.mu.Unlock()
		return
	}
	p.visited[j.name] = true
	p.active++
	p.mu.Unlock()

	p.jobs <- j
}

func (p *parser[T]) processResults(root string) error {
	for r := range p.results {
		p.decrementActive()

		if r.err != nil {
			if r.name == root {
				return r.err
			}
			p.opts.Logger("failed to fetch %s: %v", r.name, r.err)
		} else {
			p.processResult(r)
		}

		if p.isDone() {
			break
		}
	}
	return nil
}

func (p *parser[T]) processResult(r result[T]) {
	_ = p.g.AddNode(dag.Node{ID: r.name})

	meta := enrichMetadata(p.ctx, r.info, p.opts)
	p.storeMeta(r.name, meta)

	if r.depth >= p.opts.MaxDepth || len(p.visited) >= p.opts.MaxNodes {
		return
	}

	for _, dep := range r.info.GetDependencies() {
		_ = p.g.AddNode(dag.Node{ID: dep})
		_ = p.g.AddEdge(dag.Edge{From: r.name, To: dep})
		p.submit(job{name: dep, depth: r.depth + 1})
	}
}

func (p *parser[T]) applyMetadata() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for id, m := range p.meta {
		if n, ok := p.g.Node(id); ok {
			n.Meta = m
		}
	}
}

func (p *parser[T]) decrementActive() {
	p.mu.Lock()
	p.active--
	p.mu.Unlock()
}

func (p *parser[T]) isDone() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.active == 0
}

func (p *parser[T]) storeMeta(name string, meta map[string]any) {
	if len(meta) == 0 {
		return
	}
	p.mu.Lock()
	p.meta[name] = meta
	p.mu.Unlock()
}

func enrichMetadata(ctx context.Context, info PackageInfo, opts Options) map[string]any {
	m := info.ToMetadata()
	repo := info.ToRepoInfo()
	for _, provider := range opts.MetadataProviders {
		enriched, err := provider.Enrich(ctx, repo, opts.Refresh)
		if err != nil {
			opts.Logger("failed to enrich %s via %s: %v", info.GetName(), provider.Name(), err)
			continue
		}
		maps.Copy(m, enriched)
	}
	return m
}
