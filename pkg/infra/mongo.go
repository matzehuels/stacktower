package infra

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/matzehuels/stacktower/pkg/infra/storage"
)

// =============================================================================
// MongoDB Client
// =============================================================================

// Mongo is a unified MongoDB client that provides document storage
// and GridFS operations.
type Mongo struct {
	client   *mongo.Client
	database *mongo.Database
	config   MongoConfig
	store    *mongoStore
}

// NewMongo creates a new unified MongoDB client.
func NewMongo(ctx context.Context, cfg MongoConfig) (*Mongo, error) {
	if cfg.Database == "" {
		cfg.Database = "stacktower"
	}
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 10 * time.Second
	}
	if cfg.ServerSelectionTimeout == 0 {
		cfg.ServerSelectionTimeout = 5 * time.Second
	}

	clientOpts := options.Client().
		ApplyURI(cfg.URI).
		SetConnectTimeout(cfg.ConnectTimeout).
		SetServerSelectionTimeout(cfg.ServerSelectionTimeout)

	connectCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	client, err := mongo.Connect(connectCtx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("connect to mongodb: %w", err)
	}

	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		client.Disconnect(ctx)
		return nil, fmt.Errorf("ping mongodb: %w", err)
	}

	return &Mongo{
		client:   client,
		database: client.Database(cfg.Database),
		config:   cfg,
	}, nil
}

// DocumentStore returns a storage.DocumentStore for document operations.
func (m *Mongo) DocumentStore() storage.DocumentStore {
	if m.store == nil {
		m.store = newMongoStore(m.database)
	}
	return m.store
}

// Database returns the underlying mongo.Database.
func (m *Mongo) Database() *mongo.Database {
	return m.database
}

// Close closes the MongoDB connection.
func (m *Mongo) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.client.Disconnect(ctx)
}

// Info returns connection info for logging.
func (m *Mongo) Info() string {
	return fmt.Sprintf("mongodb (%s/%s)", m.config.URI, m.config.Database)
}

// =============================================================================
// Store Implementation
// =============================================================================

type mongoStore struct {
	db        *mongo.Database
	graphs    *mongo.Collection
	renders   *mongo.Collection
	libraries *mongo.Collection
	gridfs    *gridfs.Bucket
}

func newMongoStore(db *mongo.Database) *mongoStore {
	graphs := db.Collection("graphs")
	renders := db.Collection("renders")
	libraries := db.Collection("libraries")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Indexes
	graphs.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "scope", Value: 1}, {Key: "language", Value: 1}, {Key: "package", Value: 1}}},
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
		{Keys: bson.D{{Key: "content_hash", Value: 1}}},
	})
	renders.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "source.language", Value: 1}, {Key: "source.package", Value: 1}}},
		{Keys: bson.D{
			{Key: "source.language", Value: 1},
			{Key: "source.package", Value: 1},
			{Key: "layout_options.viz_type", Value: 1},
		}},
	})

	// Library indexes
	libraries.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "saved_at", Value: -1}}},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "language", Value: 1}, {Key: "package", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{Keys: bson.D{{Key: "language", Value: 1}, {Key: "package", Value: 1}}},
	})

	bucket, _ := gridfs.NewBucket(db, options.GridFSBucket().SetName("render_artifacts"))

	return &mongoStore{db: db, graphs: graphs, renders: renders, libraries: libraries, gridfs: bucket}
}

// Graph operations

func (s *mongoStore) GetGraphDoc(ctx context.Context, id string) (*storage.Graph, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid graph ID: %w", err)
	}
	var graph storage.Graph
	err = s.graphs.FindOne(ctx, bson.M{"_id": objID}).Decode(&graph)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find graph: %w", err)
	}
	return &graph, nil
}

