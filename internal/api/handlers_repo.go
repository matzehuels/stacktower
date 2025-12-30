package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/matzehuels/stacktower/pkg/infra/queue"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
	"github.com/matzehuels/stacktower/pkg/integrations/github"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// handleRepos lists the user's GitHub repositories.
// GET /api/v1/repos
// Auth enforced by requireAuth middleware.
//
// Note: This handler needs the full session (for AccessToken), not just userID.
// The requireAuth middleware guarantees a valid session is in context.
func (s *Server) handleRepos(w http.ResponseWriter, r *http.Request) {
	hctx := s.handlerContext()
	sess := getSessionFromContext(r)

	client := github.NewContentClient(sess.AccessToken)
	repos, err := client.FetchUserRepos(r.Context())
	if err != nil {
		hctx.Logger.Error("failed to fetch GitHub repos", "error", err, "user_id", sess.UserID(), "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, errMsgRepoFetchFailed)
		return
	}

	s.jsonResponse(w, http.StatusOK, repos)
}

// handleRepoManifests detects manifest files in a repository.
// GET /api/v1/repos/{owner}/{repo}/manifests
// Auth enforced by requireAuth middleware.
func (s *Server) handleRepoManifests(w http.ResponseWriter, r *http.Request) {
	hctx := s.handlerContext()
	sess := getSessionFromContext(r)

	owner := chi.URLParam(r, "owner")
	repo := chi.URLParam(r, "repo")

	if err := github.ValidateRepoRef(owner, repo); err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	client := github.NewContentClient(sess.AccessToken)
	manifests, err := client.DetectManifests(r.Context(), owner, repo, s.manifestPatterns)
	if err != nil {
		hctx.Logger.Error("failed to detect manifests", "error", err, "owner", owner, "repo", repo, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, errMsgManifestDetectFailed)
		return
	}

	s.jsonResponse(w, http.StatusOK, manifests)
}

// handleRepoAnalyze analyzes a repository's dependencies.
// POST /api/v1/repos/{owner}/{repo}/analyze
// Auth enforced by requireAuth middleware.
func (s *Server) handleRepoAnalyze(w http.ResponseWriter, r *http.Request) {
	hctx := s.handlerContext()
	sess := getSessionFromContext(r)

	owner := chi.URLParam(r, "owner")
	repo := chi.URLParam(r, "repo")

	if err := github.ValidateRepoRef(owner, repo); err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	var req RepoAnalyzeRequest
	if err := s.decodeJSON(w, r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, msgInvalidRequest(err))
		return
	}

	if req.ManifestPath == "" {
		s.errorResponse(w, http.StatusBadRequest, msgFieldRequired("manifest_path"))
		return
	}

	// Fetch manifest content
	client := github.NewContentClient(sess.AccessToken)
	content, err := client.FetchFileRaw(r.Context(), owner, repo, req.ManifestPath)
	if err != nil {
		hctx.Logger.Error("failed to fetch manifest", "error", err, "owner", owner, "repo", repo, "path", req.ManifestPath, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, errMsgManifestFetchFailed)
		return
	}

	// Detect language from manifest filename
	language := github.DetectLanguageFromManifest(req.ManifestPath)
	if language == "" {
		s.errorResponse(w, http.StatusBadRequest, errMsgUnknownManifest)
		return
	}

	// Build options
	formats := req.Formats
	if len(formats) == 0 {
		formats = []string{"svg", "png", "pdf", "json"}
	}

	manifestFilename := req.ManifestPath
	if idx := strings.LastIndex(req.ManifestPath, "/"); idx >= 0 {
		manifestFilename = req.ManifestPath[idx+1:]
	}

	opts := pipeline.Options{
		UserID:           sess.UserID(),
		Language:         language,
		Manifest:         content,
		ManifestFilename: manifestFilename,
		Repo:             fmt.Sprintf("%s/%s", owner, repo),
		Formats:          formats,
		VizType:          pipeline.VizTypeTower,
	}
	setWebAPIDefaults(&opts)

	if err := opts.ValidateAndSetDefaults(); err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	rctx := r.Context()

	// Fast path: check cached render
	manifestHash := storage.ManifestHash(opts.Manifest)
	renderID := storage.Keys.RenderDocumentID(sess.UserID(), language, manifestHash, opts.VizType)
	if stored, err := hctx.Backend.DocumentStore().GetRenderDocScoped(rctx, renderID, sess.UserID()); err == nil && stored != nil {
		resp := storageRenderToResponse(stored)
		resp.Cached = true
		s.jsonResponse(w, http.StatusOK, resp)
		return
	}

	// Queue job for async processing
	job := s.enqueueJob(hctx, w, r, queue.TypeRender, JobRequest{
		UserID:  sess.UserID(),
		TraceID: getRequestID(r),
		Options: opts,
	})
	if job == nil {
		return // Error response already sent
	}

	s.jsonResponse(w, http.StatusAccepted, JobResponse{
		JobID:     job.ID,
		Status:    "pending",
		CreatedAt: job.CreatedAt,
	})
}
