package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/matzehuels/stacktower/pkg/infra"
	"github.com/matzehuels/stacktower/pkg/integrations/github"
	"github.com/matzehuels/stacktower/pkg/queue"
	"github.com/matzehuels/stacktower/pkg/session"
)

// getGitHubOAuthConfig returns GitHub OAuth configuration from environment.
func getGitHubOAuthConfig() github.OAuthConfig {
	cfg := infra.LoadGitHubConfig()
	return github.OAuthConfig{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURI:  cfg.RedirectURI,
	}
}

// handleGitHubAuth starts the GitHub OAuth flow.
// GET /api/v1/auth/github
func (s *Server) handleGitHubAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := getGitHubOAuthConfig()
	if cfg.ClientID == "" {
		s.errorResponse(w, http.StatusServiceUnavailable, "GitHub OAuth not configured")
		return
	}

	// Generate state token using the configured state store (memory or Redis)
	state, err := s.states.Generate(r.Context(), session.DefaultStateTTL)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to generate state")
		return
	}

	// Use the consolidated OAuth client
	oauthClient := github.NewOAuthClient(cfg)
	authURL := oauthClient.AuthorizationURL(state)

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// handleGitHubCallback handles the OAuth callback from GitHub.
// GET /api/v1/auth/github/callback
func (s *Server) handleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify state using the configured state store (memory or Redis)
	state := r.URL.Query().Get("state")
	valid, err := s.states.Validate(r.Context(), state)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to validate state")
		return
	}
	if !valid {
		s.errorResponse(w, http.StatusBadRequest, "Invalid or expired state")
		return
	}

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		errMsg := r.URL.Query().Get("error_description")
		if errMsg == "" {
			errMsg = "No authorization code received"
		}
		s.errorResponse(w, http.StatusBadRequest, errMsg)
		return
	}

	// Exchange code for access token using consolidated client
	cfg := getGitHubOAuthConfig()
	oauthClient := github.NewOAuthClient(cfg)
	token, err := oauthClient.ExchangeCode(code)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to exchange code: "+err.Error())
		return
	}

	// Fetch user info using content client
	contentClient := github.NewContentClient(token.AccessToken)
	user, err := contentClient.FetchUser(r.Context())
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to fetch user: "+err.Error())
		return
	}

	// Create session
	sess, err := session.New(token.AccessToken, user, session.DefaultTTL)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	// Store session
	if err := s.sessions.Set(r.Context(), sess); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to store session")
		return
	}

	// Set session cookie and redirect to frontend
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sess.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(session.DefaultTTL.Seconds()),
	})

	// Redirect to frontend with success
	frontendURL := infra.LoadGitHubConfig().FrontendURL
	http.Redirect(w, r, frontendURL+"?auth=success", http.StatusTemporaryRedirect)
}

// handleAuthMe returns the current user's info.
// GET /api/v1/auth/me
func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sess := s.getSession(r)
	if sess == nil {
		s.errorResponse(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	s.jsonResponse(w, http.StatusOK, sess.User)
}

// handleAuthLogout logs out the current user.
// POST /api/v1/auth/logout
func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("session")
	if err == nil {
		s.sessions.Delete(r.Context(), cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "logged out"})
}

// handleRepos lists the user's GitHub repositories.
// GET /api/v1/repos
func (s *Server) handleRepos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sess := s.getSession(r)
	if sess == nil {
		s.errorResponse(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	// Use consolidated GitHub client
	client := github.NewContentClient(sess.AccessToken)
	repos, err := client.FetchUserRepos(r.Context())
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to fetch repos: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, repos)
}

// handleRepoManifests detects manifest files in a repository.
// GET /api/v1/repos/:owner/:repo/manifests
func (s *Server) handleRepoManifests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sess := s.getSession(r)
	if sess == nil {
		s.errorResponse(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	// Parse path: /api/v1/repos/:owner/:repo/manifests
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/repos/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[2] != "manifests" {
		s.errorResponse(w, http.StatusBadRequest, "Invalid path")
		return
	}
	owner, repo := parts[0], parts[1]

	// Use consolidated GitHub client
	client := github.NewContentClient(sess.AccessToken)
	manifests, err := client.DetectManifests(r.Context(), owner, repo, s.manifestPatterns)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to detect manifests: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, manifests)
}

// handleRepoAnalyze analyzes a repository's dependencies.
// POST /api/v1/repos/:owner/:repo/analyze
func (s *Server) handleRepoAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sess := s.getSession(r)
	if sess == nil {
		s.errorResponse(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	// Parse path: /api/v1/repos/:owner/:repo/analyze
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/repos/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[2] != "analyze" {
		s.errorResponse(w, http.StatusBadRequest, "Invalid path")
		return
	}
	owner, repo := parts[0], parts[1]

	// Parse request body
	var req struct {
		ManifestPath string   `json:"manifest_path"`
		Formats      []string `json:"formats"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.ManifestPath == "" {
		s.errorResponse(w, http.StatusBadRequest, "manifest_path is required")
		return
	}

	// Fetch manifest content using consolidated client
	client := github.NewContentClient(sess.AccessToken)
	content, err := client.FetchFileRaw(r.Context(), owner, repo, req.ManifestPath)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to fetch manifest: "+err.Error())
		return
	}

	// Detect language from manifest filename
	language := github.DetectLanguageFromManifest(req.ManifestPath)
	if language == "" {
		s.errorResponse(w, http.StatusBadRequest, "Unknown manifest type")
		return
	}

	// Create a visualize job with the manifest content
	formats := req.Formats
	if len(formats) == 0 {
		formats = []string{"svg", "png", "pdf"}
	}

	// Extract just the filename from the path
	manifestFilename := req.ManifestPath
	if idx := strings.LastIndex(req.ManifestPath, "/"); idx >= 0 {
		manifestFilename = req.ManifestPath[idx+1:]
	}

	jobID := generateJobID()
	job := &queue.Job{
		ID:        jobID,
		Type:      "visualize",
		Status:    queue.StatusPending,
		CreatedAt: time.Now(),
		Payload: map[string]interface{}{
			"language":          language,
			"manifest":          content, // Pass manifest content directly
			"manifest_filename": manifestFilename,
			"package":           fmt.Sprintf("%s/%s", owner, repo), // Use repo as package name
			"formats":           formats,
			"viz_type":          "tower",
		},
	}

	if err := s.queue.Enqueue(r.Context(), job); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "Failed to enqueue job: "+err.Error())
		return
	}

	s.jsonResponse(w, http.StatusAccepted, map[string]interface{}{
		"job_id":     jobID,
		"status":     "pending",
		"created_at": job.CreatedAt,
	})
}

// getSession retrieves the session from the request cookie.
func (s *Server) getSession(r *http.Request) *session.Session {
	cookie, err := r.Cookie("session")
	if err != nil {
		return nil
	}

	sess, err := s.sessions.Get(r.Context(), cookie.Value)
	if err != nil || sess == nil {
		return nil
	}

	return sess
}

// handleRepoRoutes routes requests to /api/v1/repos/:owner/:repo/*
func (s *Server) handleRepoRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/repos/")
	parts := strings.Split(path, "/")

	if len(parts) < 3 {
		s.errorResponse(w, http.StatusBadRequest, "Invalid repo path")
		return
	}

	switch parts[2] {
	case "manifests":
		s.handleRepoManifests(w, r)
	case "analyze":
		s.handleRepoAnalyze(w, r)
	default:
		s.errorResponse(w, http.StatusNotFound, "Unknown endpoint")
	}
}
