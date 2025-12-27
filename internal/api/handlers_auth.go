package api

import (
	"context"
	"net/http"
	"time"

	"github.com/matzehuels/stacktower/pkg/infra/session"
	"github.com/matzehuels/stacktower/pkg/integrations/github"
)

// handleGitHubAuth starts the GitHub OAuth flow.
// GET /api/v1/auth/github
func (s *Server) handleGitHubAuth(w http.ResponseWriter, r *http.Request) {
	if s.githubOAuth.ClientID == "" {
		s.errorResponse(w, http.StatusServiceUnavailable, "GitHub OAuth not configured")
		return
	}

	// Generate state token using the configured state store (memory or Redis)
	state, err := s.states.Generate(r.Context(), session.DefaultStateTTL)
	if err != nil {
		s.logger.Error("failed to generate OAuth state", "error", err, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "failed to generate state")
		return
	}

	// Use the pre-loaded OAuth config
	oauthClient := github.NewOAuthClient(s.githubOAuth)
	authURL := oauthClient.AuthorizationURL(state)

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// handleGitHubCallback handles the OAuth callback from GitHub.
// GET /api/v1/auth/github/callback
func (s *Server) handleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	// Verify state using the configured state store (memory or Redis)
	state := r.URL.Query().Get("state")
	valid, err := s.states.Validate(r.Context(), state)
	if err != nil {
		s.logger.Error("failed to validate OAuth state", "error", err, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "failed to validate state")
		return
	}
	if !valid {
		s.errorResponse(w, http.StatusBadRequest, "invalid or expired state")
		return
	}

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		errMsg := r.URL.Query().Get("error_description")
		if errMsg == "" {
			errMsg = "no authorization code received"
		}
		s.errorResponse(w, http.StatusBadRequest, errMsg)
		return
	}

	// Exchange code for access token using pre-loaded config
	oauthClient := github.NewOAuthClient(s.githubOAuth)
	token, err := oauthClient.ExchangeCode(code)
	if err != nil {
		s.logger.Error("failed to exchange OAuth code", "error", err, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "authentication failed")
		return
	}

	// Fetch user info using content client
	contentClient := github.NewContentClient(token.AccessToken)
	user, err := contentClient.FetchUser(r.Context())
	if err != nil {
		s.logger.Error("failed to fetch GitHub user", "error", err, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "failed to retrieve user information")
		return
	}

	// Create session
	sess, err := session.New(token.AccessToken, user, session.DefaultTTL)
	if err != nil {
		s.logger.Error("failed to create session", "error", err, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	// Store session
	if err := s.sessions.Set(r.Context(), sess); err != nil {
		s.logger.Error("failed to store session", "error", err, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "failed to store session")
		return
	}

	// Set session cookie and redirect to frontend
	http.SetCookie(w, s.sessionCookie(sess.ID, int(session.DefaultTTL.Seconds())))

	// Redirect to frontend with success
	http.Redirect(w, r, s.frontendURL+"?auth=success", http.StatusTemporaryRedirect)
}

// handleAuthMe returns the current user's info.
// GET /api/v1/auth/me
// Auth enforced by requireAuth middleware.
func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	sess := getSessionFromContext(r)
	// Defensive nil check - should never happen with requireAuth middleware,
	// but prevents a panic if middleware is misconfigured.
	if sess == nil {
		s.errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}
	s.jsonResponse(w, http.StatusOK, sess.User)
}

// handleAuthLogout logs out the current user.
// POST /api/v1/auth/logout
// Auth handled by middleware.
func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		// Use a background context with timeout for cleanup to ensure it completes
		// even if the client disconnects
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if delErr := s.sessions.Delete(cleanupCtx, cookie.Value); delErr != nil {
			s.logger.Warn("failed to delete session from store",
				"error", delErr,
				"session_id", cookie.Value,
				"request_id", getRequestID(r))
		}
	}

	http.SetCookie(w, s.sessionCookie("", -1))

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "logged out"})
}
