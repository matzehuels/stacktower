package observability

import (
	"context"
	"testing"
	"time"
)

func TestNoopHooksDoNotPanic(t *testing.T) {
	ctx := context.Background()

	// Pipeline hooks
	p := NoopPipelineHooks{}
	p.OnParseStart(ctx, "python", "requests")
	p.OnParseComplete(ctx, "python", "requests", 100, time.Second, nil)
	p.OnLayoutStart(ctx, "tower", 100)
	p.OnLayoutComplete(ctx, "tower", time.Second, nil)
	p.OnRenderStart(ctx, []string{"svg"})
	p.OnRenderComplete(ctx, []string{"svg"}, time.Second, nil)

	// Cache hooks
	c := NoopCacheHooks{}
	c.OnCacheHit(ctx, "graph")
	c.OnCacheMiss(ctx, "layout")
	c.OnCacheSet(ctx, "artifact", 1024)

	// HTTP hooks
	h := NoopHTTPHooks{}
	h.OnRequest(ctx, "GET", "pypi.org", "/simple/requests")
	h.OnResponse(ctx, "GET", "pypi.org", "/simple/requests", 200, time.Second)
	h.OnError(ctx, "GET", "pypi.org", "/simple/requests", nil)
}

func TestGlobalHooksRegistry(t *testing.T) {
	// Reset to known state
	Reset()

	// Verify defaults are noop
	if _, ok := Pipeline().(NoopPipelineHooks); !ok {
		t.Error("Pipeline() should return NoopPipelineHooks by default")
	}
	if _, ok := Cache().(NoopCacheHooks); !ok {
		t.Error("Cache() should return NoopCacheHooks by default")
	}
	if _, ok := HTTP().(NoopHTTPHooks); !ok {
		t.Error("HTTP() should return NoopHTTPHooks by default")
	}

	// Set custom hooks
	customPipeline := &testPipelineHooks{}
	SetPipelineHooks(customPipeline)
	if Pipeline() != customPipeline {
		t.Error("SetPipelineHooks should set custom hooks")
	}

	customCache := &testCacheHooks{}
	SetCacheHooks(customCache)
	if Cache() != customCache {
		t.Error("SetCacheHooks should set custom hooks")
	}

	customHTTP := &testHTTPHooks{}
	SetHTTPHooks(customHTTP)
	if HTTP() != customHTTP {
		t.Error("SetHTTPHooks should set custom hooks")
	}

	// Reset and verify
	Reset()
	if _, ok := Pipeline().(NoopPipelineHooks); !ok {
		t.Error("Reset() should restore NoopPipelineHooks")
	}
}

func TestSetNilHooksIsIgnored(t *testing.T) {
	Reset()

	custom := &testPipelineHooks{}
	SetPipelineHooks(custom)

	// Setting nil should be ignored
	SetPipelineHooks(nil)

	if Pipeline() != custom {
		t.Error("SetPipelineHooks(nil) should be ignored")
	}

	Reset()
}

// Test implementations
type testPipelineHooks struct{ NoopPipelineHooks }
type testCacheHooks struct{ NoopCacheHooks }
type testHTTPHooks struct{ NoopHTTPHooks }
