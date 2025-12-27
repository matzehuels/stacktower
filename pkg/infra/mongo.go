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

// OperationStore returns a storage.OperationStore for operation logging.
func (m *Mongo) OperationStore() storage.OperationStore {
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
	db         *mongo.Database
	graphs     *mongo.Collection
	renders    *mongo.Collection
	operations *mongo.Collection
	gridfs     *gridfs.Bucket
}

func newMongoStore(db *mongo.Database) *mongoStore {
	graphs := db.Collection("graphs")
	renders := db.Collection("renders")
	operations := db.Collection("operations")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create indexes (best effort)
	graphs.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "scope", Value: 1}, {Key: "language", Value: 1}, {Key: "package", Value: 1}}},
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
		{Keys: bson.D{{Key: "content_hash", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
	})
	renders.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "graph_hash", Value: 1}}},
		{Keys: bson.D{{Key: "graph_id", Value: 1}}},
	})
	operations.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "type", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "created_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(90 * 24 * 60 * 60)}, // TTL 90 days
	})

	bucket, _ := gridfs.NewBucket(db, options.GridFSBucket().SetName("render_artifacts"))

	return &mongoStore{db: db, graphs: graphs, renders: renders, operations: operations, gridfs: bucket}
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
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid render ID: %w", err)
	}
	var render storage.Render
	err = s.renders.FindOne(ctx, bson.M{"_id": objID}).Decode(&render)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find render: %w", err)
	}
	s.renders.UpdateByID(ctx, objID, bson.M{"$set": bson.M{"accessed_at": time.Now()}})
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

func (s *mongoStore) DeleteRenderDoc(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid render ID: %w", err)
	}

	var render storage.Render
	s.renders.FindOne(ctx, bson.M{"_id": objID}).Decode(&render)

	// Delete artifacts
	for _, aid := range []string{render.Artifacts.SVG, render.Artifacts.PNG, render.Artifacts.PDF} {
		if aid != "" {
			if oid, err := primitive.ObjectIDFromHex(aid); err == nil {
				s.gridfs.Delete(oid)
			}
		}
	}

	_, err = s.renders.DeleteOne(ctx, bson.M{"_id": objID})
	return err
}

func (s *mongoStore) ListRenderDocs(ctx context.Context, userID string, limit, offset int) ([]*storage.Render, int64, error) {
	filter := bson.M{"user_id": userID}
	total, err := s.renders.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(int64(limit)).SetSkip(int64(offset))
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

// Artifact operations

func (s *mongoStore) StoreArtifact(ctx context.Context, renderID, filename string, data []byte) (string, error) {
	fullPath := fmt.Sprintf("%s/%s", renderID, filename)
	uploadOpts := options.GridFSUpload().SetMetadata(bson.M{
		"render_id": renderID, "filename": filename, "created_at": time.Now(),
	})
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
	// Find the render that owns this artifact
	var render storage.Render
	filter := bson.M{
		"$or": []bson.M{
			{"artifacts.svg": artifactID},
			{"artifacts.png": artifactID},
			{"artifacts.pdf": artifactID},
		},
	}
	err := s.renders.FindOne(ctx, filter).Decode(&render)
	if err == mongo.ErrNoDocuments {
		// Artifact not associated with a render - allow access (shared artifact)
		return s.GetArtifact(ctx, artifactID)
	}
	if err != nil {
		return nil, fmt.Errorf("find artifact owner: %w", err)
	}

	// Check ownership
	if render.UserID != userID {
		return nil, storage.ErrAccessDenied
	}

	return s.GetArtifact(ctx, artifactID)
}

// OperationStore implementation

func (s *mongoStore) RecordOperation(ctx context.Context, op *storage.Operation) error {
	if op.ID == "" {
		op.ID = primitive.NewObjectID().Hex()
	}
	if op.CreatedAt.IsZero() {
		op.CreatedAt = time.Now()
	}

	_, err := s.operations.InsertOne(ctx, op)
	return err
}

func (s *mongoStore) ListOperations(ctx context.Context, userID string, opType storage.OperationType, limit, offset int) ([]*storage.Operation, int64, error) {
	filter := bson.M{"user_id": userID}
	if opType != "" {
		filter["type"] = opType
	}

	total, err := s.operations.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(int64(limit)).SetSkip(int64(offset))
	cursor, err := s.operations.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var ops []*storage.Operation
	if err := cursor.All(ctx, &ops); err != nil {
		return nil, 0, err
	}
	return ops, total, nil
}

func (s *mongoStore) CountOperationsInWindow(ctx context.Context, userID string, opType storage.OperationType, windowStart int64) (int64, error) {
	filter := bson.M{
		"user_id":    userID,
		"type":       opType,
		"created_at": bson.M{"$gte": time.Unix(windowStart, 0)},
	}
	return s.operations.CountDocuments(ctx, filter)
}

func (s *mongoStore) GetOperationStats(ctx context.Context, userID string) (*storage.UserOperationStats, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"user_id": userID}}},
		{{Key: "$group", Value: bson.M{
			"_id":              nil,
			"total_operations": bson.M{"$sum": 1},
			"total_parses":     bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$type", "parse"}}, 1, 0}}},
			"total_layouts":    bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$type", "layout"}}, 1, 0}}},
			"total_renders":    bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$type", "render"}}, 1, 0}}},
			"total_cache_hits": bson.M{"$sum": bson.M{"$cond": bson.A{"$stats.cache_hit", 1, 0}}},
		}}},
	}

	cursor, err := s.operations.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []struct {
		TotalOperations int64 `bson:"total_operations"`
		TotalParses     int64 `bson:"total_parses"`
		TotalLayouts    int64 `bson:"total_layouts"`
		TotalRenders    int64 `bson:"total_renders"`
		TotalCacheHits  int64 `bson:"total_cache_hits"`
	}
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return &storage.UserOperationStats{}, nil
	}

	return &storage.UserOperationStats{
		TotalOperations: results[0].TotalOperations,
		TotalParses:     results[0].TotalParses,
		TotalLayouts:    results[0].TotalLayouts,
		TotalRenders:    results[0].TotalRenders,
		TotalCacheHits:  results[0].TotalCacheHits,
	}, nil
}

func (s *mongoStore) Ping(ctx context.Context) error {
	return s.db.Client().Ping(ctx, nil)
}

func (s *mongoStore) Close() error { return nil }

var _ storage.DocumentStore = (*mongoStore)(nil)
var _ storage.OperationStore = (*mongoStore)(nil)
