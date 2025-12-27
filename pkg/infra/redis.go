// Package infra provides unified infrastructure clients for Stacktower.
//
// This package consolidates all Redis and MongoDB connections into single
// clients that implement multiple interfaces. This eliminates scattered
// connections and provides a clean, centralized configuration.
//
// # Redis Client
//
// The Redis client implements:
//   - queue.Queue (job queue via Redis Streams)
//   - session.Store (session storage with TTL)
//   - session.StateStore (OAuth state tokens)
//   - storage.Index (fast lookup cache, Tier 1)
//
// # MongoDB Client
//
// The MongoDB client implements:
//   - storage.DocumentStore (document storage for graphs, renders)
//   - Provides GridFS bucket for binary artifacts
//
// # Usage
//
//	redis, _ := infra.NewRedis(ctx, infra.RedisConfig{Addr: "localhost:6379"})
//	defer redis.Close()
//
//	mongo, _ := infra.NewMongo(ctx, infra.MongoConfig{URI: "mongodb://localhost:27017"})
//	defer mongo.Close()
//
//	// All share the same connections
//	q := redis.Queue()
//	sess := redis.Sessions()
//	lookup := redis.Cache()
//	store := mongo.Store()
package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/matzehuels/stacktower/pkg/infra/queue"
	"github.com/matzehuels/stacktower/pkg/infra/session"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
)

// =============================================================================
// Redis Client
// =============================================================================

// Redis is a unified Redis client that provides sub-interfaces for
// queue, session, and cache operations.
type Redis struct {
	client    *redis.Client
	config    RedisConfig
	encryptor *session.TokenEncryptor // Optional: encrypts access tokens at rest
}

// NewRedis creates a new unified Redis client.
func NewRedis(ctx context.Context, cfg RedisConfig) (*Redis, error) {
	// Apply defaults
	if cfg.Addr == "" {
		cfg.Addr = "localhost:6379"
	}
	if cfg.PoolSize == 0 {
		cfg.PoolSize = 10
	}
	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = 5 * time.Second
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 3 * time.Second
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 3 * time.Second
	}
	if cfg.QueueStream == "" {
		cfg.QueueStream = "stacktower:queue:stream"
	}
	if cfg.QueueGroup == "" {
		cfg.QueueGroup = "workers"
	}
	if cfg.QueueConsumer == "" {
		cfg.QueueConsumer = fmt.Sprintf("consumer-%d", time.Now().UnixNano())
	}
	if cfg.QueueBlockTimeout == 0 {
		cfg.QueueBlockTimeout = 5 * time.Second
	}
	if cfg.QueueClaimTimeout == 0 {
		cfg.QueueClaimTimeout = 5 * time.Minute
	}
	if cfg.QueueMaxRetries == 0 {
		cfg.QueueMaxRetries = 3
	}

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &Redis{client: client, config: cfg}, nil
}

// WithSessionEncryption configures token encryption for sessions.
// Call this after NewRedis to enable at-rest encryption of access tokens.
// The key must be a base64-encoded 32-byte key.
// If key is empty, encryption is disabled (not recommended for production).
func (r *Redis) WithSessionEncryption(encodedKey string) error {
	if encodedKey == "" {
		return nil // Encryption disabled
	}

	key, err := session.ParseKey(encodedKey)
	if err != nil {
		return fmt.Errorf("parse session key: %w", err)
	}

	encryptor, err := session.NewTokenEncryptor(key)
	if err != nil {
		return fmt.Errorf("create encryptor: %w", err)
	}

	r.encryptor = encryptor
	return nil
}

// Queue returns a job queue using Redis Streams.
func (r *Redis) Queue() queue.Queue {
	return &redisQueue{
		client:       r.client,
		stream:       r.config.QueueStream,
		group:        r.config.QueueGroup,
		consumer:     r.config.QueueConsumer,
		blockTimeout: r.config.QueueBlockTimeout,
		claimTimeout: r.config.QueueClaimTimeout,
		maxRetries:   r.config.QueueMaxRetries,
	}
}

// Sessions returns a session store with TTL.
// If an encryption key was configured, access tokens are encrypted at rest.
func (r *Redis) Sessions() session.Store {
	store := &redisSessionStore{client: r.client, prefix: "stacktower:session:"}
	if r.encryptor != nil {
		return session.NewEncryptingStore(store, r.encryptor)
	}
	return store
}

// OAuthStates returns an OAuth state token store.
func (r *Redis) OAuthStates() session.StateStore {
	return &redisStateStore{client: r.client, prefix: "stacktower:oauth_state:"}
}

