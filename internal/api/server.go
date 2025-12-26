// Package api provides the HTTP REST API for Stacktower.
//
// The API implements a two-tier cache architecture:
//   - Tier 1 (Redis): Fast TTL-based lookups
//   - Tier 2 (MongoDB): Durable storage for graphs, renders, artifacts
//
// # Main Endpoint
//
//	POST /api/v1/render - Main rendering endpoint with cache chain
//
// The render endpoint implements:
//  1. Check Redis for graph cache entry
//  2. Fetch graph from MongoDB if cache hit
//  3. Check Redis for render cache entry
//  4. Fetch render from MongoDB if cache hit
//  5. Compute and store if cache miss
//  6. Stale-while-revalidate for expired entries
//
// # Supporting Endpoints
//
//	GET    /api/v1/render/:id    - Get render result by ID
//	DELETE /api/v1/render/:id    - Delete a render
//	GET    /api/v1/history       - List user's render history
//	GET    /api/v1/artifacts/:id - Download an artifact
//	GET    /api/v1/jobs/:id      - Get job status
//
// # Authentication
//
//	GET  /api/v1/auth/github          - Initiate GitHub OAuth
//	GET  /api/v1/auth/github/callback - OAuth callback
//	GET  /api/v1/auth/me              - Get current user
//	POST /api/v1/auth/logout          - Logout
//
// # Usage
//
//	server := api.New(queue, cache,
//	    api.WithHost("0.0.0.0"),
//	    api.WithPort(8080),
//	    api.WithSessions(sessionStore),
//	)
//
//	if err := server.Start(); err != nil {
//	    log.Fatal(err)
//	}
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/matzehuels/stacktower/pkg/infra/artifact"
	"github.com/matzehuels/stacktower/pkg/infra/cache"
	"github.com/matzehuels/stacktower/pkg/infra/common"
	"github.com/matzehuels/stacktower/pkg/infra/queue"
	"github.com/matzehuels/stacktower/pkg/infra/session"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// Server is the HTTP API server.
type Server struct {
	queue            queue.Queue
	cache            cache.Cache        // Two-tier cache (Redis lookup + MongoDB storage)
	pipeline         *pipeline.Service  // Pipeline service for parse/layout/render
	sessions         session.Store      // Session storage (memory or Redis)
	states           session.StateStore // OAuth state storage (memory or Redis)
	manifestPatterns map[string]string  // Manifest filename -> language
	logger           *common.Logger     // Logger for server events
	host             string             // Address to bind to
	port             int                // Port to listen on
	readTimeout      time.Duration      // Max duration for reading requests
	writeTimeout     time.Duration      // Max duration for writing responses
	maxRequestSize   int64              // Max request body size
	allowOrigins     []string           // CORS allowed origins
	http             *http.Server
}

// Option configures a Server.
type Option func(*Server)

// WithHost sets the address to bind to (default: "0.0.0.0").
func WithHost(host string) Option {
	return func(s *Server) { s.host = host }
}

// WithPort sets the port to listen on (default: 8080).
func WithPort(port int) Option {
	return func(s *Server) { s.port = port }
}

// WithReadTimeout sets the max duration for reading requests.
func WithReadTimeout(d time.Duration) Option {
	return func(s *Server) { s.readTimeout = d }
}

// WithWriteTimeout sets the max duration for writing responses.
func WithWriteTimeout(d time.Duration) Option {
	return func(s *Server) { s.writeTimeout = d }
}

// WithMaxRequestSize sets the max request body size (default: 10MB).
func WithMaxRequestSize(size int64) Option {
	return func(s *Server) { s.maxRequestSize = size }
}

// WithAllowOrigins sets the CORS allowed origins (default: ["*"]).
func WithAllowOrigins(origins []string) Option {
	return func(s *Server) { s.allowOrigins = origins }
}

// WithSessions sets the session store.
func WithSessions(store session.Store) Option {
	return func(s *Server) { s.sessions = store }
}

// WithStates sets the OAuth state store.
func WithStates(store session.StateStore) Option {
	return func(s *Server) { s.states = store }
}

// WithManifestPatterns sets the manifest detection patterns.
func WithManifestPatterns(patterns map[string]string) Option {
	return func(s *Server) { s.manifestPatterns = patterns }
}

// WithLogger sets the logger for server events.
func WithLogger(logger *common.Logger) Option {
	return func(s *Server) { s.logger = logger }
}

