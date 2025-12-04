package source

import (
	"context"
	"maps"
	"sync"
	"time"

	"github.com/matzehuels/stacktower/pkg/dag"
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
	ManifestFile string
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

func Parse[T PackageInfo](ctx context.Context, root string, opts Options, fetch fetchFunc[T]) (*dag.DAG, error) {
	opts = opts.withDefaults()

	p := &parser[T]{
		ctx:     ctx,
		opts:    opts,
		fetch:   fetch,
		g:       dag.New(nil),
		visited: make(map[string]bool),
		meta:    make(map[string]map[string]any),
		jobs:    make(chan job, numWorkers*2),
		results: make(chan result[T], numWorkers*2),
		done:    make(chan struct{}),
	}

	return p.parse(root)
}

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

type parser[T PackageInfo] struct {
	ctx   context.Context
	opts  Options
	fetch fetchFunc[T]

	g       *dag.DAG
	visited map[string]bool
	meta    map[string]map[string]any

	jobs    chan job
	results chan result[T]
	done    chan struct{}

	mu        sync.Mutex
	inflight  int64
	nodeCount int32
}

func (p *parser[T]) adjustInflight(delta int64) {
	p.mu.Lock()
	p.inflight += delta
	isDone := p.inflight == 0
	p.mu.Unlock()

	if isDone {
		close(p.done)
	}
}

func (p *parser[T]) parse(root string) (*dag.DAG, error) {
	var workerWg sync.WaitGroup
	for range numWorkers {
		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			p.worker()
		}()
	}

	p.submit(job{name: root, depth: 0})

	rootErr := p.processResults(root)

	close(p.jobs)
	workerWg.Wait()

	if rootErr != nil {
		return nil, rootErr
	}

	p.applyMetadata()
	return p.g, nil
}

func (p *parser[T]) worker() {
	for j := range p.jobs {
		if p.ctx.Err() != nil {
			p.adjustInflight(-1) // job cancelled
			continue
		}
		info, err := p.fetch(p.ctx, j.name, p.opts.Refresh)
		p.results <- result[T]{name: j.name, info: info, depth: j.depth, err: err}
	}
}

func (p *parser[T]) submit(j job) bool {
	p.mu.Lock()
	if p.visited[j.name] {
		p.mu.Unlock()
		return false
	}
	p.visited[j.name] = true
	p.inflight++
	p.mu.Unlock()

	p.jobs <- j
	return true
}

func (p *parser[T]) processResults(root string) error {
	for {
		select {
		case r := <-p.results:
			if err := p.handleResult(r, root); err != nil {
				return err
			}

		case <-p.done:
			return nil

		case <-p.ctx.Done():
			return p.ctx.Err()
		}
	}
}

func (p *parser[T]) handleResult(r result[T], root string) error {
	defer p.adjustInflight(-1)

	if r.err != nil {
		if r.name == root {
			return r.err
		}
		p.opts.Logger("failed to fetch %s: %v", r.name, r.err)
		return nil
	}

	p.addNode(r)
	p.submitDependencies(r)
	return nil
}

func (p *parser[T]) addNode(r result[T]) {
	_ = p.g.AddNode(dag.Node{ID: r.name})

	p.mu.Lock()
	p.nodeCount++
	p.mu.Unlock()

	meta := enrichMetadata(p.ctx, r.info, p.opts)
	if len(meta) > 0 {
		p.mu.Lock()
		p.meta[r.name] = meta
		p.mu.Unlock()
	}
}

func (p *parser[T]) submitDependencies(r result[T]) {
	if r.depth >= p.opts.MaxDepth {
		return
	}

	deps := r.info.GetDependencies()
	if len(deps) == 0 {
		return
	}

	// Add edges and collect jobs
	p.mu.Lock()
	nodeCount := p.nodeCount
	p.mu.Unlock()

	var toSubmit []job
	for _, dep := range deps {
		_ = p.g.AddNode(dag.Node{ID: dep})
		_ = p.g.AddEdge(dag.Edge{From: r.name, To: dep})

		if int(nodeCount) < p.opts.MaxNodes {
			toSubmit = append(toSubmit, job{name: dep, depth: r.depth + 1})
		}
	}

	if len(toSubmit) == 0 {
		return
	}

	// Reserve a slot for the async submitter BEFORE spawning it.
	// This prevents processResults from exiting prematurely.
	p.mu.Lock()
	p.inflight++
	p.mu.Unlock()

	go func() {
		defer p.adjustInflight(-1) // release slot when done submitting

		for _, j := range toSubmit {
			p.submit(j)
		}
	}()
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