// Index returns the cache index (Tier 1 of two-tier caching).
func (r *Redis) Index() storage.Index {
	return &redisIndex{client: r.client}
}

// HTTPCache returns the HTTP response cache.
func (r *Redis) HTTPCache() storage.HTTPCache {
	return &redisHTTPCache{client: r.client}
}

// RateLimiter returns the rate limiter for quota management.
func (r *Redis) RateLimiter() storage.RateLimiter {
	return &redisRateLimiter{client: r.client}
}

// Raw returns the underlying redis.Client for advanced operations.
func (r *Redis) Raw() *redis.Client {
	return r.client
}

// Close closes the Redis connection.
func (r *Redis) Close() error {
	return r.client.Close()
}

// Info returns connection info for logging.
func (r *Redis) Info() string {
	return fmt.Sprintf("redis (%s)", r.config.Addr)
}

// Ping checks if Redis is reachable.
func (r *Redis) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// =============================================================================
// Queue Implementation
// =============================================================================

type redisQueue struct {
	client       *redis.Client
	stream       string
	group        string
	consumer     string
	blockTimeout time.Duration
	claimTimeout time.Duration
	maxRetries   int
}

func (q *redisQueue) Enqueue(ctx context.Context, job *queue.Job) error {
	// Ensure consumer group exists
	q.client.XGroupCreateMkStream(ctx, q.stream, q.group, "0")

	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}

	_, err = q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: q.stream,
		Values: map[string]interface{}{"job_id": job.ID, "data": string(data)},
	}).Result()
	if err != nil {
		return fmt.Errorf("enqueue job: %w", err)
	}

	if err := q.client.Set(ctx, q.jobKey(job.ID), data, 24*time.Hour).Err(); err != nil {
		return err
	}

	// Add to user's job index if user_id is present in payload
	if userID, ok := job.Payload["user_id"].(string); ok && userID != "" {
		q.client.SAdd(ctx, q.userJobsKey(userID), job.ID)
		q.client.Expire(ctx, q.userJobsKey(userID), 7*24*time.Hour) // Expire user job sets after 7 days
	}

	return nil
}

func (q *redisQueue) Dequeue(ctx context.Context, jobTypes ...string) (*queue.Job, error) {
	// Ensure consumer group exists
	q.client.XGroupCreateMkStream(ctx, q.stream, q.group, "0")

	// Try to claim pending messages first
	if job, _ := q.claimPending(ctx); job != nil {
		return job, nil
	}

	streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    q.group,
		Consumer: q.consumer,
		Streams:  []string{q.stream, ">"},
		Count:    1,
		Block:    q.blockTimeout,
	}).Result()

	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read from stream: %w", err)
	}
	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, nil
	}

	return q.parseMessage(ctx, streams[0].Messages[0])
}

func (q *redisQueue) claimPending(ctx context.Context) (*queue.Job, error) {
	pending, err := q.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: q.stream, Group: q.group, Start: "-", End: "+", Count: 10,
	}).Result()
	if err != nil {
		return nil, nil
	}

	for _, p := range pending {
		if p.Idle < q.claimTimeout {
			continue
		}
		messages, err := q.client.XClaim(ctx, &redis.XClaimArgs{
			Stream: q.stream, Group: q.group, Consumer: q.consumer,
			MinIdle: q.claimTimeout, Messages: []string{p.ID},
		}).Result()
		if err != nil || len(messages) == 0 {
			continue
		}
		job, err := q.parseMessage(ctx, messages[0])
		if err != nil {
			if p.RetryCount > int64(q.maxRetries) {
				q.client.XAck(ctx, q.stream, q.group, p.ID)
			}
			continue
		}
		return job, nil
	}
	return nil, nil
}

func (q *redisQueue) parseMessage(ctx context.Context, msg redis.XMessage) (*queue.Job, error) {
	data, ok := msg.Values["data"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid message format")
	}
	var job queue.Job
	if err := json.Unmarshal([]byte(data), &job); err != nil {
		return nil, fmt.Errorf("unmarshal job: %w", err)
	}
	q.client.Set(ctx, q.msgIDKey(job.ID), msg.ID, 24*time.Hour)
	return &job, nil
}

