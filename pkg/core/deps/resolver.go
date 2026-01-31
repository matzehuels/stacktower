package deps

import (
	"context"
	"maps"
	"sync"
	"sync/atomic"

	"github.com/matzehuels/stacktower/pkg/core/dag"
)

// workers is the number of concurrent goroutines for fetching packages.
// This limits parallelism to prevent overwhelming registries and to bound
// memory usage. Each worker consumes one job at a time from a buffered channel.
const workers = 20

// Fetcher retrieves package metadata from a registry.
//
// Implementations wrap HTTP clients for specific registries (PyPI, npm, crates.io).
// The Fetcher is responsible for HTTP caching, rate limiting, and error handling.
//
// Fetchers are found in the integrations subpackages (e.g., integrations/pypi).
type Fetcher interface {
	// Fetch retrieves package information by name.
	//
	// The name is the package identifier in the registry (e.g., "requests", "serde").
	// If refresh is true, cached HTTP responses are bypassed and fresh data is fetched.
	//
	// Returns an error if:
	//   - The package does not exist in the registry
	//   - The registry API is unreachable or returns an error
	//   - The response cannot be parsed
	//
	// Implementations should respect context cancellation and return ctx.Err()
	// when the context is canceled.
	//
	// Fetch must be safe for concurrent use by multiple goroutines.
	Fetch(ctx context.Context, name string, refresh bool) (*Package, error)
}

// Resolver builds a dependency graph starting from a root package.
//
// Implementations typically wrap a [Fetcher] and provide concurrent crawling
// logic. The [Registry] type is the standard implementation.
type Resolver interface {
	// Resolve fetches the package and its transitive dependencies.
	//
	// Starting from pkg, the resolver recursively fetches dependencies up to
	// Options.MaxDepth levels deep and Options.MaxNodes total packages.
	//
	// Returns a [dag.DAG] where:
	//   - Nodes represent packages (ID = package name)
	//   - Edges represent dependencies (From depends on To)
	//   - Node.Meta contains package metadata from [Package.Metadata] and
	//     enrichment from Options.MetadataProviders
	//
	// The DAG is fully connected from the root package. Isolated nodes may
	// appear if dependency fetching fails for non-root packages.
	//
	// Returns an error if:
	//   - The root package cannot be fetched (registry error or does not exist)
	//   - The context is canceled
	//   - Internal errors occur
	//
	// Partial failures (missing transitive dependencies) are logged via
	// Options.Logger but do not fail the entire resolution.
	//
	// Resolve is safe for concurrent use if the underlying Fetcher is safe.
	Resolve(ctx context.Context, pkg string, opts Options) (*dag.DAG, error)

	// Name returns the resolver's identifier (e.g., "pypi", "npm", "crates").
	//
	// This is used for logging and error messages.
	Name() string
}

// Registry implements Resolver by wrapping a Fetcher with concurrent crawling.
//
// Registry uses a worker pool to fetch packages concurrently, respecting
// Options limits (MaxDepth, MaxNodes). It tracks visited packages to avoid
// redundant fetches and handles cycles gracefully.
//
// Use [NewRegistry] to construct instances.
type Registry struct {
	name    string
	fetcher Fetcher
}

// NewRegistry creates a Resolver that crawls dependencies using the given Fetcher.
//
// The name is the resolver identifier (e.g., "pypi", "npm") and appears in
// Name() results and error messages.
//
// The fetcher must be safe for concurrent use, as multiple worker goroutines
// will call Fetch simultaneously.
func NewRegistry(name string, fetcher Fetcher) *Registry {
	return &Registry{name: name, fetcher: fetcher}
}

// Name returns the registry name.
func (r *Registry) Name() string { return r.name }

// Resolve crawls dependencies starting from pkg, respecting Options limits.
//
// This method spawns a worker pool of goroutines that fetch packages concurrently.
// The crawl proceeds breadth-first, tracking visited packages to avoid duplicates.
//
// Resolution stops when:
//   - All reachable dependencies are fetched
//   - MaxDepth is reached
//   - MaxNodes is reached (deeper dependencies are ignored)
//   - The context is canceled
//
// Failed fetches for non-root packages are logged but do not fail resolution.
// Failed fetches for the root package return an error immediately.
//
// The returned DAG has nodes for all successfully fetched packages and edges
// for all declared dependencies (even if the target package fetch failed).
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

// crawler manages concurrent package fetching with depth and node limits.
//
// It uses a worker pool pattern: jobs are enqueued to a channel, workers fetch
// packages concurrently, and results are collected in a single goroutine to
// avoid data races on the DAG and metadata map.
//
// The crawler tracks visited packages to avoid duplicate fetches and maintains
// a pending counter to know when all work is complete.
type crawler struct {
	ctx   context.Context
	opts  Options
	fetch func(context.Context, string, bool) (*Package, error)

	g    *dag.DAG                  // The dependency graph being built
	meta map[string]map[string]any // Metadata to apply after crawl completes

	jobs    chan job    // Work queue for package fetch jobs
	results chan result // Results from worker goroutines
	wg      sync.WaitGroup

	mu        sync.Mutex      // Protects visited map and meta map writes
	visited   map[string]bool // Tracks which packages have been queued
	pending   int64           // Atomic counter of in-flight jobs
	nodeCount int32           // Atomic counter of total nodes added
	closing   int32           // Atomic flag: 1 when shutting down (prevents sends to closed channel)
}

