package pipeline

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/matzehuels/stacktower/pkg/infra/storage"
	pkgio "github.com/matzehuels/stacktower/pkg/io"
)

// RenderOutput contains the data needed to store a render result.
// This separates computation (pipeline) from storage (service).
type RenderOutput struct {
	GraphID   string
	GraphHash string
	Result    *Result
	Options   Options
	CacheHit  bool
	TraceID   string
	Repo      string // GitHub repository (owner/repo)
}

// RenderRecord is the stored result of a render operation.
// Uses storage.RenderSource to avoid duplicating types across layers.
// API handlers transform this to API-specific response types.
//
// Note: GraphID and GraphHash are NOT included here because the caller
// already has them (they're inputs to StoreRenderResult). This avoids
// the pass-through anti-pattern.
//
// Note: Nebraska rankings are embedded in the Layout JSON (stored in Render.Layout),
// not tracked separately. Use storage.ParseNebraskaFromLayout to extract them.
type RenderRecord struct {
	RenderID   string
	NodeCount  int
	EdgeCount  int
	VizType    string
	Artifacts  map[string]string // format -> artifact URL
	Source     storage.RenderSource
	LayoutData []byte // For extracting Nebraska rankings
	CacheHit   bool
}

// StoreRenderResult stores a completed render in the document store.
// This is the single point of truth for render storage logic.
//
// For public packages:
//   - Creates/updates a canonical render (user_id = "") shared by all users
//   - Adds the package to the user's collection
//
// For private repos (manifests):
//   - Creates a user-scoped render (user_id set)
//   - No collection entry (private repos shown via ListPrivateRenders)
//
// It handles:
//   - Canonical vs user-scoped render storage
//   - Artifact storage (SVG, PNG, PDF)
//   - User collection management
//   - Cache deduplication
func (s *Service) StoreRenderResult(ctx context.Context, out RenderOutput) (*RenderRecord, error) {
	if s.cacheBackend == nil {
		return nil, fmt.Errorf("document store not available")
	}

	opts := out.Options
	userID := opts.UserID

	// Build source info
	source := storage.RenderSource{
		Type:     "package",
		Language: opts.Language,
		Package:  opts.Package,
		Repo:     out.Repo,
	}
	if opts.Manifest != "" {
		source.Type = "manifest"
		source.ManifestFilename = opts.ManifestFilename
	}

	// For public packages, use canonical (shared) renders
	// For private repos (manifests), use user-scoped renders
	isPublicPackage := opts.Manifest == ""

	if isPublicPackage {
		return s.storeCanonicalRender(ctx, out, source, userID)
	}
	return s.storePrivateRender(ctx, out, source, userID)
}

