package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestClient_Fetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		path := r.URL.Path
		switch {
		case path == "/repos/owner/repo":
			json.NewEncoder(w).Encode(repoResponse{
				Stars: 100,
				Size:  500,
				License: struct {
					SPDXID string `json:"spdx_id"`
				}{SPDXID: "MIT"},
			})
		case path == "/repos/owner/repo/releases/latest":
			w.WriteHeader(http.StatusNotFound)
		case path == "/repos/owner/repo/contributors":
			json.NewEncoder(w).Encode([]contributorResponse{
				{Login: "user1", Contributions: 10, Type: "User"},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c, _ := NewClient("", time.Hour)
	c.HTTP = server.Client()
	c.baseURL = server.URL

	metrics, err := c.Fetch(context.Background(), "owner", "repo", false)
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}

	if metrics.Stars != 100 {
		t.Errorf("expected 100 stars, got %d", metrics.Stars)
	}
	if metrics.SizeKB != 500 {
		t.Errorf("expected 500 KB, got %d", metrics.SizeKB)
	}
}

func TestExtractURL(t *testing.T) {
	tests := []struct {
		urls      map[string]string
		home      string
		wantOwner string
		wantRepo  string
		wantOK    bool
	}{
		{
			urls:      map[string]string{"Source": "https://github.com/foo/bar"},
			wantOwner: "foo",
			wantRepo:  "bar",
			wantOK:    true,
		},
		{
			urls:      nil,
			home:      "http://github.com/baz/qux",
			wantOwner: "baz",
			wantRepo:  "qux",
			wantOK:    true,
		},
		{
			urls:   map[string]string{"Homepage": "https://google.com"},
			wantOK: false,
		},
	}

	for _, tt := range tests {
		owner, repo, ok := ExtractURL(tt.urls, tt.home)
		if ok != tt.wantOK {
			t.Errorf("got ok=%v, want %v", ok, tt.wantOK)
		}
		if ok {
			if owner != tt.wantOwner {
				t.Errorf("got owner %s, want %s", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("got repo %s, want %s", repo, tt.wantRepo)
			}
		}
	}
}

func TestNewClient(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("skipping cache creation test in CI")
	}

	c, err := NewClient("test-token", time.Hour)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if c.token != "test-token" {
		t.Errorf("expected token 'test-token', got %s", c.token)
	}
	if c.Cache == nil {
		t.Error("expected cache to be initialized")
	}
	if c.HTTP == nil {
		t.Error("expected http client to be initialized")
	}
}