func (s *mongoStore) StoreGraphDoc(ctx context.Context, graph *storage.Graph) error {
	now := time.Now()
	graph.UpdatedAt = now
	if graph.ID == "" {
		graph.ID = primitive.NewObjectID().Hex()
		graph.CreatedAt = now
	}

	objID, _ := primitive.ObjectIDFromHex(graph.ID)
	doc := bson.M{
		"_id": objID, "scope": graph.Scope, "user_id": graph.UserID,
		"language": graph.Language, "package": graph.Package, "repo": graph.Repo,
		"options": graph.Options, "node_count": graph.NodeCount, "edge_count": graph.EdgeCount,
		"content_hash": graph.ContentHash, "data": graph.Data,
		"created_at": graph.CreatedAt, "updated_at": graph.UpdatedAt,
	}
	_, err := s.graphs.ReplaceOne(ctx, bson.M{"_id": objID}, doc, options.Replace().SetUpsert(true))
	return err
}

// Render operations

func (s *mongoStore) GetRenderDoc(ctx context.Context, id string) (*storage.Render, error) {
	// Render.ID is stored as a string, so query by string _id
	var render storage.Render
	err := s.renders.FindOne(ctx, bson.M{"_id": id}).Decode(&render)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find render: %w", err)
	}
	s.renders.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"accessed_at": time.Now()}})
	return &render, nil
}

func (s *mongoStore) StoreRenderDoc(ctx context.Context, render *storage.Render) error {
	now := time.Now()
	if render.ID == "" {
		render.ID = primitive.NewObjectID().Hex()
	}
	render.CreatedAt = now
	render.AccessedAt = now
	_, err := s.renders.InsertOne(ctx, render)
	return err
}

func (s *mongoStore) UpsertRenderDoc(ctx context.Context, render *storage.Render) error {
	now := time.Now()
	if render.ID == "" {
		render.ID = primitive.NewObjectID().Hex()
	}
	render.AccessedAt = now

	// Use upsert: if exists, update; otherwise insert with CreatedAt
	filter := bson.M{"_id": render.ID}
	update := bson.M{
		"$set": bson.M{
			"user_id":        render.UserID,
			"graph_id":       render.GraphID,
			"graph_hash":     render.GraphHash,
			"layout_options": render.LayoutOptions,
			"render_options": render.RenderOptions,
			"layout":         render.Layout,
			"artifacts":      render.Artifacts,
			"node_count":     render.NodeCount,
			"edge_count":     render.EdgeCount,
			"source":         render.Source,
			"accessed_at":    render.AccessedAt,
		},
		"$setOnInsert": bson.M{
			"created_at": now,
		},
	}
	opts := options.Update().SetUpsert(true)
	_, err := s.renders.UpdateOne(ctx, filter, update, opts)
	return err
}