// storeCanonicalRender stores a public package render as canonical (shared).
func (s *Service) storeCanonicalRender(ctx context.Context, out RenderOutput, source storage.RenderSource, userID string) (*RenderRecord, error) {
	docstore := s.cacheBackend.DocumentStore()
	opts := out.Options

	// Check for existing canonical render
	existing, _ := docstore.GetCanonicalRender(ctx, opts.Language, opts.Package, opts.VizType)
	if existing != nil && storage.RenderHasFormats(existing, opts.Formats) {
		// Canonical exists with all formats - add to user's collection and return
		_ = docstore.SaveToLibrary(ctx, userID, opts.Language, opts.Package)
		return &RenderRecord{
			RenderID:   existing.ID,
			NodeCount:  existing.NodeCount,
			EdgeCount:  existing.EdgeCount,
			VizType:    opts.VizType,
			Artifacts:  storage.BuildArtifactURLs(existing.Artifacts, out.GraphID),
			Source:     existing.Source,
			LayoutData: documentToJSON(existing.Layout),
			CacheHit:   true,
		}, nil
	}

	// Generate canonical render ID (no user prefix)
	renderID := storage.Keys.RenderDocumentID("", opts.Language, opts.Package, opts.VizType)

	// Build canonical render document (user_id = "")
	render := &storage.Render{
		ID:        renderID,
		UserID:    "", // Canonical - shared by all users
		GraphID:   out.GraphID,
		GraphHash: out.GraphHash,
		Layout:    jsonToDocument(out.Result.LayoutData), // Convert to BSON document (Nebraska rankings embedded)
		NodeCount: out.Result.Graph.NodeCount(),
		EdgeCount: out.Result.Graph.EdgeCount(),
		Source:    source,
		LayoutOptions: storage.LayoutOptions{
			VizType: opts.VizType,
		},
	}

	// Preserve existing artifacts (merge new with old)
	if existing != nil {
		render.Artifacts = existing.Artifacts
	}

	// Store new artifacts (with empty userID for global access)
	for format, data := range out.Result.Artifacts {
		filename := fmt.Sprintf("%s.%s", opts.VizType, format)
		artifactID, err := docstore.StoreArtifact(ctx, renderID, filename, data, "") // Empty userID = global
		if err != nil {
			return nil, fmt.Errorf("store artifact %s: %w", format, err)
		}
		switch format {
		case "svg":
			render.Artifacts.SVG = artifactID
		case "png":
			render.Artifacts.PNG = artifactID
		case "pdf":
			render.Artifacts.PDF = artifactID
		case "json":
			render.Artifacts.JSON = artifactID
		}
	}

	// Upsert canonical render document
	if err := docstore.UpsertRenderDoc(ctx, render); err != nil {
		return nil, fmt.Errorf("upsert render: %w", err)
	}

	// Add to user's collection
	_ = docstore.SaveToLibrary(ctx, userID, opts.Language, opts.Package)

	return &RenderRecord{
		RenderID:   render.ID,
		NodeCount:  render.NodeCount,
		EdgeCount:  render.EdgeCount,
		VizType:    opts.VizType,
		Artifacts:  storage.BuildArtifactURLs(render.Artifacts, out.GraphID),
		Source:     source,
		LayoutData: documentToJSON(render.Layout),
		CacheHit:   out.CacheHit,
	}, nil
}

// storePrivateRender stores a private repo render (user-scoped).
func (s *Service) storePrivateRender(ctx context.Context, out RenderOutput, source storage.RenderSource, userID string) (*RenderRecord, error) {
	docstore := s.cacheBackend.DocumentStore()
	opts := out.Options

	// Determine manifest identifier for render ID
	manifestHash := storage.ManifestHash(opts.Manifest)

	// Generate user-scoped render ID
	renderID := storage.Keys.RenderDocumentID(userID, opts.Language, manifestHash, opts.VizType)

	// Check existing render - return early if we have all requested artifacts
	existing, _ := docstore.GetRenderDoc(ctx, renderID)
	if existing != nil && storage.RenderHasFormats(existing, opts.Formats) {
		return &RenderRecord{
			RenderID:   existing.ID,
			NodeCount:  existing.NodeCount,
			EdgeCount:  existing.EdgeCount,
			VizType:    opts.VizType,
			Artifacts:  storage.BuildArtifactURLs(existing.Artifacts, out.GraphID),
			Source:     existing.Source,
			LayoutData: documentToJSON(existing.Layout),
			CacheHit:   true,
		}, nil
	}

	// Build user-scoped render document
	render := &storage.Render{
		ID:        renderID,
		UserID:    userID,
		GraphID:   out.GraphID,
		GraphHash: out.GraphHash,
		Layout:    jsonToDocument(out.Result.LayoutData), // Convert to BSON document (Nebraska rankings embedded)
		NodeCount: out.Result.Graph.NodeCount(),
		EdgeCount: out.Result.Graph.EdgeCount(),
		Source:    source,
		LayoutOptions: storage.LayoutOptions{
			VizType: opts.VizType,
		},
	}

	// Preserve existing artifacts (merge new with old)
	if existing != nil {
		render.Artifacts = existing.Artifacts
	}

	// Store new artifacts (with userID for scoped access)
	for format, data := range out.Result.Artifacts {
		filename := fmt.Sprintf("%s.%s", opts.VizType, format)
		artifactID, err := docstore.StoreArtifact(ctx, renderID, filename, data, userID)
		if err != nil {
			return nil, fmt.Errorf("store artifact %s: %w", format, err)
		}
		switch format {
		case "svg":
			render.Artifacts.SVG = artifactID
		case "png":
			render.Artifacts.PNG = artifactID
		case "pdf":
			render.Artifacts.PDF = artifactID
		case "json":
			render.Artifacts.JSON = artifactID
		}
	}

	// Upsert render document
	if err := docstore.UpsertRenderDoc(ctx, render); err != nil {
		return nil, fmt.Errorf("upsert render: %w", err)
	}

	// Note: Private renders are NOT added to collection - they're queried via ListPrivateRenders

	return &RenderRecord{
		RenderID:   render.ID,
		NodeCount:  render.NodeCount,
		EdgeCount:  render.EdgeCount,
		VizType:    opts.VizType,
		Artifacts:  storage.BuildArtifactURLs(render.Artifacts, out.GraphID),
		Source:     source,
		LayoutData: documentToJSON(render.Layout),
		CacheHit:   out.CacheHit,
	}, nil
}

