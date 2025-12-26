// Package infra provides infrastructure configuration and clients.
//
// Configuration is loaded from environment variables with sensible defaults.
// Use Load() to get the full configuration, or use individual Config*() functions.

package infra

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all infrastructure configuration.
type Config struct {
	Redis  RedisConfig
	Mongo  MongoConfig
	GitHub GitHubConfig
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr              string        // STACKTOWER_REDIS_ADDR (required for API)
	Password          string        // STACKTOWER_REDIS_PASSWORD
	DB                int           // STACKTOWER_REDIS_DB (default: 0)
	PoolSize          int           // STACKTOWER_REDIS_POOL_SIZE (default: 10)
	DialTimeout       time.Duration // STACKTOWER_REDIS_DIAL_TIMEOUT (default: 5s)
	ReadTimeout       time.Duration // STACKTOWER_REDIS_READ_TIMEOUT (default: 3s)
	WriteTimeout      time.Duration // STACKTOWER_REDIS_WRITE_TIMEOUT (default: 3s)
	QueueStream       string        // STACKTOWER_QUEUE_STREAM (default: stacktower:queue:stream)
	QueueGroup        string        // STACKTOWER_QUEUE_GROUP (default: workers)
	QueueConsumer     string        // STACKTOWER_QUEUE_CONSUMER (default: auto-generated)
	QueueBlockTimeout time.Duration // Block timeout for dequeue (default: 5s)
	QueueClaimTimeout time.Duration // Claim timeout for orphaned jobs (default: 5m)
	QueueMaxRetries   int           // Max retries for failed jobs (default: 3)
}

// MongoConfig holds MongoDB connection settings.
type MongoConfig struct {
	URI                    string        // STACKTOWER_MONGODB_URI (required for API)
	Database               string        // STACKTOWER_MONGODB_DATABASE (default: stacktower)
	ConnectTimeout         time.Duration // Connection timeout (default: 10s)
	ServerSelectionTimeout time.Duration // Server selection timeout (default: 5s)
}

// GitHubConfig holds GitHub OAuth and API settings.
type GitHubConfig struct {
	ClientID     string // GITHUB_CLIENT_ID (required for OAuth)
	ClientSecret string // GITHUB_CLIENT_SECRET (required for OAuth)
	RedirectURI  string // GITHUB_REDIRECT_URI
	Token        string // GITHUB_TOKEN (for CLI/API GitHub access)
	FrontendURL  string // FRONTEND_URL (redirect after OAuth)
}

// Load reads all configuration from environment variables.
func Load() *Config {
	return &Config{
		Redis:  LoadRedisConfig(),
		Mongo:  LoadMongoConfig(),
		GitHub: LoadGitHubConfig(),
	}
}

// LoadRedisConfig reads Redis configuration from environment.
func LoadRedisConfig() RedisConfig {
	return RedisConfig{
		Addr:              env("STACKTOWER_REDIS_ADDR", ""),
		Password:          env("STACKTOWER_REDIS_PASSWORD", ""),
		DB:                envInt("STACKTOWER_REDIS_DB", 0),
		PoolSize:          envInt("STACKTOWER_REDIS_POOL_SIZE", 10),
		DialTimeout:       envDuration("STACKTOWER_REDIS_DIAL_TIMEOUT", 5*time.Second),
		ReadTimeout:       envDuration("STACKTOWER_REDIS_READ_TIMEOUT", 3*time.Second),
		WriteTimeout:      envDuration("STACKTOWER_REDIS_WRITE_TIMEOUT", 3*time.Second),
		QueueStream:       env("STACKTOWER_QUEUE_STREAM", "stacktower:queue:stream"),
		QueueGroup:        env("STACKTOWER_QUEUE_GROUP", "workers"),
		QueueConsumer:     env("STACKTOWER_QUEUE_CONSUMER", fmt.Sprintf("consumer-%d", time.Now().UnixNano())),
		QueueBlockTimeout: envDuration("STACKTOWER_QUEUE_BLOCK_TIMEOUT", 5*time.Second),
		QueueClaimTimeout: envDuration("STACKTOWER_QUEUE_CLAIM_TIMEOUT", 5*time.Minute),
		QueueMaxRetries:   envInt("STACKTOWER_QUEUE_MAX_RETRIES", 3),
	}
}

// LoadMongoConfig reads MongoDB configuration from environment.
func LoadMongoConfig() MongoConfig {
	return MongoConfig{
		URI:                    env("STACKTOWER_MONGODB_URI", ""),
		Database:               env("STACKTOWER_MONGODB_DATABASE", "stacktower"),
		ConnectTimeout:         envDuration("STACKTOWER_MONGODB_CONNECT_TIMEOUT", 10*time.Second),
		ServerSelectionTimeout: envDuration("STACKTOWER_MONGODB_SERVER_SELECTION_TIMEOUT", 5*time.Second),
	}
}

// LoadGitHubConfig reads GitHub configuration from environment.
func LoadGitHubConfig() GitHubConfig {
	return GitHubConfig{
		ClientID:     env("GITHUB_CLIENT_ID", ""),
		ClientSecret: env("GITHUB_CLIENT_SECRET", ""),
		RedirectURI:  env("GITHUB_REDIRECT_URI", ""),
		Token:        env("GITHUB_TOKEN", ""),
		FrontendURL:  env("FRONTEND_URL", "http://localhost:3000"),
	}
}

// Validate checks that required fields are set for API mode.
func (c *Config) ValidateForAPI() error {
	if c.Redis.Addr == "" {
		return fmt.Errorf("STACKTOWER_REDIS_ADDR is required")
	}
	if c.Mongo.URI == "" {
		return fmt.Errorf("STACKTOWER_MONGODB_URI is required")
	}
	return nil
}

// ValidateGitHubOAuth checks that GitHub OAuth is properly configured.
func (c *Config) ValidateGitHubOAuth() error {
	if c.GitHub.ClientID == "" {
		return fmt.Errorf("GITHUB_CLIENT_ID is required for OAuth")
	}
	if c.GitHub.ClientSecret == "" {
		return fmt.Errorf("GITHUB_CLIENT_SECRET is required for OAuth")
	}
	return nil
}

// Helper functions for reading environment variables

func env(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func envInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultValue
}

func envDuration(key string, defaultValue time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultValue
}