// job represents a package fetch task with depth tracking.
type job struct {
	name  string // Package name to fetch
	depth int    // Depth from root (root = 0)
}

// result holds the outcome of a fetch job.
type result struct {
	job
	pkg *Package // Fetched package metadata (nil if err is set)
	err error    // Fetch error (nil on success)
}

// run executes the crawl by starting workers, enqueuing the root, collecting
// results, and applying metadata. Returns the completed DAG or an error.
func (c *crawler) run(root string) (*dag.DAG, error) {
	// Start worker pool
	for range workers {
		c.wg.Add(1)
		go c.worker()
	}

	// Kick off crawl with root package
	c.enqueue(job{name: root})
	if err := c.collect(root); err != nil {
		// Signal shutdown before closing channel to prevent send-on-closed-channel panics
		atomic.StoreInt32(&c.closing, 1)
		close(c.jobs)
		c.wg.Wait()
		return nil, err
	}

	// Wait for workers to finish and apply collected metadata
	atomic.StoreInt32(&c.closing, 1)
	close(c.jobs)
	c.wg.Wait()
	c.applyMeta()

	return c.g, nil
}

// worker fetches packages from the jobs channel until it closes.
// Each fetch result is sent to the results channel for processing.
func (c *crawler) worker() {
	defer c.wg.Done()
	for j := range c.jobs {
		// Respect context cancellation
		if c.ctx.Err() != nil {
			atomic.AddInt64(&c.pending, -1)
			continue
		}
		pkg, err := c.fetch(c.ctx, j.name, c.opts.Refresh)
		c.results <- result{job: j, pkg: pkg, err: err}
	}
}

// enqueue adds a job to the work queue if the package hasn't been visited.
// Returns false if the package was already visited (duplicate).
func (c *crawler) enqueue(j job) bool {
	// Check if shutting down before doing anything
	if atomic.LoadInt32(&c.closing) == 1 {
		return false
	}

	c.mu.Lock()
	if c.visited[j.name] {
		c.mu.Unlock()
		return false
	}
	c.visited[j.name] = true
	c.mu.Unlock()

	atomic.AddInt64(&c.pending, 1)

	// Send to jobs channel in a goroutine to avoid blocking
	go func() {
		defer func() {
			// If send panics (closed channel during shutdown), treat as cancelled
			if recover() != nil {
				atomic.AddInt64(&c.pending, -1)
			}
		}()

		// Double-check closing flag to reduce chance of send-on-closed-channel
		if atomic.LoadInt32(&c.closing) == 1 {
			atomic.AddInt64(&c.pending, -1)
			return
		}

		select {
		case c.jobs <- j:
			// Successfully enqueued
		case <-c.ctx.Done():
			// Context canceled, abort
			atomic.AddInt64(&c.pending, -1)
		}
	}()
	return true
}

// collect processes results from workers until all pending jobs complete.
// Returns an error if the root package fails or if the context is canceled.
func (c *crawler) collect(root string) error {
	for {
		select {
		case r := <-c.results:
			if err := c.handle(r, root); err != nil {
				return err
			}
			// Check if all work is done
			if atomic.AddInt64(&c.pending, -1) == 0 {
				return nil
			}
		case <-c.ctx.Done():
			return c.ctx.Err()
		}
	}
}

// handle processes a single fetch result: adds nodes/edges, enriches metadata,
// and enqueues dependencies. Returns an error only if the root package fails.
func (c *crawler) handle(r result, root string) error {
	if r.err != nil {
		// Root package errors are fatal; others are logged
		if r.name == root {
			return r.err
		}
		c.opts.Logger("fetch failed: %s: %v", r.name, r.err)
		return nil
	}

	// Add package node to graph (mutex protects DAG mutations)
	c.mu.Lock()
	_ = c.g.AddNode(dag.Node{ID: r.name})
	c.mu.Unlock()
	atomic.AddInt32(&c.nodeCount, 1)

	// Collect metadata for later application
	if meta := c.enrich(r.pkg); len(meta) > 0 {
		c.mu.Lock()
		c.meta[r.name] = meta
		c.mu.Unlock()
	}

	c.enqueueDeps(r)
	return nil
}

// enqueueDeps adds dependency edges and enqueues child packages if limits allow.
func (c *crawler) enqueueDeps(r result) {
	// Stop if at depth limit or no dependencies
	if r.depth >= c.opts.MaxDepth || len(r.pkg.Dependencies) == 0 {
		return
	}

	next := r.depth + 1
	count := atomic.LoadInt32(&c.nodeCount)

	for _, dep := range r.pkg.Dependencies {
		// Always add nodes and edges, even if not fetching (mutex protects DAG)
		c.mu.Lock()
		_ = c.g.AddNode(dag.Node{ID: dep})
		_ = c.g.AddEdge(dag.Edge{From: r.name, To: dep})
		c.mu.Unlock()

		// Only fetch if under node limit
		if int(count) < c.opts.MaxNodes {
			c.enqueue(job{name: dep, depth: next})
		}
	}
}

// applyMeta attaches collected metadata to nodes in the DAG.
// Called after all fetching is complete to avoid concurrent node modifications.
func (c *crawler) applyMeta() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, m := range c.meta {
		if n, ok := c.g.Node(id); ok {
			n.Meta = m
		}
	}
}

// enrich combines package metadata with external provider data.
// Calls all MetadataProviders concurrently (providers must be goroutine-safe).
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