// GetCachedRenderRecord checks if a cached render exists for the given options.
// For public packages, checks for canonical (shared) renders.
// For private repos (manifests), checks for user-scoped renders.
// Returns nil if no cached render is found.
func (s *Service) GetCachedRenderRecord(ctx context.Context, opts Options) *RenderRecord {
	if s.cacheBackend == nil {
		return nil
	}
	docstore := s.cacheBackend.DocumentStore()

	isPublicPackage := opts.Manifest == ""

	var render *storage.Render
	var err error

	if isPublicPackage {
		// For public packages, check canonical render
		render, err = docstore.GetCanonicalRender(ctx, opts.Language, opts.Package, opts.VizType)
		if err != nil || render == nil {
			return nil
		}
		// Add to user's library
		_ = docstore.SaveToLibrary(ctx, opts.UserID, opts.Language, opts.Package)
	} else {
		// For private repos, check user-scoped render
		manifestHash := storage.ManifestHash(opts.Manifest)
		renderID := storage.Keys.RenderDocumentID(opts.UserID, opts.Language, manifestHash, opts.VizType)
		render, err = docstore.GetRenderDocScoped(ctx, renderID, opts.UserID)
		if err != nil || render == nil {
			return nil
		}
	}

	return &RenderRecord{
		RenderID:   render.ID,
		NodeCount:  render.NodeCount,
		EdgeCount:  render.EdgeCount,
		VizType:    render.LayoutOptions.VizType,
		Artifacts:  storage.BuildArtifactURLs(render.Artifacts, render.GraphID),
		Source:     render.Source,
		LayoutData: documentToJSON(render.Layout),
		CacheHit:   true,
	}
}

// GetGraphDocumentID returns the graph document ID from the cache entry.
// Returns empty string if not found.
func (s *Service) GetGraphDocumentID(ctx context.Context, cacheKey string) string {
	if s.cacheBackend == nil {
		return ""
	}
	if entry, _ := s.cacheBackend.Index().GetGraphEntry(ctx, cacheKey); entry != nil {
		return entry.DocumentID
	}
	return ""
}

// ComputeGraphHash computes a content hash for a Result's graph.
func ComputeGraphHash(result *Result) string {
	if result.GraphHash != "" {
		return result.GraphHash
	}
	graphData, _ := pkgio.SerializeDAG(result.Graph)
	return storage.Hash(graphData)
}

// jsonToDocument converts JSON bytes to an interface{} for BSON document storage.
// Returns nil if input is empty or invalid.
func jsonToDocument(jsonData []byte) interface{} {
	if len(jsonData) == 0 {
		return nil
	}
	var doc interface{}
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		return nil
	}
	return doc
}

// documentToJSON converts a BSON document (interface{}) back to JSON bytes.
// Handles both legacy []byte format and new document format.
// Returns nil if input is nil or invalid.
func documentToJSON(doc interface{}) []byte {
	if doc == nil {
		return nil
	}

	// Handle legacy []byte format
	if bytes, ok := doc.([]byte); ok {
		return bytes
	}

	// Convert document to JSON
	jsonData, err := json.Marshal(doc)
	if err != nil {
		return nil
	}
	return jsonData
}
