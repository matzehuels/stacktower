package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/matzehuels/stacktower/pkg/infra/queue"
	"github.com/matzehuels/stacktower/pkg/integrations/github"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// handleRepos lists the user's GitHub repositories.
// GET /api/v1/repos
// Auth enforced by requireAuth middleware.
func (s *Server) handleRepos(w http.ResponseWriter, r *http.Request) {
	sess := getSessionFromContext(r)

	// Use consolidated GitHub client
	client := github.NewContentClient(sess.AccessToken)
	repos, err := client.FetchUserRepos(r.Context())
	if err != nil {
		s.logger.Error("failed to fetch GitHub repos", "error", err, "user_id", sess.UserID(), "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "failed to fetch repositories")
		return
	}

	s.jsonResponse(w, http.StatusOK, repos)
}

// handleRepoManifests detects manifest files in a repository.
// GET /api/v1/repos/{owner}/{repo}/manifests
// Auth enforced by requireAuth middleware.
func (s *Server) handleRepoManifests(w http.ResponseWriter, r *http.Request) {
	sess := getSessionFromContext(r)

	owner := chi.URLParam(r, "owner")
	repo := chi.URLParam(r, "repo")

	if err := validateGitHubRepoParams(owner, repo); err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Use consolidated GitHub client
	client := github.NewContentClient(sess.AccessToken)
	manifests, err := client.DetectManifests(r.Context(), owner, repo, s.manifestPatterns)
	if err != nil {
		s.logger.Error("failed to detect manifests", "error", err, "owner", owner, "repo", repo, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "failed to detect manifests")
		return
	}

	s.jsonResponse(w, http.StatusOK, manifests)
}

// handleRepoAnalyze analyzes a repository's dependencies.
// POST /api/v1/repos/{owner}/{repo}/analyze
// Auth enforced by requireAuth middleware.
func (s *Server) handleRepoAnalyze(w http.ResponseWriter, r *http.Request) {
	sess := getSessionFromContext(r)

	owner := chi.URLParam(r, "owner")
	repo := chi.URLParam(r, "repo")

	if err := validateGitHubRepoParams(owner, repo); err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Parse request body using server's decodeJSON for consistent validation
	var req struct {
		ManifestPath string   `json:"manifest_path"`
		Formats      []string `json:"formats"`
	}
	if err := s.decodeJSON(w, r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, errInvalidRequest(err))
		return
	}

	if req.ManifestPath == "" {
		s.errorResponse(w, http.StatusBadRequest, errFieldRequired("manifest_path"))
		return
	}

	// Fetch manifest content using consolidated client
	client := github.NewContentClient(sess.AccessToken)
	content, err := client.FetchFileRaw(r.Context(), owner, repo, req.ManifestPath)
	if err != nil {
		s.logger.Error("failed to fetch manifest", "error", err, "owner", owner, "repo", repo, "path", req.ManifestPath, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "failed to fetch manifest")
		return
	}

	// Detect language from manifest filename
	language := github.DetectLanguageFromManifest(req.ManifestPath)
	if language == "" {
		s.errorResponse(w, http.StatusBadRequest, "unknown manifest type")
		return
	}

	// Create a render job with the manifest content
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
	payload := &pipeline.JobPayload{
		Options: pipeline.Options{
			UserID:           sess.UserID(),
			Language:         language,
			Manifest:         content,
			ManifestFilename: manifestFilename,
			Repo:             fmt.Sprintf("%s/%s", owner, repo),
			Formats:          formats,
			VizType:          "tower",
		},
	}

	payloadMap, err := payload.ToMap()
	if err != nil {
		s.logger.Error("failed to serialize payload", "error", err, "owner", owner, "repo", repo, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "failed to create job")
		return
	}

	job := &queue.Job{
		ID:        jobID,
		Type:      string(queue.TypeRender),
		Status:    queue.StatusPending,
		CreatedAt: time.Now(),
		Payload:   payloadMap,
	}

	if err := s.queue.Enqueue(r.Context(), job); err != nil {
		s.logger.Error("failed to enqueue analyze job", "error", err, "owner", owner, "repo", repo, "user_id", sess.UserID(), "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "failed to start analysis")
		return
	}

	s.jsonResponse(w, http.StatusAccepted, map[string]interface{}{
		"job_id":     jobID,
		"status":     "pending",
		"created_at": job.CreatedAt,
	})
}