// New creates a new API server with the given options.
func New(q queue.Queue, c cache.Cache, opts ...Option) *Server {
	// Create pipeline service with production backend
	pipelineSvc := pipeline.NewService(artifact.NewProdBackend(c))

	srv := &Server{
		queue:          q,
		cache:          c,
		pipeline:       pipelineSvc,
		sessions:       session.NewMemoryStore(),
		states:         session.NewMemoryStateStore(),
		logger:         common.DiscardLogger(),
		host:           "0.0.0.0",
		port:           8080,
		readTimeout:    30 * time.Second,
		writeTimeout:   30 * time.Second,
		maxRequestSize: 10 * 1024 * 1024, // 10MB
		allowOrigins:   []string{"*"},
	}

	for _, opt := range opts {
		opt(srv)
	}

	mux := http.NewServeMux()

	// Pipeline endpoints (parse, layout, visualize, render)
	mux.HandleFunc("/api/v1/parse", srv.handleParse)
	mux.HandleFunc("/api/v1/layout", srv.handleLayout)
	mux.HandleFunc("/api/v1/visualize", srv.handleVisualize)
	mux.HandleFunc("/api/v1/render", srv.handleRender)
	mux.HandleFunc("/api/v1/render/", srv.handleRenderByID)

	// History endpoint
	mux.HandleFunc("/api/v1/history", srv.handleHistory)

	// Artifact endpoint (from cache/MongoDB GridFS)
	mux.HandleFunc("/api/v1/artifacts/", srv.handleArtifactByID)

	// Job management endpoints
	mux.HandleFunc("/api/v1/jobs/", srv.handleJobs)
	mux.HandleFunc("/api/v1/jobs", srv.handleJobsList)

	// GitHub OAuth endpoints
	mux.HandleFunc("/api/v1/auth/github", srv.handleGitHubAuth)
	mux.HandleFunc("/api/v1/auth/github/callback", srv.handleGitHubCallback)
	mux.HandleFunc("/api/v1/auth/me", srv.handleAuthMe)
	mux.HandleFunc("/api/v1/auth/logout", srv.handleAuthLogout)

	// GitHub repo endpoints
	mux.HandleFunc("/api/v1/repos", srv.handleRepos)
	mux.HandleFunc("/api/v1/repos/", srv.handleRepoRoutes)

	// Health check
	mux.HandleFunc("/health", srv.handleHealth)

	// Wrap with middleware
	handler := srv.corsMiddleware(srv.loggingMiddleware(mux))

	srv.http = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", srv.host, srv.port),
		Handler:      handler,
		ReadTimeout:  srv.readTimeout,
		WriteTimeout: srv.writeTimeout,
	}

	return srv
}

// Start starts the HTTP server.
// This method blocks until the server is shut down.
func (s *Server) Start() error {
	s.logger.Info("starting API server", "addr", s.http.Addr)
	return s.http.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down API server")
	return s.http.Shutdown(ctx)
}

// handleJobs handles GET/DELETE /api/v1/jobs/:id
func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	// Extract job ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/jobs/")
	if path == "" {
		s.errorResponse(w, http.StatusBadRequest, "job ID required")
		return
	}

	jobID := path

	switch r.Method {
	case http.MethodGet:
		s.handleGetJob(w, r, jobID)
	case http.MethodDelete:
		s.handleDeleteJob(w, r, jobID)
	default:
		s.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleGetJob retrieves job status
func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request, jobID string) {
	job, err := s.queue.Get(r.Context(), jobID)
	if err != nil {
		s.errorResponse(w, http.StatusNotFound, fmt.Sprintf("job not found: %v", err))
		return
	}

	s.jsonResponse(w, http.StatusOK, s.jobToResponse(job))
}

// handleDeleteJob cancels or deletes a job
func (s *Server) handleDeleteJob(w http.ResponseWriter, r *http.Request, jobID string) {
	// Try to cancel first (if pending)
	if err := s.queue.Cancel(r.Context(), jobID); err == nil {
		s.jsonResponse(w, http.StatusOK, map[string]string{"message": "job cancelled"})
		return
	}

	// Otherwise delete it
	if err := s.queue.Delete(r.Context(), jobID); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("delete job: %v", err))
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"message": "job deleted"})
}

// handleJobsList handles GET /api/v1/jobs
func (s *Server) handleJobsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Parse query parameters for filtering
	statusFilter := r.URL.Query().Get("status")
	var statuses []queue.Status
	if statusFilter != "" {
		statuses = []queue.Status{queue.Status(statusFilter)}
	}

	jobs, err := s.queue.List(r.Context(), statuses...)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("list jobs: %v", err))
		return
	}

	responses := make([]JobStatusResponse, len(jobs))
	for i, job := range jobs {
		responses[i] = s.jobToResponse(job)
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"jobs":  responses,
		"count": len(responses),
	})
}

// handleHealth handles GET /health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// Helper methods

func (s *Server) jobToResponse(job *queue.Job) JobStatusResponse {
	resp := JobStatusResponse{
		JobID:     job.ID,
		Type:      job.Type,
		Status:    string(job.Status),
		CreatedAt: job.CreatedAt,
	}

	if job.StartedAt != nil {
		resp.StartedAt = job.StartedAt
	}
	if job.CompletedAt != nil {
		resp.CompletedAt = job.CompletedAt
		duration := job.Duration()
		resp.Duration = &duration
	}
	if job.Result != nil {
		resp.Result = job.Result
	}
	if job.Error != "" {
		resp.Error = &job.Error
	}

	return resp
}

func (s *Server) decodeJSON(r *http.Request, v interface{}) error {
	// Limit request size
	r.Body = http.MaxBytesReader(nil, r.Body, s.maxRequestSize)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}

func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) errorResponse(w http.ResponseWriter, status int, message string) {
	s.jsonResponse(w, status, map[string]string{"error": message})
}

// Middleware

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Debug("request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && s.isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) isAllowedOrigin(origin string) bool {
	for _, allowed := range s.allowOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

// generateJobID generates a unique job ID using UUID v4.
func generateJobID() string {
	return uuid.New().String()
}
