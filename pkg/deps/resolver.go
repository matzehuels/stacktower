package deps

import (
	"context"
	"maps"
	"sync"
	"sync/atomic"

	"github.com/matzehuels/stacktower/pkg/dag"
)

const workers = 20

type Fetcher interface {
	Fetch(ctx context.Context, name string, refresh bool) (*Package, error)
}

type Resolver interface {
	Resolve(ctx context.Context, pkg string, opts Options) (*dag.DAG, error)
	Name() string
}

type Registry struct {
	name    string
	fetcher Fetcher
}

func NewRegistry(name string, fetcher Fetcher) *Registry {
	return &Registry{name: name, fetcher: fetcher}
}

func (r *Registry) Name() string { return r.name }

func (r *Registry) Resolve(ctx context.Context, pkg string, opts Options) (*dag.DAG, error) {
	c := &crawler{
		ctx:     ctx,
		opts:    opts.WithDefaults(),
		fetch:   r.fetcher.Fetch,
		g:       dag.New(nil),
		meta:    make(map[string]map[string]any),
		visited: make(map[string]bool),
		jobs:    make(chan job, workers*2),
		results: make(chan result, workers*2),
	}
	return c.run(pkg)
}

type crawler struct {
	ctx   context.Context
	opts  Options
	fetch func(context.Context, string, bool) (*Package, error)

	g    *dag.DAG
	meta map[string]map[string]any

	jobs    chan job
	results chan result
	wg      sync.WaitGroup

	mu        sync.Mutex
	visited   map[string]bool
	pending   int64
	nodeCount int32
}

type job struct {
	name  string
	depth int
}

type result struct {
	job
	pkg *Package
	err error
}

func (c *crawler) run(root string) (*dag.DAG, error) {
	for range workers {
		c.wg.Add(1)
		go c.worker()
	}

	c.enqueue(job{name: root})
	if err := c.collect(root); err != nil {
		close(c.jobs)
		c.wg.Wait()
		return nil, err
	}

	close(c.jobs)
	c.wg.Wait()
	c.applyMeta()

	return c.g, nil
}

func (c *crawler) worker() {
	defer c.wg.Done()
	for j := range c.jobs {
		if c.ctx.Err() != nil {
			atomic.AddInt64(&c.pending, -1)
			continue
		}
		pkg, err := c.fetch(c.ctx, j.name, c.opts.Refresh)
		c.results <- result{job: j, pkg: pkg, err: err}
	}
}

func (c *crawler) enqueue(j job) bool {
	c.mu.Lock()
	if c.visited[j.name] {
		c.mu.Unlock()
		return false
	}
	c.visited[j.name] = true
	c.mu.Unlock()

	atomic.AddInt64(&c.pending, 1)

	go func() { c.jobs <- j }()
	return true
}

func (c *crawler) collect(root string) error {
	for {
		select {
		case r := <-c.results:
			if err := c.handle(r, root); err != nil {
				return err
			}
			if atomic.AddInt64(&c.pending, -1) == 0 {
				return nil
			}
		case <-c.ctx.Done():
			return c.ctx.Err()
		}
	}
}

func (c *crawler) handle(r result, root string) error {
	if r.err != nil {
		if r.name == root {
			return r.err
		}
		c.opts.Logger("fetch failed: %s: %v", r.name, r.err)
		return nil
	}

	_ = c.g.AddNode(dag.Node{ID: r.name})
	atomic.AddInt32(&c.nodeCount, 1)

	if meta := c.enrich(r.pkg); len(meta) > 0 {
		c.mu.Lock()
		c.meta[r.name] = meta
		c.mu.Unlock()
	}

	c.enqueueDeps(r)
	return nil
}

func (c *crawler) enqueueDeps(r result) {
	if r.depth >= c.opts.MaxDepth || len(r.pkg.Dependencies) == 0 {
		return
	}

	next := r.depth + 1
	count := atomic.LoadInt32(&c.nodeCount)

	for _, dep := range r.pkg.Dependencies {
		_ = c.g.AddNode(dag.Node{ID: dep})
		_ = c.g.AddEdge(dag.Edge{From: r.name, To: dep})

		if int(count) < c.opts.MaxNodes {
			c.enqueue(job{name: dep, depth: next})
		}
	}
}

func (c *crawler) applyMeta() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, m := range c.meta {
		if n, ok := c.g.Node(id); ok {
			n.Meta = m
		}
	}
}

func (c *crawler) enrich(pkg *Package) map[string]any {
	m := pkg.Metadata()
	ref := pkg.Ref()
	for _, p := range c.opts.MetadataProviders {
		if enriched, err := p.Enrich(c.ctx, ref, c.opts.Refresh); err == nil {
			maps.Copy(m, enriched)
		} else {
			c.opts.Logger("enrich failed: %s: %v", pkg.Name, err)
		}
	}
	return m
}
