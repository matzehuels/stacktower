package rubygems

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/matzehuels/stacktower/pkg/integrations"
)

func TestClient_FetchGem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/gems/rails.json" {
			resp := gemResponse{
				Name:          "rails",
				Version:       "7.1.0",
				Info:          "Ruby on Rails is a full-stack web framework",
				Licenses:      []string{"MIT"},
				SourceCodeURI: "https://github.com/rails/rails",
				HomepageURI:   "https://rubyonrails.org",
				Authors:       "David Heinemeier Hansson",
				Downloads:     500000000,
				Dependencies: dependenciesResponse{
					Runtime: []dependencyInfo{
						{Name: "activesupport", Requirements: "= 7.1.0"},
						{Name: "actionpack", Requirements: "= 7.1.0"},
					},
					Development: []dependencyInfo{
						{Name: "rake", Requirements: ">= 0"},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c, _ := NewClient(time.Hour)
	c.HTTP = server.Client()
	c.baseURL = server.URL + "/api/v1"

	info, err := c.FetchGem(context.Background(), "rails", false)
	if err != nil {
		t.Fatalf("FetchGem failed: %v", err)
	}

	if info.Name != "rails" {
		t.Errorf("expected name rails, got %s", info.Name)
	}
	if info.Version != "7.1.0" {
		t.Errorf("expected version 7.1.0, got %s", info.Version)
	}
	if len(info.Dependencies) != 2 {
		t.Errorf("expected 2 runtime dependencies, got %d", len(info.Dependencies))
	}
	if info.License != "MIT" {
		t.Errorf("expected license MIT, got %s", info.License)
	}
}

func TestClient_FetchGem_NotFound(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	c, _ := NewClient(time.Hour)
	c.HTTP = server.Client()
	c.baseURL = server.URL

	_, err := c.FetchGem(context.Background(), "missing-gem", false)
	if err == nil {
		t.Fatal("expected error for missing gem")
	}
	if !errors.Is(err, integrations.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestExtractDeps_RuntimeOnly(t *testing.T) {
	deps := dependenciesResponse{
		Runtime: []dependencyInfo{
			{Name: "activesupport", Requirements: ">= 0"},
			{Name: "actionpack", Requirements: ">= 0"},
		},
		Development: []dependencyInfo{
			{Name: "rake", Requirements: ">= 0"},
			{Name: "rspec", Requirements: ">= 0"},
		},
	}

	result := extractDeps(deps)
	if len(result) != 2 {
		t.Errorf("expected 2 runtime deps, got %d", len(result))
	}

	// Verify only runtime deps are included
	hasRake := false
	for _, d := range result {
		if d == "rake" || d == "rspec" {
			hasRake = true
		}
	}
	if hasRake {
		t.Error("expected development dependencies to be excluded")
	}
}

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Rails", "rails"},
		{"  ActiveRecord  ", "activerecord"},
		{"UPPERCASE", "uppercase"},
		{"some_gem", "some_gem"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeName(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestJoinLicenses(t *testing.T) {
	tests := []struct {
		input    []string
		expected string
	}{
		{nil, ""},
		{[]string{}, ""},
		{[]string{"MIT"}, "MIT"},
		{[]string{"MIT", "Apache-2.0"}, "MIT, Apache-2.0"},
	}

	for _, tt := range tests {
		result := joinLicenses(tt.input)
		if result != tt.expected {
			t.Errorf("joinLicenses(%v): expected %s, got %s", tt.input, tt.expected, result)
		}
	}
}
