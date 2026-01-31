package goproxy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/matzehuels/stacktower/pkg/cache"

	"github.com/matzehuels/stacktower/pkg/integrations"
)

func TestEscapePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"github.com/gin-gonic/gin", "github.com/gin-gonic/gin"},
		{"github.com/Azure/azure-sdk-for-go", "github.com/!azure/azure-sdk-for-go"},
		{"golang.org/x/sync", "golang.org/x/sync"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := escapePath(tt.input); got != tt.want {
				t.Errorf("escapePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseGoMod(t *testing.T) {
	content := `module github.com/example/myapp

go 1.21

require (
	github.com/gin-gonic/gin v1.9.0
	github.com/spf13/cobra v1.7.0
	golang.org/x/sync v0.3.0 // indirect
)

require github.com/stretchr/testify v1.8.0
`

	deps, err := parseGoMod(strings.NewReader(content))
	if err != nil {
		t.Fatalf("parseGoMod failed: %v", err)
	}

	// Should have 3 direct deps (indirect ones filtered out)
	if len(deps) != 3 {
		t.Errorf("expected 3 deps, got %d: %v", len(deps), deps)
	}

	want := map[string]bool{
		"github.com/gin-gonic/gin":    true,
		"github.com/spf13/cobra":      true,
		"github.com/stretchr/testify": true,
	}
	for _, dep := range deps {
		if !want[dep] {
			t.Errorf("unexpected dep: %s", dep)
		}
	}
}

func TestParseRequireLine(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"github.com/gin-gonic/gin v1.9.0", "github.com/gin-gonic/gin"},
		{"golang.org/x/sync v0.3.0 // indirect", ""},
		{"github.com/pkg/errors v0.9.1 // some comment", "github.com/pkg/errors"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := parseRequireLine(tt.line); got != tt.want {
				t.Errorf("parseRequireLine(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestClient_FetchModule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/github.com/example/mylib/@latest":
			json.NewEncoder(w).Encode(latestResponse{Version: "v1.2.3"})
		case "/github.com/example/mylib/@v/v1.2.3.mod":
			w.Write([]byte(`module github.com/example/mylib

go 1.21

require github.com/pkg/errors v0.9.1
`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c := testClient(t, server.URL)

	info, err := c.FetchModule(context.Background(), "github.com/example/mylib", true)
	if err != nil {
		t.Fatalf("FetchModule failed: %v", err)
	}

	if info.Path != "github.com/example/mylib" {
		t.Errorf("expected path github.com/example/mylib, got %s", info.Path)
	}
	if info.Version != "v1.2.3" {
		t.Errorf("expected version v1.2.3, got %s", info.Version)
	}
	if len(info.Dependencies) != 1 {
		t.Errorf("expected 1 dep, got %d", len(info.Dependencies))
	}
	if len(info.Dependencies) > 0 && info.Dependencies[0] != "github.com/pkg/errors" {
		t.Errorf("expected github.com/pkg/errors, got %s", info.Dependencies[0])
	}
}

func TestClient_FetchModule_NotFound(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	c := testClient(t, server.URL)

	_, err := c.FetchModule(context.Background(), "github.com/missing/module", true)
	if err == nil {
		t.Fatal("expected error for missing module")
	}
}

func testClient(t *testing.T, serverURL string) *Client {
	t.Helper()
	return &Client{
		Client:  integrations.NewClient(cache.NewNullCache(), "goproxy:", time.Hour, nil),
		baseURL: serverURL,
	}
}
