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

	"github.com/matzehuels/stacktower/pkg/infra/cache"
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

// Store returns a cache.Store for document operations.
func (m *Mongo) Store() cache.Store {
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
	db      *mongo.Database
	graphs  *mongo.Collection
	renders *mongo.Collection
	gridfs  *gridfs.Bucket
}

func newMongoStore(db *mongo.Database) *mongoStore {
	graphs := db.Collection("graphs")
	renders := db.Collection("renders")

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

	bucket, _ := gridfs.NewBucket(db, options.GridFSBucket().SetName("render_artifacts"))

	return &mongoStore{db: db, graphs: graphs, renders: renders, gridfs: bucket}
}

// Graph operations

func (s *mongoStore) GetGraph(ctx context.Context, id string) (*cache.Graph, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid graph ID: %w", err)
	}
	var graph cache.Graph
	err = s.graphs.FindOne(ctx, bson.M{"_id": objID}).Decode(&graph)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find graph: %w", err)
	}
	return &graph, nil
}

func (s *mongoStore) StoreGraph(ctx context.Context, graph *cache.Graph) error {
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

func (s *mongoStore) GetRender(ctx context.Context, id string) (*cache.Render, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid render ID: %w", err)
	}
	var render cache.Render
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

func (s *mongoStore) StoreRender(ctx context.Context, render *cache.Render) error {
	now := time.Now()
	if render.ID == "" {
		render.ID = primitive.NewObjectID().Hex()
	}
	render.CreatedAt = now
	render.AccessedAt = now
	_, err := s.renders.InsertOne(ctx, render)
	return err
}

func (s *mongoStore) DeleteRender(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid render ID: %w", err)
	}

	var render cache.Render
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

func (s *mongoStore) ListRenders(ctx context.Context, userID string, limit, offset int) ([]*cache.Render, int64, error) {
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

	var renders []*cache.Render
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

func (s *mongoStore) Close() error { return nil }

var _ cache.Store = (*mongoStore)(nil)
