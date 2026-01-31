// Package observability provides hooks for metrics, tracing, and logging.
//
// This package enables optional instrumentation without adding hard dependencies
// on specific observability backends. Consumers can register hooks at startup
// to receive events about pipeline execution, cache operations, and API calls.
//
// # Architecture
//
// The package uses a simple hooks pattern:
//   - Define hook interfaces for different event categories
//   - Provide no-op default implementations
//   - Allow registration of custom implementations at startup
//
// This approach:
//   - Avoids import cycles (hooks are registered by main, not by libraries)
//   - Keeps the core library dependency-free from observability frameworks
//   - Allows different backends (OpenTelemetry, Prometheus, DataDog, etc.)
//
// # Usage
//
// Register hooks at application startup:
//
//	func main() {
//	    observability.SetPipelineHooks(&myPipelineHooks{})
//	    observability.SetCacheHooks(&myCacheHooks{})
//	    // ... run application
//	}
//
// Libraries call hooks to emit events:
//
//	observability.Pipeline().OnParseStart(ctx, language, pkg)
//	// ... do parsing ...
//	observability.Pipeline().OnParseComplete(ctx, language, pkg, nodeCount, duration, err)
package observability

import (
	"context"
	"sync"
	"time"
)

// =============================================================================
// Pipeline Hooks
// =============================================================================

// PipelineHooks receives events from the visualization pipeline.
type PipelineHooks interface {
	// Parse events
	OnParseStart(ctx context.Context, language, pkg string)
	OnParseComplete(ctx context.Context, language, pkg string, nodeCount int, duration time.Duration, err error)

	// Layout events
	OnLayoutStart(ctx context.Context, vizType string, nodeCount int)
	OnLayoutComplete(ctx context.Context, vizType string, duration time.Duration, err error)

	// Render events
	OnRenderStart(ctx context.Context, formats []string)
	OnRenderComplete(ctx context.Context, formats []string, duration time.Duration, err error)
}

// =============================================================================
// Cache Hooks
// =============================================================================

// CacheHooks receives events from cache operations.
type CacheHooks interface {
	// OnCacheHit records a cache hit.
	OnCacheHit(ctx context.Context, keyType string)

	// OnCacheMiss records a cache miss.
	OnCacheMiss(ctx context.Context, keyType string)

	// OnCacheSet records a cache write.
	OnCacheSet(ctx context.Context, keyType string, size int)
}

// =============================================================================
// HTTP Hooks
// =============================================================================

// HTTPHooks receives events from HTTP client operations.
type HTTPHooks interface {
	// OnRequest records an outgoing HTTP request.
	OnRequest(ctx context.Context, method, host, path string)

	// OnResponse records an HTTP response.
	OnResponse(ctx context.Context, method, host, path string, statusCode int, duration time.Duration)

	// OnError records an HTTP error (network failure, timeout).
	OnError(ctx context.Context, method, host, path string, err error)
}

// =============================================================================
// No-op Implementations
// =============================================================================

// NoopPipelineHooks is a no-op implementation of PipelineHooks.
type NoopPipelineHooks struct{}

func (NoopPipelineHooks) OnParseStart(context.Context, string, string) {}
func (NoopPipelineHooks) OnParseComplete(context.Context, string, string, int, time.Duration, error) {
}
func (NoopPipelineHooks) OnLayoutStart(context.Context, string, int)                       {}
func (NoopPipelineHooks) OnLayoutComplete(context.Context, string, time.Duration, error)   {}
func (NoopPipelineHooks) OnRenderStart(context.Context, []string)                          {}
func (NoopPipelineHooks) OnRenderComplete(context.Context, []string, time.Duration, error) {}

// NoopCacheHooks is a no-op implementation of CacheHooks.
type NoopCacheHooks struct{}

func (NoopCacheHooks) OnCacheHit(context.Context, string)      {}
func (NoopCacheHooks) OnCacheMiss(context.Context, string)     {}
func (NoopCacheHooks) OnCacheSet(context.Context, string, int) {}

// NoopHTTPHooks is a no-op implementation of HTTPHooks.
type NoopHTTPHooks struct{}

func (NoopHTTPHooks) OnRequest(context.Context, string, string, string)                      {}
func (NoopHTTPHooks) OnResponse(context.Context, string, string, string, int, time.Duration) {}
func (NoopHTTPHooks) OnError(context.Context, string, string, string, error)                 {}

// =============================================================================
// Global Hook Registry
// =============================================================================

var (
	pipelineHooks PipelineHooks = NoopPipelineHooks{}
	cacheHooks    CacheHooks    = NoopCacheHooks{}
	httpHooks     HTTPHooks     = NoopHTTPHooks{}
	hooksMu       sync.RWMutex
)

// SetPipelineHooks registers custom pipeline hooks.
// This should be called once at application startup before any pipeline operations.
func SetPipelineHooks(h PipelineHooks) {
	hooksMu.Lock()
	defer hooksMu.Unlock()
	if h != nil {
		pipelineHooks = h
	}
}

// SetCacheHooks registers custom cache hooks.
// This should be called once at application startup before any cache operations.
func SetCacheHooks(h CacheHooks) {
	hooksMu.Lock()
	defer hooksMu.Unlock()
	if h != nil {
		cacheHooks = h
	}
}

// SetHTTPHooks registers custom HTTP hooks.
// This should be called once at application startup before any HTTP operations.
func SetHTTPHooks(h HTTPHooks) {
	hooksMu.Lock()
	defer hooksMu.Unlock()
	if h != nil {
		httpHooks = h
	}
}

// Pipeline returns the registered pipeline hooks.
func Pipeline() PipelineHooks {
	hooksMu.RLock()
	defer hooksMu.RUnlock()
	return pipelineHooks
}

// Cache returns the registered cache hooks.
func Cache() CacheHooks {
	hooksMu.RLock()
	defer hooksMu.RUnlock()
	return cacheHooks
}

// HTTP returns the registered HTTP hooks.
func HTTP() HTTPHooks {
	hooksMu.RLock()
	defer hooksMu.RUnlock()
	return httpHooks
}

// Reset restores all hooks to their no-op defaults.
// This is primarily useful for testing.
func Reset() {
	hooksMu.Lock()
	defer hooksMu.Unlock()
	pipelineHooks = NoopPipelineHooks{}
	cacheHooks = NoopCacheHooks{}
	httpHooks = NoopHTTPHooks{}
}
