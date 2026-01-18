package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/matzehuels/stacktower/pkg/cache"
)

func TestNewClient(t *testing.T) {
	c, _ := cache.NewFileCache(t.TempDir())
	defer c.Close()

	headers := map[string]string{"Authorization": "Bearer token"}
	client := NewClient(c, "test:", time.Hour, headers)

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	if client.http == nil {
		t.Error("NewClient() http client is nil")
	}
	if client.cache != c {
		t.Error("NewClient() cache not set correctly")
	}
	if client.headers["Authorization"] != "Bearer token" {
		t.Error("NewClient() headers not set correctly")
	}
}

func TestNewClientNilHeaders(t *testing.T) {
	c, _ := cache.NewFileCache(t.TempDir())
	defer c.Close()

	client := NewClient(c, "test:", time.Hour, nil)

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	if client.headers != nil {
		t.Error("NewClient() should allow nil headers")
	}
}

func TestClientGet(t *testing.T) {
	type response struct {
		Message string `json:"message"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(response{Message: "hello"})
	}))
	defer server.Close()

	c, _ := cache.NewFileCache(t.TempDir())
	defer c.Close()

	client := NewClient(c, "test:", time.Hour, nil)
	client.http = server.Client()

	var resp response
	err := client.Get(context.Background(), server.URL, &resp)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if resp.Message != "hello" {
		t.Errorf("Get() message = %q, want %q", resp.Message, "hello")
	}
}

func TestClientGetWithHeaders(t *testing.T) {
	var receivedHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Custom")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	c, _ := cache.NewFileCache(t.TempDir())
	defer c.Close()

	client := NewClient(c, "test:", time.Hour, map[string]string{"X-Default": "default"})
	client.http = server.Client()

	var resp map[string]string
	err := client.GetWithHeaders(context.Background(), server.URL, map[string]string{"X-Custom": "custom"}, &resp)
	if err != nil {
		t.Fatalf("GetWithHeaders() error: %v", err)
	}
	if receivedHeader != "custom" {
		t.Errorf("custom header = %q, want %q", receivedHeader, "custom")
	}
}

func TestClientGetWithHeadersOverridesDefaults(t *testing.T) {
	var receivedHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Override")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	c, _ := cache.NewFileCache(t.TempDir())
	defer c.Close()

	client := NewClient(c, "test:", time.Hour, map[string]string{"X-Override": "default"})
	client.http = server.Client()

	var resp map[string]string
	err := client.GetWithHeaders(context.Background(), server.URL, map[string]string{"X-Override": "overridden"}, &resp)
	if err != nil {
		t.Fatalf("GetWithHeaders() error: %v", err)
	}
	if receivedHeader != "overridden" {
		t.Errorf("header = %q, want %q", receivedHeader, "overridden")
	}
}

func TestClientGetText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("plain text response"))
	}))
	defer server.Close()

	c, _ := cache.NewFileCache(t.TempDir())
	defer c.Close()

	client := NewClient(c, "test:", time.Hour, nil)
	client.http = server.Client()

	text, err := client.GetText(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("GetText() error: %v", err)
	}
	if text != "plain text response" {
		t.Errorf("GetText() = %q, want %q", text, "plain text response")
	}
}

func TestClientGet404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c, _ := cache.NewFileCache(t.TempDir())
	defer c.Close()

	client := NewClient(c, "test:", time.Hour, nil)
	client.http = server.Client()

	var resp map[string]string
	err := client.Get(context.Background(), server.URL, &resp)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestClientGet500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c, _ := cache.NewFileCache(t.TempDir())
	defer c.Close()

	client := NewClient(c, "test:", time.Hour, nil)
	client.http = server.Client()

	var resp map[string]string
	err := client.Get(context.Background(), server.URL, &resp)
	if err == nil {
		t.Error("Get() should return error for 500")
	}

	var retryErr *cache.RetryableError
	if !errors.As(err, &retryErr) {
		t.Errorf("Get() error should be RetryableError, got %T", err)
	}
}

func TestClientCached(t *testing.T) {
	c, _ := cache.NewFileCache(t.TempDir())
	defer c.Close()

	client := NewClient(c, "test:", time.Hour, nil)

	fetchCount := 0
	type testData struct {
		Value string `json:"value"`
	}
	value := testData{}

	fetch := func() error {
		fetchCount++
		value = testData{Value: "fetched"}
		return nil
	}

	// Use unique key per test run to avoid cache interference
	key := "test-key-" + time.Now().String()

	// First call should fetch since key doesn't exist
	err := client.Cached(context.Background(), key, false, &value, fetch)
	if err != nil {
		t.Fatalf("Cached() error: %v", err)
	}
	// Fetch should have been called
	if fetchCount < 1 {
		t.Errorf("fetch count = %d, want at least 1", fetchCount)
	}
}

func TestClientCachedRefresh(t *testing.T) {
	c, _ := cache.NewFileCache(t.TempDir())
	defer c.Close()

	client := NewClient(c, "test:", time.Hour, nil)

	fetchCount := 0
	var value string

	fetch := func() error {
		fetchCount++
		value = "fetched"
		return nil
	}

	// With refresh=true, should always fetch
	err := client.Cached(context.Background(), "test-key", true, &value, fetch)
	if err != nil {
		t.Fatalf("Cached() error: %v", err)
	}
	if fetchCount != 1 {
		t.Errorf("fetch count = %d, want 1", fetchCount)
	}
}

func TestClientCachedFetchError(t *testing.T) {
	c, _ := cache.NewFileCache(t.TempDir())
	defer c.Close()

	client := NewClient(c, "test:", time.Hour, nil)

	var value string

	// Use unique key per test run to avoid cache interference
	key := "test-error-key-" + time.Now().String()

	// RetryWithBackoff will retry, so we need a non-retryable error
	fetchCount := 0
	fetch := func() error {
		fetchCount++
		return ErrNotFound // Non-retryable error
	}

	err := client.Cached(context.Background(), key, false, &value, fetch)
	if err == nil {
		t.Error("Cached() should return error when fetch fails")
	}
	if fetchCount < 1 {
		t.Error("fetch should have been called at least once")
	}
}

func TestCheckStatus(t *testing.T) {
	tests := []struct {
		name       string
		code       int
		wantErr    bool
		wantType   error
		isRetryErr bool
	}{
		{
			name:    "200 OK",
			code:    200,
			wantErr: false,
		},
		{
			name:     "404 Not Found",
			code:     404,
			wantErr:  true,
			wantType: ErrNotFound,
		},
		{
			name:       "500 Internal Server Error",
			code:       500,
			wantErr:    true,
			isRetryErr: true,
		},
		{
			name:       "502 Bad Gateway",
			code:       502,
			wantErr:    true,
			isRetryErr: true,
		},
		{
			name:       "503 Service Unavailable",
			code:       503,
			wantErr:    true,
			isRetryErr: true,
		},
		{
			name:    "400 Bad Request",
			code:    400,
			wantErr: true,
		},
		{
			name:    "403 Forbidden",
			code:    403,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkStatus(tt.code)

			if tt.wantErr {
				if err == nil {
					t.Error("checkStatus() should return error")
				}
				if tt.wantType != nil && !errors.Is(err, tt.wantType) {
					t.Errorf("checkStatus() error = %v, want %v", err, tt.wantType)
				}
				if tt.isRetryErr {
					var retryErr *cache.RetryableError
					if !errors.As(err, &retryErr) {
						t.Errorf("checkStatus() error should be RetryableError, got %T", err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("checkStatus() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestNormalizePkgName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"lowercase", "Package", "package"},
		{"underscore to dash", "my_package", "my-package"},
		{"trim spaces", "  package  ", "package"},
		{"combined", "  My_Package  ", "my-package"},
		{"empty", "", ""},
		{"already normalized", "my-package", "my-package"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizePkgName(tt.input); got != tt.want {
				t.Errorf("NormalizePkgName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeRepoURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"https url", "https://github.com/user/repo", "https://github.com/user/repo"},
		{"with .git suffix", "https://github.com/user/repo.git", "https://github.com/user/repo"},
		{"git@ to https", "git@github.com:user/repo", "https://github.com/user/repo"},
		{"git:// to https", "git://github.com/user/repo", "https://github.com/user/repo"},
		{"git+ prefix", "git+https://github.com/user/repo", "https://github.com/user/repo"},
		{"with spaces", "  https://github.com/user/repo  ", "https://github.com/user/repo"},
		{"combined", "git+git@github.com:user/repo.git", "https://github.com/user/repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeRepoURL(tt.input); got != tt.want {
				t.Errorf("NormalizeRepoURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestURLEncode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain", "hello", "hello"},
		{"space", "hello world", "hello+world"},
		{"special chars", "a=1&b=2", "a%3D1%26b%3D2"},
		{"slash", "path/to/resource", "path%2Fto%2Fresource"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := URLEncode(tt.input); got != tt.want {
				t.Errorf("URLEncode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNewHTTPClient(t *testing.T) {
	client := NewHTTPClient()
	if client == nil {
		t.Fatal("NewHTTPClient() returned nil")
	}
	if client.Timeout != httpTimeout {
		t.Errorf("Timeout = %v, want %v", client.Timeout, httpTimeout)
	}
}