func (s *mongoStore) DeleteRenderDoc(ctx context.Context, id string) error {
	// Render.ID is stored as a string, so query by string _id
	var render storage.Render
	err := s.renders.FindOne(ctx, bson.M{"_id": id}).Decode(&render)
	if err != nil {
		return fmt.Errorf("render not found: %w", err)
	}

	// Delete artifacts from GridFS
	for _, aid := range []string{render.Artifacts.SVG, render.Artifacts.PNG, render.Artifacts.PDF} {
		if aid != "" {
			if oid, err := primitive.ObjectIDFromHex(aid); err == nil {
				s.gridfs.Delete(oid)
			}
		}
	}

	_, err = s.renders.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// Artifact operations

func (s *mongoStore) StoreArtifact(ctx context.Context, renderID, filename string, data []byte, userID string) (string, error) {
	fullPath := fmt.Sprintf("%s/%s", renderID, filename)
	meta := bson.M{
		"render_id":  renderID,
		"filename":   filename,
		"created_at": time.Now(),
		// ALWAYS store user_id in metadata for authorization checks:
		// - user_id="" (empty) → shared/global artifact, accessible to all
		// - user_id="github:123" → user-owned artifact, restricted access
		// GetArtifactScoped relies on this metadata being present.
		"user_id": userID,
	}
	uploadOpts := options.GridFSUpload().SetMetadata(meta)
	objID, err := s.gridfs.UploadFromStream(fullPath, bytes.NewReader(data), uploadOpts)
	if err != nil {
		return "", fmt.Errorf("upload artifact: %w", err)
	}
	return objID.Hex(), nil
}

func (s *mongoStore) GetArtifact(ctx context.Context, artifactID string) ([]byte, error) {
	objID, err := primitive.ObjectIDFromHex(artifactID)
	if err != nil {
		return nil, fmt.Errorf("invalid artifact ID: %w", err)
	}
	stream, err := s.gridfs.OpenDownloadStream(objID)
	if err != nil {
		return nil, fmt.Errorf("open artifact stream: %w", err)
	}
	defer stream.Close()
	return io.ReadAll(stream)
}

// Scoped operations (with authorization)

func (s *mongoStore) GetGraphDocScoped(ctx context.Context, id string, userID string) (*storage.Graph, error) {
	graph, err := s.GetGraphDoc(ctx, id)
	if err != nil || graph == nil {
		return graph, err
	}

	// Check authorization for user-scoped graphs
	if graph.Scope == storage.ScopeUser && graph.UserID != userID {
		return nil, storage.ErrAccessDenied
	}

	return graph, nil
}

func (s *mongoStore) GetRenderDocScoped(ctx context.Context, id string, userID string) (*storage.Render, error) {
	render, err := s.GetRenderDoc(ctx, id)
	if err != nil || render == nil {
		return render, err
	}

	// Renders are always user-scoped
	if render.UserID != userID {
		return nil, storage.ErrAccessDenied
	}

	return render, nil
}

func (s *mongoStore) DeleteRenderDocScoped(ctx context.Context, id string, userID string) error {
	// First check ownership
	render, err := s.GetRenderDocScoped(ctx, id, userID)
	if err != nil {
		return err
	}
	if render == nil {
		return nil
	}

	// Delete via normal method
	return s.DeleteRenderDoc(ctx, id)
}

func (s *mongoStore) GetArtifactScoped(ctx context.Context, artifactID string, userID string) ([]byte, error) {
	objID, err := primitive.ObjectIDFromHex(artifactID)
	if err != nil {
		return nil, fmt.Errorf("invalid artifact ID: %w", err)
	}

	// Check GridFS metadata for ownership.
	// StoreArtifact always sets user_id in metadata (empty for global, non-empty for user-owned).
	filesColl := s.db.Collection("render_artifacts.files")
	var fileMeta struct {
		Metadata struct {
			UserID string `bson:"user_id"`
		} `bson:"metadata"`
	}
	err = filesColl.FindOne(ctx, bson.M{"_id": objID}).Decode(&fileMeta)
	if err == mongo.ErrNoDocuments {
		return nil, fmt.Errorf("artifact %s: %w", artifactID, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("find artifact metadata: %w", err)
	}

	// Authorization: empty user_id = global (allow all), otherwise must match
	storedUserID := fileMeta.Metadata.UserID
	if storedUserID != "" && storedUserID != userID {
		return nil, storage.ErrAccessDenied
	}

	return s.GetArtifact(ctx, artifactID)
}

func (s *mongoStore) Ping(ctx context.Context) error {
	return s.db.Client().Ping(ctx, nil)
}

func (s *mongoStore) CountUniqueTowers(ctx context.Context) (int64, error) {
	// Count distinct (language, package) combinations from renders.
	// Different viz types for the same package count as 1 tower.
	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"language": "$source.language",
				"package":  "$source.package",
			},
		}}},
		{{Key: "$count", Value: "total"}},
	}
	cursor, err := s.renders.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return 0, err
	}
	if len(results) == 0 {
		return 0, nil
	}
	if total, ok := results[0]["total"].(int32); ok {
		return int64(total), nil
	}
	if total, ok := results[0]["total"].(int64); ok {
		return total, nil
	}
	return 0, nil
}

