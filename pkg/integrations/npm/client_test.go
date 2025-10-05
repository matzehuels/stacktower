package npm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	c, err := NewClient(time.Hour)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if c.baseURL != "https://registry.npmjs.org" {
		t.Errorf("expected base URL %s, got %s", "https://registry.npmjs.org", c.baseURL)
	}
}

func TestClient_FetchPackage(t *testing.T) {
	response := registryResponse{
		Name: "express",
		DistTags: distTags{
			Latest: "4.18.0",
		},
		Versions: map[string]versionDetails{
			"4.18.0": {
				Description: "Fast, unopinionated web framework",
				License:     "MIT",
				Author:      "TJ Holowaychuk",
				Repository: map[string]interface{}{
					"type": "git",
					"url":  "git+https://github.com/expressjs/express.git",
				},
				HomePage: "https://expressjs.com",
				Dependencies: map[string]string{
					"body-parser": "1.20.0",
					"cookie":      "0.5.0",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/express" {
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c, err := NewClient(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	c.baseURL = server.URL

	info, err := c.FetchPackage(context.Background(), "express", true)
	if err != nil {
		t.Fatalf("FetchPackage failed: %v", err)
	}

	if info.Name != "express" {
		t.Errorf("expected name express, got %s", info.Name)
	}
	if info.Version != "4.18.0" {
		t.Errorf("expected version 4.18.0, got %s", info.Version)
	}
	if len(info.Dependencies) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(info.Dependencies))
	}
	if info.Repository != "https://github.com/expressjs/express" {
		t.Errorf("expected normalized repo URL, got %s", info.Repository)
	}
}

func TestClient_FetchPackage_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c, err := NewClient(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	c.baseURL = server.URL

	_, err = c.FetchPackage(context.Background(), "nonexistent", true)
	if err == nil {
		t.Error("expected error for nonexistent package")
	}
}

func TestNormalizeRepoURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "git+https",
			input:    "git+https://github.com/user/repo.git",
			expected: "https://github.com/user/repo",
		},
		{
			name:     "git protocol",
			input:    "git://github.com/user/repo.git",
			expected: "https://github.com/user/repo",
		},
		{
			name:     "ssh format",
			input:    "git@github.com:user/repo.git",
			expected: "https://github.com/user/repo",
		},
		{
			name:     "plain https",
			input:    "https://github.com/user/repo",
			expected: "https://github.com/user/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeRepoURL(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestExtractString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		field    string
		expected string
	}{
		{"string", "MIT", "type", "MIT"},
		{"object", map[string]interface{}{"type": "MIT"}, "type", "MIT"},
		{"nil", nil, "type", ""},
		{"empty object", map[string]interface{}{}, "type", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractString(tt.input, tt.field)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