func (q *redisQueue) Get(ctx context.Context, jobID string) (*queue.Job, error) {
	data, err := q.client.Get(ctx, q.jobKey(jobID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	var job queue.Job
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, fmt.Errorf("unmarshal job: %w", err)
	}
	return &job, nil
}

func (q *redisQueue) UpdateStatus(ctx context.Context, jobID string, status queue.Status, result map[string]interface{}, errorMsg string) error {
	job, err := q.Get(ctx, jobID)
	if err != nil {
		return err
	}
	job.Status = status
	if result != nil {
		job.Result = result
	}
	if errorMsg != "" {
		job.Error = errorMsg
	}
	if status == queue.StatusCompleted || status == queue.StatusFailed {
		now := time.Now()
		job.CompletedAt = &now
	}
	data, _ := json.Marshal(job)
	ttl := 24 * time.Hour
	if status == queue.StatusCompleted || status == queue.StatusFailed {
		ttl = 7 * 24 * time.Hour
	}
	return q.client.Set(ctx, q.jobKey(jobID), data, ttl).Err()
}

func (q *redisQueue) Cancel(ctx context.Context, jobID string) error {
	job, err := q.Get(ctx, jobID)
	if err != nil {
		return err
	}
	if job.Status == queue.StatusRunning {
		return fmt.Errorf("cannot cancel running job")
	}
	if job.Status == queue.StatusCompleted || job.Status == queue.StatusFailed {
		return fmt.Errorf("cannot cancel finished job")
	}
	job.Status = queue.StatusCancelled
	data, _ := json.Marshal(job)
	return q.client.Set(ctx, q.jobKey(jobID), data, 24*time.Hour).Err()
}

func (q *redisQueue) List(ctx context.Context, statuses ...queue.Status) ([]*queue.Job, error) {
	// Scan all job keys
	var cursor uint64
	var allJobs []*queue.Job
	statusMap := make(map[queue.Status]bool)
	for _, s := range statuses {
		statusMap[s] = true
	}
	filterByStatus := len(statuses) > 0

	for {
		keys, nextCursor, err := q.client.Scan(ctx, cursor, "stacktower:queue:job:*", 100).Result()
		if err != nil {
			return nil, fmt.Errorf("scan job keys: %w", err)
		}

		for _, key := range keys {
			data, err := q.client.Get(ctx, key).Bytes()
			if err != nil {
				continue
			}
			var job queue.Job
			if err := json.Unmarshal(data, &job); err != nil {
				continue
			}
			if filterByStatus && !statusMap[job.Status] {
				continue
			}
			allJobs = append(allJobs, &job)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(allJobs, func(i, j int) bool {
		return allJobs[i].CreatedAt.After(allJobs[j].CreatedAt)
	})

	return allJobs, nil
}

func (q *redisQueue) ListByUser(ctx context.Context, userID string, statuses ...queue.Status) ([]*queue.Job, error) {
	// Get job IDs from user's job set
	jobIDs, err := q.client.SMembers(ctx, q.userJobsKey(userID)).Result()
	if err != nil {
		return nil, fmt.Errorf("get user jobs: %w", err)
	}

	statusMap := make(map[queue.Status]bool)
	for _, s := range statuses {
		statusMap[s] = true
	}
	filterByStatus := len(statuses) > 0

	var jobs []*queue.Job
	for _, jobID := range jobIDs {
		job, err := q.Get(ctx, jobID)
		if err != nil {
			// Job might have expired, remove from index
			q.client.SRem(ctx, q.userJobsKey(userID), jobID)
			continue
		}
		if filterByStatus && !statusMap[job.Status] {
			continue
		}
		jobs = append(jobs, job)
	}

	// Sort by creation time (newest first)
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt.After(jobs[j].CreatedAt)
	})

	return jobs, nil
}

func (q *redisQueue) Delete(ctx context.Context, jobID string) error {
	// Try to get the job to clean up user index
	job, err := q.Get(ctx, jobID)
	if err == nil && job != nil {
		if userID, ok := job.Payload["user_id"].(string); ok && userID != "" {
			q.client.SRem(ctx, q.userJobsKey(userID), jobID)
		}
	}

	q.client.Del(ctx, q.msgIDKey(jobID))
	return q.client.Del(ctx, q.jobKey(jobID)).Err()
}

func (q *redisQueue) Ping(ctx context.Context) error {
	return q.client.Ping(ctx).Err()
}

func (q *redisQueue) Close() error { return nil }

func (q *redisQueue) jobKey(id string) string          { return "stacktower:queue:job:" + id }
func (q *redisQueue) msgIDKey(id string) string        { return "stacktower:queue:msgid:" + id }
func (q *redisQueue) userJobsKey(userID string) string { return "stacktower:queue:user:" + userID }

var _ queue.Queue = (*redisQueue)(nil)

// =============================================================================
// Session Implementation
// =============================================================================

type redisSessionStore struct {
	client *redis.Client
	prefix string
}

func (s *redisSessionStore) Get(ctx context.Context, sessionID string) (*session.Session, error) {
	data, err := s.client.Get(ctx, s.prefix+sessionID).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get: %w", err)
	}
	var sess session.Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	if sess.IsExpired() {
		return nil, nil
	}
	return &sess, nil
}