func (s *mongoStore) CountUniqueUsers(ctx context.Context) (int64, error) {
	// Use distinct to count unique user_ids
	values, err := s.renders.Distinct(ctx, "user_id", bson.M{})
	if err != nil {
		return 0, err
	}
	return int64(len(values)), nil
}

func (s *mongoStore) CountUniqueDependencies(ctx context.Context) (int64, error) {
	// Sum node counts from unique towers (distinct language+package).
	// This gives the total unique dependencies analyzed without double-counting
	// the same package rendered with different viz types.
	pipeline := mongo.Pipeline{
		// Group by (language, package) to get unique towers and their max node_count
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"language": "$source.language",
				"package":  "$source.package",
			},
			"node_count": bson.M{"$max": "$node_count"}, // Take max in case of variations
		}}},
		// Sum all unique tower node counts
		{{Key: "$group", Value: bson.M{
			"_id":   nil,
			"total": bson.M{"$sum": "$node_count"},
		}}},
	}
	cursor, err := s.renders.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return 0, err
	}
	if len(results) == 0 {
		return 0, nil
	}
	if total, ok := results[0]["total"].(int32); ok {
		return int64(total), nil
	}
	if total, ok := results[0]["total"].(int64); ok {
		return total, nil
	}
	return 0, nil
}

func (s *mongoStore) ListPackageSuggestions(ctx context.Context, language string, query string, limit int) ([]storage.PackageSuggestion, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	// Build aggregation pipeline on user_libraries to find popular packages
	// and rank them by how many users have them in their library
	matchStage := bson.M{}
	if language != "" {
		matchStage["language"] = language
	}
	if query != "" {
		// Case-insensitive prefix match
		matchStage["package"] = bson.M{"$regex": "^" + query, "$options": "i"}
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: matchStage}},
		// Group by (language, package) and count users
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"language": "$language",
				"package":  "$package",
			},
			"popularity": bson.M{"$sum": 1},
		}}},
		// Sort by popularity (most bookmarked first)
		{{Key: "$sort", Value: bson.M{"popularity": -1}}},
		// Limit results
		{{Key: "$limit", Value: limit}},
		// Project to final shape
		{{Key: "$project", Value: bson.M{
			"_id":        0,
			"language":   "$_id.language",
			"package":    "$_id.package",
			"popularity": 1,
		}}},
	}

	cursor, err := s.libraries.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []storage.PackageSuggestion
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// =============================================================================
// Canonical Renders & User Collections
// =============================================================================

// GetCanonicalRender looks up a canonical (shared) render for a public package.
func (s *mongoStore) GetCanonicalRender(ctx context.Context, language, pkg, vizType string) (*storage.Render, error) {
	filter := bson.M{
		"user_id":                 "",
		"source.type":             "package",
		"source.language":         language,
		"source.package":          pkg,
		"layout_options.viz_type": vizType,
	}

	var render storage.Render
	err := s.renders.FindOne(ctx, filter).Decode(&render)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find canonical render: %w", err)
	}
	return &render, nil
}

// SaveToLibrary adds a package to a user's library.
func (s *mongoStore) SaveToLibrary(ctx context.Context, userID, language, pkg string) error {
	filter := bson.M{
		"user_id":  userID,
		"language": language,
		"package":  pkg,
	}
	update := bson.M{
		"$setOnInsert": bson.M{
			"_id":      primitive.NewObjectID().Hex(),
			"user_id":  userID,
			"language": language,
			"package":  pkg,
			"saved_at": time.Now(),
		},
	}
	opts := options.Update().SetUpsert(true)
	_, err := s.libraries.UpdateOne(ctx, filter, update, opts)
	return err
}

