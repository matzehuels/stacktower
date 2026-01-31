package crates

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/matzehuels/stacktower/pkg/cache"

	"github.com/matzehuels/stacktower/pkg/integrations"
)

func TestNewClient(t *testing.T) {
	c := NewClient(cache.NewNullCache(), time.Hour)
	if c.Client == nil {
		t.Error("expected client to be initialized")
	}
}

func TestClient_FetchCrate(t *testing.T) {
	crateResp := crateResponse{}
	crateResp.Crate.Name = "serde"
	crateResp.Crate.MaxVersion = "1.0.0"
	crateResp.Crate.Description = "A serialization framework"
	crateResp.Crate.License = "MIT"
	crateResp.Crate.Repository = "https://github.com/serde-rs/serde"
	crateResp.Crate.Downloads = 1000000

	depsResp := depsResponse{
		Dependencies: []struct {
			CrateID  string `json:"crate_id"`
			Kind     string `json:"kind"`
			Optional bool   `json:"optional"`
		}{
			{CrateID: "serde_derive", Kind: "normal", Optional: false},
			{CrateID: "test_dep", Kind: "dev", Optional: false},
			{CrateID: "optional_dep", Kind: "normal", Optional: true},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/crates/serde":
			json.NewEncoder(w).Encode(crateResp)
		case "/crates/serde/1.0.0/dependencies":
			json.NewEncoder(w).Encode(depsResp)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := testClient(t, server.URL)

	info, err := c.FetchCrate(context.Background(), "serde", true)
	if err != nil {
		t.Fatalf("FetchCrate failed: %v", err)
	}

	if info.Name != "serde" {
		t.Errorf("expected name serde, got %s", info.Name)
	}
	if info.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", info.Version)
	}
	if len(info.Dependencies) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(info.Dependencies))
	}
	if len(info.Dependencies) > 0 && info.Dependencies[0] != "serde_derive" {
		t.Errorf("expected serde_derive, got %s", info.Dependencies[0])
	}
}

func TestClient_FetchCrate_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := testClient(t, server.URL)

	_, err := c.FetchCrate(context.Background(), "nonexistent", true)
	if err == nil {
		t.Error("expected error for nonexistent crate")
	}
}

func testClient(t *testing.T, serverURL string) *Client {
	t.Helper()
	headers := map[string]string{
		"User-Agent": "stacktower/1.0 (https://github.com/matzehuels/stacktower)",
	}
	return &Client{
		Client:  integrations.NewClient(cache.NewNullCache(), "crates:", time.Hour, headers),
		baseURL: serverURL,
	}
}
