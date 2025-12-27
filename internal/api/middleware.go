package api

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/matzehuels/stacktower/pkg/infra/session"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
)

// Context keys for middleware values.
type contextKey string

const (
	userIDKey    contextKey = "userID"
	sessionKey   contextKey = "session"
	requestIDKey contextKey = "requestID"
)

// getUserID extracts the authenticated user ID from the request context.
// Returns empty string if not authenticated (should not happen if requireAuth middleware is used).
func getUserID(r *http.Request) string {
	if v, ok := r.Context().Value(userIDKey).(string); ok {
		return v
	}
	return ""
}

// getSessionFromContext extracts the session from the request context.
// Returns nil if not authenticated.
func getSessionFromContext(r *http.Request) *session.Session {
	if v, ok := r.Context().Value(sessionKey).(*session.Session); ok {
		return v
	}
	return nil
}

// requireAuth is middleware that enforces authentication.
// It extracts the session from the cookie and injects userID and session into context.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := s.getSession(r)
		if sess == nil {
			s.errorResponse(w, http.StatusUnauthorized, "authentication required")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, sess.UserID())
		ctx = context.WithValue(ctx, sessionKey, sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// optionalAuth is middleware that extracts auth if present but doesn't require it.
// Useful for endpoints that behave differently for authenticated vs anonymous users.
func (s *Server) optionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := s.getSession(r)
		if sess != nil {
			ctx := context.WithValue(r.Context(), userIDKey, sess.UserID())
			ctx = context.WithValue(ctx, sessionKey, sess)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

// rateLimitFor returns middleware that enforces rate limiting for a specific operation type.
func (s *Server) rateLimitFor(opType storage.OperationType) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := getUserID(r)
			if userID == "" {
				// No auth = no rate limit check (endpoint should use requireAuth first)
				next.ServeHTTP(w, r)
				return
			}

			if !s.checkRateLimit(w, r, userID, opType) {
				return // Response already sent by checkRateLimit
			}
			next.ServeHTTP(w, r)
		})
	}
}

// rateLimitVisualize provides rate limiting for the visualize endpoint.
// Uses user ID if authenticated, falls back to IP-based limiting for anonymous users.
func (s *Server) rateLimitVisualize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		if userID == "" {
			// Use IP address for anonymous rate limiting
			userID = "ip:" + getClientIP(r, s.trustProxyHeaders)
		}

		// Use layout operation type for visualize (CPU-bound, similar cost)
		if !s.checkRateLimit(w, r, userID, storage.OpTypeLayout) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

// getClientIP extracts the client IP address from the request.
// If trustProxy is true, checks X-Forwarded-For and X-Real-IP headers.
// Only trust proxy headers when running behind a load balancer you control.
func getClientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		// Check X-Forwarded-For first (may contain multiple IPs)
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// Take the first IP (original client)
			if idx := strings.Index(xff, ","); idx != -1 {
				return strings.TrimSpace(xff[:idx])
			}
			return strings.TrimSpace(xff)
		}

		// Check X-Real-IP
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return xri
		}
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// timeout returns middleware that wraps the handler with a timeout.
func timeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, d, `{"error":"request timeout"}`)
	}
}

// requestID adds a unique request ID to the context for tracing.
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = generateJobID() // Reuse UUID generator
		}
		w.Header().Set("X-Request-ID", reqID)
		ctx := context.WithValue(r.Context(), requestIDKey, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getRequestID extracts the request ID from the context.
func getRequestID(r *http.Request) string {
	if v, ok := r.Context().Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// recoverer is middleware that recovers from panics and returns a 500 error.
func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()
				s.logger.Error("panic recovered",
					"error", err,
					"path", r.URL.Path,
					"method", r.Method,
					"stack", string(stack),
					"request_id", getRequestID(r))
				s.errorResponse(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// cors handles CORS headers.
func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && s.isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// logging logs request details including request ID for tracing.
func (s *Server) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		s.logger.Debug("request",
			"request_id", getRequestID(r),
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.status,
			"duration", time.Since(start),
		)
	})
}

// Regex patterns for GitHub resource validation.
var (
	// GitHub usernames/orgs: 1-39 alphanumeric or hyphen, not starting with hyphen
	validGitHubOwner = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,38}$`)
	// GitHub repo names: 1-100 alphanumeric, hyphen, underscore, or dot
	validGitHubRepo = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,100}$`)
)

// validateGitHubOwner validates a GitHub username or organization name.
func validateGitHubOwner(owner string) error {
	if owner == "" {
		return errors.New("owner is required")
	}
	if !validGitHubOwner.MatchString(owner) {
		return errors.New("invalid owner format")
	}
	return nil
}

// validateGitHubRepo validates a GitHub repository name.
func validateGitHubRepo(repo string) error {
	if repo == "" {
		return errors.New("repo is required")
	}
	if !validGitHubRepo.MatchString(repo) {
		return errors.New("invalid repo format")
	}
	return nil
}

// validateGitHubRepoParams validates both owner and repo parameters.
func validateGitHubRepoParams(owner, repo string) error {
	if err := validateGitHubOwner(owner); err != nil {
		return err
	}
	return validateGitHubRepo(repo)
}

// responseWriter wraps http.ResponseWriter to capture status code.
// It also implements optional interfaces (Hijacker, Flusher) by delegation.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.wroteHeader = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// Hijack implements http.Hijacker for WebSocket support.
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("hijacking not supported")
}

// Flush implements http.Flusher for streaming responses.
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
