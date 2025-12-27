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
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/matzehuels/stacktower/pkg/infra"
	"github.com/matzehuels/stacktower/pkg/infra/queue"
	"github.com/matzehuels/stacktower/pkg/infra/session"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
	"github.com/matzehuels/stacktower/pkg/integrations/github"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// Server is the HTTP API server.
type Server struct {
	queue             queue.Queue
	backend           *storage.DistributedBackend
	pipeline          *pipeline.Service
	sessions          session.Store
	states            session.StateStore
	manifestPatterns  map[string]string
	logger            *infra.Logger
	quota             storage.QuotaConfig
	githubOAuth       github.OAuthConfig // Loaded once at startup
	frontendURL       string             // Frontend redirect URL for OAuth
	host              string
	port              int
	readTimeout       time.Duration
	writeTimeout      time.Duration
	requestTimeout    time.Duration // Per-request timeout
	maxRequestSize    int64
	allowOrigins      []string
	secureCookies     bool // Use Secure flag on cookies (should be true in production)
	trustProxyHeaders bool // Trust X-Forwarded-For and X-Real-IP headers (only behind trusted proxy)
	router            chi.Router
	http              *http.Server
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

// WithRequestTimeout sets the per-request timeout.
func WithRequestTimeout(d time.Duration) Option {
	return func(s *Server) { s.requestTimeout = d }
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
func WithLogger(logger *infra.Logger) Option {
	return func(s *Server) { s.logger = logger }
}

// WithQuota sets the rate limiting configuration.
func WithQuota(quota storage.QuotaConfig) Option {
	return func(s *Server) { s.quota = quota }
}

// WithGitHubOAuth sets the GitHub OAuth configuration.
func WithGitHubOAuth(cfg github.OAuthConfig) Option {
	return func(s *Server) { s.githubOAuth = cfg }
}

// WithFrontendURL sets the frontend URL for OAuth redirects.
func WithFrontendURL(url string) Option {
	return func(s *Server) { s.frontendURL = url }
}

// WithSecureCookies enables the Secure flag on cookies.
// Should be true in production (HTTPS), false for local development (HTTP).
func WithSecureCookies(secure bool) Option {
	return func(s *Server) { s.secureCookies = secure }
}

// WithTrustProxyHeaders enables trusting X-Forwarded-For and X-Real-IP headers.
// Only enable this when running behind a trusted load balancer/proxy.
// If disabled, client IP will be taken from RemoteAddr only.
func WithTrustProxyHeaders(trust bool) Option {
	return func(s *Server) { s.trustProxyHeaders = trust }
}

// New creates a new API server with the given options.
func New(q queue.Queue, backend *storage.DistributedBackend, opts ...Option) *Server {
	srv := &Server{
		queue:             q,
		backend:           backend,
		pipeline:          pipeline.NewService(backend),
		sessions:          session.NewMemoryStore(),
		states:            session.NewMemoryStateStore(),
		logger:            infra.DiscardLogger(),
		quota:             storage.DefaultQuotaConfig(),
		host:              "0.0.0.0",
		port:              8080,
		readTimeout:       30 * time.Second,
		writeTimeout:      30 * time.Second,
		requestTimeout:    25 * time.Second,        // Slightly less than write timeout
		maxRequestSize:    10 * 1024 * 1024,        // 10MB
		allowOrigins:      nil,                     // Must be explicitly set or derived from frontendURL
		secureCookies:     false,                   // Default false for dev; set true in production
		trustProxyHeaders: false,                   // Default false; only enable behind trusted proxy
		frontendURL:       "http://localhost:5173", // Default for dev
	}

	for _, opt := range opts {
		opt(srv)
	}

	// Default CORS to frontend URL if not explicitly set
	if len(srv.allowOrigins) == 0 && srv.frontendURL != "" {
		srv.allowOrigins = []string{srv.frontendURL}
	}

	srv.router = srv.setupRoutes()

	srv.http = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", srv.host, srv.port),
		Handler:      srv.router,
		ReadTimeout:  srv.readTimeout,
		WriteTimeout: srv.writeTimeout,
	}

	return srv
}

// API version for headers
const apiVersion = "1.0.0"

// setupRoutes creates the chi router with all routes and middleware.
func (s *Server) setupRoutes() chi.Router {
	r := chi.NewRouter()

	// Global middleware stack
	r.Use(s.recoverer)
	r.Use(requestID)
	r.Use(s.apiVersionHeader)
	r.Use(s.cors)
	r.Use(s.logging)

	// Health checks (no auth, no timeout)
	r.Get("/health", s.handleHealth)            // Liveness probe
	r.Get("/health/ready", s.handleHealthReady) // Readiness probe

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Apply request timeout to all API routes
		r.Use(timeout(s.requestTimeout))

		// Auth routes (no auth required to initiate)
		r.Route("/auth", func(r chi.Router) {
			r.Get("/github", s.handleGitHubAuth)
			r.Get("/github/callback", s.handleGitHubCallback)
			r.With(s.requireAuth).Get("/me", s.handleAuthMe)
			r.With(s.requireAuth).Post("/logout", s.handleAuthLogout)
		})

		// Pipeline routes (require auth + rate limiting)
		r.With(s.requireAuth, s.rateLimitFor(storage.OpTypeParse)).Post("/parse", s.handleParse)
		r.With(s.requireAuth, s.rateLimitFor(storage.OpTypeLayout)).Post("/layout", s.handleLayout)

		// Visualize is CPU-only, no external calls - use optional auth for rate limiting
		// Anonymous users get IP-based rate limiting, authenticated users get user-based
		r.With(s.optionalAuth, s.rateLimitVisualize).Post("/visualize", s.handleVisualize)

		// Render routes
		r.Route("/render", func(r chi.Router) {
			r.With(s.requireAuth, s.rateLimitFor(storage.OpTypeRender)).Post("/", s.handleRender)
			r.With(s.requireAuth).Get("/{renderID}", s.handleGetRender)
			r.With(s.requireAuth).Delete("/{renderID}", s.handleDeleteRender)
		})

		// History
		r.With(s.requireAuth).Get("/history", s.handleHistory)

		// Artifacts
		r.With(s.requireAuth).Get("/artifacts/{artifactID}", s.handleGetArtifact)

		// Jobs (require auth to prevent information leakage)
		r.Route("/jobs", func(r chi.Router) {
			r.Use(s.requireAuth)
			r.Get("/", s.handleJobsList)
			r.Get("/{jobID}", s.handleGetJob)
			r.Delete("/{jobID}", s.handleDeleteJob)
		})

		// GitHub repo routes (require auth)
		r.Route("/repos", func(r chi.Router) {
			r.Use(s.requireAuth)
			r.Get("/", s.handleRepos)
			r.Get("/{owner}/{repo}/manifests", s.handleRepoManifests)
			r.Post("/{owner}/{repo}/analyze", s.handleRepoAnalyze)
		})
	})

	return r
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.logger.Info("starting API server", "addr", s.http.Addr)
	return s.http.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down API server")
	return s.http.Shutdown(ctx)
}