func (s *redisSessionStore) Set(ctx context.Context, sess *session.Session) error {
	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	ttl := time.Until(sess.ExpiresAt)
	if ttl <= 0 {
		return nil
	}
	return s.client.Set(ctx, s.prefix+sess.ID, data, ttl).Err()
}

func (s *redisSessionStore) Delete(ctx context.Context, sessionID string) error {
	return s.client.Del(ctx, s.prefix+sessionID).Err()
}

func (s *redisSessionStore) Cleanup(ctx context.Context) error { return nil }
func (s *redisSessionStore) Close() error                      { return nil }

var _ session.Store = (*redisSessionStore)(nil)

// =============================================================================
// OAuth State Implementation
// =============================================================================

type redisStateStore struct {
	client *redis.Client
	prefix string
}

func (s *redisStateStore) Generate(ctx context.Context, ttl time.Duration) (string, error) {
	state, err := session.GenerateState()
	if err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}
	if err := s.client.Set(ctx, s.prefix+state, "1", ttl).Err(); err != nil {
		return "", fmt.Errorf("redis set: %w", err)
	}
	return state, nil
}

func (s *redisStateStore) Validate(ctx context.Context, state string) (bool, error) {
	result, err := s.client.GetDel(ctx, s.prefix+state).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("redis getdel: %w", err)
	}
	return result != "", nil
}

func (s *redisStateStore) Cleanup(ctx context.Context) error { return nil }

var _ session.StateStore = (*redisStateStore)(nil)

// =============================================================================
// Index Implementation (storage.Index)
// =============================================================================

type redisIndex struct {
	client *redis.Client
}

func (c *redisIndex) GetGraphEntry(ctx context.Context, key string) (*storage.CacheEntry, error) {
	return c.getEntry(ctx, "stacktower:graph:"+key)
}

func (c *redisIndex) SetGraphEntry(ctx context.Context, key string, entry *storage.CacheEntry) error {
	return c.setEntry(ctx, "stacktower:graph:"+key, entry)
}

func (c *redisIndex) GetRenderEntry(ctx context.Context, key string) (*storage.CacheEntry, error) {
	return c.getEntry(ctx, "stacktower:render:"+key)
}

func (c *redisIndex) SetRenderEntry(ctx context.Context, key string, entry *storage.CacheEntry) error {
	return c.setEntry(ctx, "stacktower:render:"+key, entry)
}

func (c *redisIndex) getEntry(ctx context.Context, key string) (*storage.CacheEntry, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get: %w", err)
	}
	var entry storage.CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("unmarshal entry: %w", err)
	}
	if entry.IsExpired() {
		c.client.Del(ctx, key)
		return nil, nil
	}
	return &entry, nil
}

func (c *redisIndex) setEntry(ctx context.Context, key string, entry *storage.CacheEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}
	ttl := time.Until(entry.ExpiresAt)
	if ttl <= 0 {
		ttl = storage.GraphTTL
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *redisIndex) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *redisIndex) Close() error { return nil }

var _ storage.Index = (*redisIndex)(nil)

// =============================================================================
// HTTP Cache Implementation (storage.HTTPCache)
// =============================================================================

type redisHTTPCache struct {
	client *redis.Client
}

func (c *redisHTTPCache) GetHTTP(ctx context.Context, key string) ([]byte, bool, error) {
	data, err := c.client.Get(ctx, "stacktower:http:"+key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("redis get: %w", err)
	}
	return data, true, nil
}

func (c *redisHTTPCache) SetHTTP(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	return c.client.Set(ctx, "stacktower:http:"+key, data, ttl).Err()
}

func (c *redisHTTPCache) DeleteHTTP(ctx context.Context, key string) error {
	return c.client.Del(ctx, "stacktower:http:"+key).Err()
}

var _ storage.HTTPCache = (*redisHTTPCache)(nil)

// =============================================================================
// Rate Limiter Implementation (storage.RateLimiter)
// =============================================================================

