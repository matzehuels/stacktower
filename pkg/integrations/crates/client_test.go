package crates

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
	if c.baseURL != "https://crates.io/api/v1" {
		t.Errorf("expected base URL %s, got %s", "https://crates.io/api/v1", c.baseURL)
	}
}

func TestClient_FetchCrate(t *testing.T) {
	crateResp := crateResponse{
		Crate: crateData{
			Name:        "serde",
			MaxVersion:  "1.0.0",
			Description: "A serialization framework",
			License:     "MIT",
			Repository:  "https://github.com/serde-rs/serde",
			Downloads:   1000000,
		},
	}
	depsResp := depsResponse{
		Dependencies: []dependency{
			{CrateID: "serde_derive", Kind: "normal", Optional: false},
			{CrateID: "test_dep", Kind: "dev", Optional: false},
			{CrateID: "optional_dep", Kind: "normal", Optional: true},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/crates/serde" {
			json.NewEncoder(w).Encode(crateResp)
		} else if r.URL.Path == "/crates/serde/1.0.0/dependencies" {
			json.NewEncoder(w).Encode(depsResp)
		}
	}))
	defer server.Close()

	c, err := NewClient(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	c.baseURL = server.URL

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
	if info.Dependencies[0] != "serde_derive" {
		t.Errorf("expected serde_derive, got %s", info.Dependencies[0])
	}
}

func TestClient_FetchCrate_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c, err := NewClient(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	c.baseURL = server.URL

	_, err = c.FetchCrate(context.Background(), "nonexistent", true)
	if err == nil {
		t.Error("expected error for nonexistent crate")
	}
}
