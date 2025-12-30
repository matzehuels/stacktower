package api

import (
	"context"
	"time"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/infra"
	"github.com/matzehuels/stacktower/pkg/infra/queue"
	"github.com/matzehuels/stacktower/pkg/infra/session"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// =============================================================================
// Handler Dependencies - Interfaces for testability
// =============================================================================
//
// These interfaces define the contracts that handlers depend on. By programming
// to interfaces rather than concrete types, we enable:
//
//   - Unit testing handlers with mock implementations
//   - Swapping implementations without changing handler code
//   - Clear documentation of what each handler actually needs
//
// Usage in tests:
//
//	func TestHandleRender(t *testing.T) {
//	    mockQueue := &MockQueue{...}
//	    mockPipeline := &MockPipelineService{...}
//	    server := &Server{queue: mockQueue, pipeline: mockPipeline, ...}
//	    // test handler
//	}

// Queue defines the job queue operations used by handlers.
// Implemented by: queue.RedisQueue
type Queue interface {
	Enqueue(ctx context.Context, job *queue.Job) error
	Get(ctx context.Context, jobID string) (*queue.Job, error)
	Delete(ctx context.Context, jobID string) error
	Cancel(ctx context.Context, jobID string) error
	ListByUser(ctx context.Context, userID string, statuses ...queue.Status) ([]*queue.Job, error)
	UpdateStatus(ctx context.Context, jobID string, status queue.Status, result map[string]interface{}, errMsg string) error
	Ping(ctx context.Context) error
}

// PipelineService defines the pipeline operations used by handlers.
// Implemented by: *pipeline.Service
type PipelineService interface {
	// Cache lookups (fast path)
	GetCachedGraph(ctx context.Context, opts pipeline.Options) (*dag.DAG, []byte, string, bool)
	GetCachedRenderRecord(ctx context.Context, opts pipeline.Options) *pipeline.RenderRecord

	// Synchronous operations
	Visualize(ctx context.Context, layoutData []byte, g *dag.DAG, opts pipeline.Options) (map[string][]byte, bool, error)
}

// SessionStore defines session operations used by handlers.
// Implemented by: session.RedisStore, session.MemoryStore
type SessionStore interface {
	Get(ctx context.Context, sessionID string) (*session.Session, error)
	Set(ctx context.Context, sess *session.Session) error
	Delete(ctx context.Context, sessionID string) error
}

// StateStore defines OAuth state operations used by handlers.
// Implemented by: session.RedisStateStore, session.MemoryStateStore
type StateStore interface {
	Generate(ctx context.Context, ttl time.Duration) (string, error)
	Validate(ctx context.Context, state string) (bool, error)
}

// DocumentStore defines document operations used by handlers.
// This is a subset of storage.DocumentStore focused on what handlers need.
type DocumentStore interface {
	// Renders
	GetRenderDoc(ctx context.Context, renderID string) (*storage.Render, error)
	GetRenderDocScoped(ctx context.Context, renderID, userID string) (*storage.Render, error)
	DeleteRenderDocScoped(ctx context.Context, renderID, userID string) error
	ListRenderDocs(ctx context.Context, userID string, limit, offset int) ([]*storage.Render, int64, error)

	// Graphs
	GetGraphDocScoped(ctx context.Context, graphID, userID string) (*storage.Graph, error)

	// Artifacts
	GetArtifactScoped(ctx context.Context, artifactID, userID string) ([]byte, error)

	// Stats (public)
	CountUniqueTowers(ctx context.Context) (int64, error)
	CountUniqueUsers(ctx context.Context) (int64, error)
	CountUniqueDependencies(ctx context.Context) (int64, error)
}

// Backend defines the distributed backend operations used by handlers.
// Implemented by: *storage.DistributedBackend
type Backend interface {
	DocumentStore() storage.DocumentStore
	CheckRateLimit(ctx context.Context, userID string, opType storage.OperationType, quota storage.QuotaConfig) error
	Ping(ctx context.Context) error
}

// =============================================================================
// Compile-time interface checks
// =============================================================================
//
// These ensure our interfaces stay in sync with the concrete types.
// If a method signature changes, compilation will fail here.

var (
	_ Queue           = (queue.Queue)(nil)
	_ PipelineService = (*pipeline.Service)(nil)
	_ SessionStore    = (session.Store)(nil)
	_ StateStore      = (session.StateStore)(nil)
)

// =============================================================================
// Handler Context - Grouped dependencies for handlers
// =============================================================================

// HandlerContext groups the dependencies that handlers actually need.
// This provides several benefits over accessing Server fields directly:
//
//   - Documents exactly what each handler requires
//   - Makes unit testing easier (construct with mocks)
//   - Reduces coupling to the Server struct
//
// Example usage in a handler:
//
//	func (s *Server) handleRender(w http.ResponseWriter, r *http.Request) {
//	    ctx := s.handlerContext()
//	    // use ctx.Queue, ctx.Pipeline, etc.
//	}
//
// Example usage in tests:
//
//	func TestHandleRender(t *testing.T) {
//	    ctx := &HandlerContext{
//	        Queue:    &mockQueue{},
//	        Pipeline: &mockPipeline{},
//	        Logger:   MockLogger(),
//	    }
//	    // inject ctx into handler under test
//	}
type HandlerContext struct {
	Queue    Queue
	Backend  Backend
	Pipeline PipelineService
	Sessions SessionStore
	States   StateStore
	Logger   *infra.Logger
	Quota    storage.QuotaConfig
}

// handlerContext creates a HandlerContext from the Server's dependencies.
// This is the bridge between the Server struct and the HandlerContext interface.
func (s *Server) handlerContext() *HandlerContext {
	return &HandlerContext{
		Queue:    s.queue,
		Backend:  s.backend,
		Pipeline: s.pipeline,
		Sessions: s.sessions,
		States:   s.states,
		Logger:   s.logger,
		Quota:    s.quota,
	}
}

// =============================================================================
// Mock Helpers for Testing
// =============================================================================

// MockLogger returns a logger that discards all output.
// Useful for tests that don't need to verify log output.
func MockLogger() *infra.Logger {
	return infra.DiscardLogger()
}

// NewTestContext creates a HandlerContext with all nil dependencies and a discard logger.
// Use this as a starting point for tests, then set only the mocks you need:
//
//	ctx := NewTestContext()
//	ctx.Queue = &mockQueue{...}
func NewTestContext() *HandlerContext {
	return &HandlerContext{
		Logger: MockLogger(),
	}
}