type redisRateLimiter struct {
	client *redis.Client
}

func (r *redisRateLimiter) CheckRateLimit(ctx context.Context, userID string, opType storage.OperationType, quota storage.QuotaConfig) error {
	key := r.rateLimitKey(userID, opType)

	count, err := r.client.Get(ctx, key).Int()
	if err == redis.Nil {
		return nil // No limit set yet
	}
	if err != nil {
		return fmt.Errorf("check rate limit: %w", err)
	}

	var limit int
	switch opType {
	case storage.OpTypeParse:
		limit = quota.MaxParsesPerHour
	case storage.OpTypeLayout:
		limit = quota.MaxLayoutsPerHour
	case storage.OpTypeRender:
		limit = quota.MaxRendersPerHour
	}

	if count >= limit {
		return storage.ErrRateLimited
	}
	return nil
}

func (r *redisRateLimiter) IncrementRateLimit(ctx context.Context, userID string, opType storage.OperationType) error {
	key := r.rateLimitKey(userID, opType)

	pipe := r.client.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, time.Hour) // 1 hour sliding window
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("increment rate limit: %w", err)
	}

	_ = incr.Val() // Result not needed
	return nil
}

func (r *redisRateLimiter) GetRateLimitStatus(ctx context.Context, userID string, quota storage.QuotaConfig) (*storage.RateLimitStatus, error) {
	pipe := r.client.Pipeline()

	parseKey := r.rateLimitKey(userID, storage.OpTypeParse)
	layoutKey := r.rateLimitKey(userID, storage.OpTypeLayout)
	renderKey := r.rateLimitKey(userID, storage.OpTypeRender)
	storageKey := r.storageKey(userID)

	parsesCmd := pipe.Get(ctx, parseKey)
	layoutsCmd := pipe.Get(ctx, layoutKey)
	rendersCmd := pipe.Get(ctx, renderKey)
	storageCmd := pipe.Get(ctx, storageKey)
	ttlCmd := pipe.TTL(ctx, parseKey)

	pipe.Exec(ctx)

	getInt := func(cmd *redis.StringCmd) int {
		v, _ := cmd.Int()
		return v
	}
	getInt64 := func(cmd *redis.StringCmd) int64 {
		v, _ := cmd.Int64()
		return v
	}

	windowResetAt := time.Now().Add(time.Hour).Unix()
	if ttl := ttlCmd.Val(); ttl > 0 {
		windowResetAt = time.Now().Add(ttl).Unix()
	}

	return &storage.RateLimitStatus{
		ParsesUsed:        getInt(parsesCmd),
		ParsesLimit:       quota.MaxParsesPerHour,
		LayoutsUsed:       getInt(layoutsCmd),
		LayoutsLimit:      quota.MaxLayoutsPerHour,
		RendersUsed:       getInt(rendersCmd),
		RendersLimit:      quota.MaxRendersPerHour,
		StorageBytesUsed:  getInt64(storageCmd),
		StorageBytesLimit: quota.MaxStorageBytes,
		WindowResetAt:     windowResetAt,
	}, nil
}

func (r *redisRateLimiter) CheckStorageQuota(ctx context.Context, userID string, bytesToAdd int64, quota storage.QuotaConfig) error {
	key := r.storageKey(userID)

	current, err := r.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		current = 0
	} else if err != nil {
		return fmt.Errorf("check storage quota: %w", err)
	}

	if current+bytesToAdd > quota.MaxStorageBytes {
		return storage.ErrQuotaExceeded
	}
	return nil
}

func (r *redisRateLimiter) UpdateStorageUsage(ctx context.Context, userID string, byteDelta int64) error {
	key := r.storageKey(userID)

	if byteDelta > 0 {
		return r.client.IncrBy(ctx, key, byteDelta).Err()
	} else if byteDelta < 0 {
		// Decrement but don't go below 0
		result := r.client.IncrBy(ctx, key, byteDelta)
		if result.Err() != nil {
			return result.Err()
		}
		if result.Val() < 0 {
			return r.client.Set(ctx, key, 0, 0).Err()
		}
	}
	return nil
}

func (r *redisRateLimiter) rateLimitKey(userID string, opType storage.OperationType) string {
	return fmt.Sprintf("stacktower:ratelimit:%s:%s", userID, opType)
}

func (r *redisRateLimiter) storageKey(userID string) string {
	return "stacktower:storage:" + userID
}

var _ storage.RateLimiter = (*redisRateLimiter)(nil)