// RemoveFromLibrary removes a package from a user's library.
func (s *mongoStore) RemoveFromLibrary(ctx context.Context, userID, language, pkg string) error {
	filter := bson.M{
		"user_id":  userID,
		"language": language,
		"package":  pkg,
	}
	_, err := s.libraries.DeleteOne(ctx, filter)
	return err
}

// IsInLibrary checks if a package is in a user's library.
func (s *mongoStore) IsInLibrary(ctx context.Context, userID, language, pkg string) (bool, error) {
	filter := bson.M{
		"user_id":  userID,
		"language": language,
		"package":  pkg,
	}
	count, err := s.libraries.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListLibrary returns a user's saved packages.
func (s *mongoStore) ListLibrary(ctx context.Context, userID string, limit, offset int) ([]storage.LibraryEntry, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	filter := bson.M{"user_id": userID}
	total, err := s.libraries.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "saved_at", Value: -1}}).
		SetSkip(int64(offset)).
		SetLimit(int64(limit))

	cursor, err := s.libraries.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var entries []storage.LibraryEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}

// ListPrivateRenders returns a user's private repo renders.
func (s *mongoStore) ListPrivateRenders(ctx context.Context, userID string, limit, offset int) ([]*storage.Render, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// Private renders: user_id is set and source.type is "manifest"
	filter := bson.M{
		"user_id":     userID,
		"source.type": "manifest",
	}

	// Get total count
	total, err := s.renders.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Find with pagination
	opts := options.Find().
		SetSort(bson.D{{Key: "accessed_at", Value: -1}}).
		SetSkip(int64(offset)).
		SetLimit(int64(limit))

	cursor, err := s.renders.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var renders []*storage.Render
	if err := cursor.All(ctx, &renders); err != nil {
		return nil, 0, err
	}

	return renders, total, nil
}

// =============================================================================
// Explore
// =============================================================================

