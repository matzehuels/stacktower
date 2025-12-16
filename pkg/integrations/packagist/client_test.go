package packagist

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/matzehuels/stacktower/pkg/integrations"
)

func TestNewClient(t *testing.T) {
	c, err := NewClient(time.Hour)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if c.Client == nil {
		t.Error("expected client to be initialized")
	}
}

func TestFetchPackage_Success(t *testing.T) {
	vStable := p2Version{
		Name:        "vendor/package",
		Version:     "1.2.3",
		Description: "A great package",
		Homepage:    "https://example.com",
		License:     []string{"MIT"},
		Require: map[string]string{
			"php":                  ">=8.0",
			"ext-json":             "*",
			"lib-icu":              "*",
			"composer-plugin-api":  "^2.0",
			"composer-runtime-api": "^2.2",
			"vendor/dep":           "^0.9.0",
			"noslash":              "1.0.0",
		},
		Source: struct {
			URL string `json:"url"`
		}{URL: "git+https://github.com/user/repo.git"},
		Authors: []struct {
			Name string `json:"name"`
		}{{Name: "  Jane Doe  "}},
	}
	vDev := p2Version{Name: "vendor/package", Version: "1.3.0-dev"}
	payload := p2Response{Packages: map[string][]p2Version{
		"vendor/package": {vDev, vStable},
	}}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/p2/vendor/package.json" {
			_ = json.NewEncoder(w).Encode(payload)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := testClient(t, server.URL)

	info, err := c.FetchPackage(context.Background(), "Vendor/Package", true)
	if err != nil {
		t.Fatalf("FetchPackage error: %v", err)
	}

	if info.Name != "vendor/package" {
		t.Errorf("want name vendor/package, got %s", info.Name)
	}
	if info.Version != "1.2.3" {
		t.Errorf("want version 1.2.3, got %s", info.Version)
	}
	if info.Description != "A great package" {
		t.Errorf("unexpected description: %s", info.Description)
	}
	if info.Author != "Jane Doe" {
		t.Errorf("want author 'Jane Doe', got %q", info.Author)
	}
	if info.Repository != "https://github.com/user/repo" {
		t.Errorf("unexpected repository url: %s", info.Repository)
	}
	if info.HomePage != "https://example.com" {
		t.Errorf("unexpected homepage: %s", info.HomePage)
	}
	if len(info.Dependencies) != 1 || info.Dependencies[0] != "vendor/dep" {
		t.Errorf("unexpected dependencies: %#v", info.Dependencies)
	}
}

func TestFetchPackage_NotFound(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	c := testClient(t, server.URL)

	if _, err := c.FetchPackage(context.Background(), "missing/pkg", true); err == nil {
		t.Fatalf("expected error for 404, got nil")
	}
}

func TestNormalizeName(t *testing.T) {
	if got := strings.ToLower(strings.TrimSpace("  VenDor/PackAge  ")); got != "vendor/package" {
		t.Errorf("normalizeName unexpected: %q", got)
	}
}

func TestNormalizeRepoURL(t *testing.T) {
	cases := []struct{ in, want string }{
		{"git+https://github.com/user/repo.git", "https://github.com/user/repo"},
		{"git://github.com/user/repo.git", "https://github.com/user/repo"},
		{"git@github.com:user/repo.git", "https://github.com/user/repo"},
		{"https://github.com/user/repo", "https://github.com/user/repo"},
		{"", ""},
	}
	for _, c := range cases {
		if got := integrations.NormalizeRepoURL(c.in); got != c.want {
			t.Errorf("NormalizeRepoURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFilterDeps(t *testing.T) {
	in := map[string]string{
		"php":                  ">=8.1",
		"ext-json":             "*",
		"lib-icu":              "*",
		"composer-plugin-api":  "^2",
		"composer-runtime-api": "^2",
		"vendor/dep1":          "^1.0",
		"Vendor/Dep2":          "*",
		"no/slash?":            "1.0",
		"noslash":              "*",
	}
	got := filterDeps(in)
	if _, ok := got["vendor/dep1"]; !ok {
		t.Errorf("missing vendor/dep1 in %v", got)
	}
	if _, ok := got["vendor/dep2"]; !ok {
		t.Errorf("missing vendor/dep2 in %v", got)
	}
	if _, ok := got["php"]; ok {
		t.Errorf("php should be filtered out")
	}
	if _, ok := got["ext-json"]; ok {
		t.Errorf("ext-json should be filtered out")
	}
	if _, ok := got["lib-icu"]; ok {
		t.Errorf("lib-icu should be filtered out")
	}
	if _, ok := got["composer-plugin-api"]; ok {
		t.Errorf("composer-plugin-api should be filtered out")
	}
	if _, ok := got["composer-runtime-api"]; ok {
		t.Errorf("composer-runtime-api should be filtered out")
	}
}

func TestLatestStable(t *testing.T) {
	versions := []p2Version{
		{Version: "2-dev"},
		{Version: "v3"},
		{Version: "1.5.0"},
	}
	got := latestStable(versions)
	if got.Version != "1.5.0" {
		t.Errorf("latestStable = %s", got.Version)
	}

	versions = []p2Version{{Version: "dev-main"}, {Version: "v2"}}
	got = latestStable(versions)
	if got.Version != "dev-main" {
		t.Errorf("latestStable fallback = %s", got.Version)
	}
}

func TestP2Version_UnmarshalJSON(t *testing.T) {
	raw := `{
        "name": "vendor/pkg",
        "version": "1.0.0",
        "description": "d",
        "homepage": "h",
        "license": "BSD-3-Clause",
        "require": {"vendor/dep": "^1", "php": ">=8.0", "weird": 5},
        "source": {"url": "https://example.com/repo.git"},
        "authors": [{"name": "Ann"}]
    }`
	var v p2Version
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(v.License) != 1 || v.License[0] != "BSD-3-Clause" {
		t.Errorf("unexpected license: %#v", v.License)
	}
	if v.Require["vendor/dep"] != "^1" || v.Require["php"] != ">=8.0" {
		t.Errorf("unexpected require: %#v", v.Require)
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
