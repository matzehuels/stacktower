package pypi

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

func TestClient_FetchPackage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/flask/json" {
			resp := apiResponse{
				Info: apiInfo{
					Name:         "Flask",
					Version:      "2.0.0",
					Summary:      "A micro web framework",
					License:      "BSD-3-Clause",
					RequiresDist: []string{"click>=7.0", "werkzeug>=2.0"},
					ProjectURLs: map[string]any{
						"Source": "https://github.com/pallets/flask",
					},
					Author: "Armin Ronacher",
				},
			}
			json.NewEncoder(w).Encode(resp)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c := testClient(t, server.URL)

	info, err := c.FetchPackage(context.Background(), "flask", true)
	if err != nil {
		t.Fatalf("FetchPackage failed: %v", err)
	}

	if info.Name != "Flask" {
		t.Errorf("expected name Flask, got %s", info.Name)
	}
	if info.Version == "" {
		t.Error("expected non-empty version")
	}
	if len(info.Dependencies) == 0 {
		t.Error("expected at least one dependency")
	}
}

func TestClient_FetchPackage_NotFound(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	c := testClient(t, server.URL)

	_, err := c.FetchPackage(context.Background(), "missing-pkg", true)
	if err == nil {
		t.Fatal("expected error for missing package")
	}
	if !errors.Is(err, integrations.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestExtractDeps_FiltersMarkers(t *testing.T) {
	tests := []struct {
		input    []string
		expected int
	}{
		{[]string{"requests", "numpy; extra == 'dev'"}, 1},
		{[]string{"django>=3.0", "pytest; extra == 'test'"}, 1},
		{[]string{"flask"}, 1},
	}

	for _, tt := range tests {
		got := extractDeps(tt.input)
		if len(got) != tt.expected {
			t.Errorf("extractDeps(%v): expected %d deps, got %d", tt.input, tt.expected, len(got))
		}
	}
}

func TestNormalizePkgName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Django", "django"},
		{"Flask_App", "flask-app"},
		{"some_package-name", "some-package-name"},
		{"UPPERCASE", "uppercase"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := integrations.NormalizePkgName(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func testClient(t *testing.T, serverURL string) *Client {
	t.Helper()
	cache, err := integrations.NewCache(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return &Client{
		Client:  integrations.NewClient(cache, nil),
		baseURL: serverURL,
	}
}