// ListExplore returns public towers for the explore page.
// sortBy: "popular" (default) or "recent"
func (s *mongoStore) ListExplore(ctx context.Context, language, sortBy string, limit, offset int) ([]storage.ExploreEntry, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// Build match stage - only canonical package-based renders (user_id = "")
	matchStage := bson.M{
		"source.type": "package",
		"user_id":     "",
	}
	if language != "" {
		matchStage["source.language"] = language
	}

	// Determine sort order based on sortBy parameter
	var sortStage bson.D
	if sortBy == "recent" {
		sortStage = bson.D{{Key: "created_at", Value: -1}}
	} else {
		// Default: sort by popularity first, then by most recent
		sortStage = bson.D{
			{Key: "popularity_count", Value: -1},
			{Key: "created_at", Value: -1},
		}
	}

	// Aggregation pipeline that groups by (language, package)
	pipeline := mongo.Pipeline{
		// Match canonical package renders only
		{{Key: "$match", Value: matchStage}},
		// Group by (language, package)
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"language": "$source.language",
				"package":  "$source.package",
			},
			"source":     bson.M{"$first": "$source"},
			"node_count": bson.M{"$max": "$node_count"},
			"edge_count": bson.M{"$max": "$edge_count"},
			"created_at": bson.M{"$max": "$created_at"},
			"viz_types": bson.M{"$push": bson.M{
				"viz_type":     "$layout_options.viz_type",
				"render_id":    bson.M{"$toString": "$_id"},
				"graph_id":     "$graph_id",
				"artifact_svg": "$artifacts.svg",
				"artifact_png": "$artifacts.png",
				"artifact_pdf": "$artifacts.pdf",
			}},
		}}},
		// Lookup popularity count from libraries
		{{Key: "$lookup", Value: bson.M{
			"from": "libraries",
			"let":  bson.M{"lang": "$_id.language", "pkg": "$_id.package"},
			"pipeline": mongo.Pipeline{
				{{Key: "$match", Value: bson.M{"$expr": bson.M{"$and": bson.A{
					bson.M{"$eq": bson.A{"$language", "$$lang"}},
					bson.M{"$eq": bson.A{"$package", "$$pkg"}},
				}}}}},
				{{Key: "$count", Value: "count"}},
			},
			"as": "popularity",
		}}},
		{{Key: "$addFields", Value: bson.M{
			"popularity_count": bson.M{
				"$ifNull": bson.A{
					bson.M{"$first": "$popularity.count"},
					0,
				},
			},
		}}},
		// Sort based on sortBy parameter
		{{Key: "$sort", Value: sortStage}},
	}

	// First get total count of unique packages
	countPipeline := mongo.Pipeline{
		{{Key: "$match", Value: matchStage}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"language": "$source.language",
				"package":  "$source.package",
			},
		}}},
		{{Key: "$count", Value: "total"}},
	}
	countCursor, err := s.renders.Aggregate(ctx, countPipeline)
	if err != nil {
		return nil, 0, err
	}
	var countResult []bson.M
	if err := countCursor.All(ctx, &countResult); err != nil {
		countCursor.Close(ctx)
		return nil, 0, err
	}
	countCursor.Close(ctx)

	var totalCount int64 = 0
	if len(countResult) > 0 {
		if t, ok := countResult[0]["total"].(int32); ok {
			totalCount = int64(t)
		} else if t, ok := countResult[0]["total"].(int64); ok {
			totalCount = t
		}
	}

	// Add pagination to main pipeline
	pipeline = append(pipeline,
		bson.D{{Key: "$skip", Value: offset}},
		bson.D{{Key: "$limit", Value: limit}},
	)

	cursor, err := s.renders.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var rawResults []bson.M
	if err := cursor.All(ctx, &rawResults); err != nil {
		return nil, 0, err
	}

	// Convert to ExploreEntry
	results := make([]storage.ExploreEntry, 0, len(rawResults))
	for _, raw := range rawResults {
		entry := storage.ExploreEntry{}

		// Parse source
		if src, ok := raw["source"].(bson.M); ok {
			entry.Source = storage.RenderSource{
				Type:     getString(src, "type"),
				Language: getString(src, "language"),
				Package:  getString(src, "package"),
			}
		}

		entry.NodeCount = getInt(raw, "node_count")
		entry.EdgeCount = getInt(raw, "edge_count")
		entry.PopularityCount = getInt(raw, "popularity_count")

		if t, ok := raw["created_at"].(primitive.DateTime); ok {
			entry.CreatedAt = t.Time()
		}

		// Parse viz_types array
		if vizTypes, ok := raw["viz_types"].(bson.A); ok {
			for _, vt := range vizTypes {
				if vtMap, ok := vt.(bson.M); ok {
					artifactSVG := getString(vtMap, "artifact_svg")
					if artifactSVG != "" {
						artifactSVG = "/api/v1/artifacts/" + artifactSVG
					}
					artifactPNG := getString(vtMap, "artifact_png")
					if artifactPNG != "" {
						artifactPNG = "/api/v1/artifacts/" + artifactPNG
					}
					artifactPDF := getString(vtMap, "artifact_pdf")
					if artifactPDF != "" {
						artifactPDF = "/api/v1/artifacts/" + artifactPDF
					}
					entry.VizTypes = append(entry.VizTypes, storage.ExploreVizType{
						VizType:     getString(vtMap, "viz_type"),
						RenderID:    getString(vtMap, "render_id"),
						GraphID:     getString(vtMap, "graph_id"),
						ArtifactSVG: artifactSVG,
						ArtifactPNG: artifactPNG,
						ArtifactPDF: artifactPDF,
					})
				}
			}
		}

		results = append(results, entry)
	}

	return results, totalCount, nil
}

// Helper functions for parsing BSON
func getString(m bson.M, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m bson.M, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

func (s *mongoStore) Close() error { return nil }

var _ storage.DocumentStore = (*mongoStore)(nil)
