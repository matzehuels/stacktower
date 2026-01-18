// Package config provides configuration loading from environment variables.
package config

import "os"

// GitHubConfig holds GitHub OAuth and API settings.
type GitHubConfig struct {
	ClientID string // GITHUB_CLIENT_ID (for OAuth device flow)
	Token    string // GITHUB_TOKEN (for API access)
}

// LoadGitHubConfig reads GitHub configuration from environment.
func LoadGitHubConfig() GitHubConfig {
	return GitHubConfig{
		ClientID: os.Getenv("GITHUB_CLIENT_ID"),
		Token:    os.Getenv("GITHUB_TOKEN"),
	}
}

// env returns the value of an environment variable or a default value.
func env(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
