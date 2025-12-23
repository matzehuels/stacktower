package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/matzehuels/stacktower/pkg/httputil"
)

// Client provides shared HTTP functionality for all registry API clients.
// It handles caching, retry logic, and common request headers.
//
// Client is safe for concurrent use by multiple goroutines.
// The underlying HTTP client, cache, and headers are all goroutine-safe.
//
// Zero values: Do not use an uninitialized Client; always create via [NewClient].
type Client struct {
	http    *http.Client
	cache   *httputil.Cache
	headers map[string]string
}

// NewClient creates a Client with the given cache and default headers.
// Headers are applied to all requests made through this client.
//
// Parameters:
//   - cache: Cache instance for storing responses (must not be nil). Create with
//     [NewCacheWithNamespace] for registry-specific caching.
//   - headers: Default HTTP headers for all requests. Pass nil if no default headers
//     are needed. Common examples: "Authorization", "User-Agent", "Accept".
//
// The returned Client is safe for concurrent use by multiple goroutines.
// Panics if cache is nil.
func NewClient(cache *httputil.Cache, headers map[string]string) *Client {
	return &Client{
		http:    NewHTTPClient(),
		cache:   cache,
		headers: headers,
	}
}

// Cached retrieves a value from cache or executes fetch and caches the result.
// If refresh is true, the cache is bypassed and fetch is always called.
//
// Parameters:
//   - ctx: Context for cancellation. If cancelled, fetch is not executed and returns ctx.Err().
//   - key: Cache key (usually package name or coordinate). Must not be empty.
//   - refresh: If true, bypass cache and always call fetch. If false, try cache first.
//   - v: Pointer to store the result. Must be a non-nil pointer to a JSON-serializable type.
//   - fetch: Function to fetch data and populate v. Called with retry on transient failures.
//
// Behavior:
//  1. If refresh=false and cache hit: returns nil immediately with v populated
//  2. If cache miss or refresh=true: calls fetch with automatic retry on [httputil.RetryableError]
//  3. On successful fetch: stores result in cache (ignoring cache write errors)
//
// The fetch function should populate v and return nil on success, or return an error.
// Network errors should be wrapped with [httputil.Retryable] to enable retry.
//
// Returns:
//   - nil on success (v is populated)
//   - error from fetch if it fails (v may be partially populated)
//   - ctx.Err() if context is cancelled
//
// This method is safe for concurrent use on the same Client.
func (c *Client) Cached(ctx context.Context, key string, refresh bool, v any, fetch func() error) error {
	if !refresh {
		if ok, _ := c.cache.Get(key, v); ok {
			return nil
		}
	}
	if err := httputil.RetryWithBackoff(ctx, fetch); err != nil {
		return err
	}
	_ = c.cache.Set(key, v)
	return nil
}

// Get performs an HTTP GET request and JSON-decodes the response into v.
// It uses the client's default headers and handles retries automatically.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - url: Full URL to request (must be absolute URL with scheme)
//   - v: Pointer to store decoded JSON response (must be non-nil)
//
// Returns:
//   - [ErrNotFound] for HTTP 404 responses
//   - [ErrNetwork] wrapped with [httputil.RetryableError] for HTTP 5xx responses
//   - [ErrNetwork] for connection failures and timeouts
//   - json decoding errors if response is not valid JSON
//
// This method is safe for concurrent use on the same Client.
func (c *Client) Get(ctx context.Context, url string, v any) error {
	return c.GetWithHeaders(ctx, url, nil, v)
}

// GetWithHeaders performs an HTTP GET with additional headers merged with defaults.
// Request-specific headers override client defaults for the same key.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - url: Full URL to request (must be absolute URL with scheme)
//   - headers: Additional headers for this request only (may be nil). Headers with the
//     same key as client defaults will override the default value for this request.
//   - v: Pointer to store decoded JSON response (must be non-nil)
//
// Example:
//
//	err := client.GetWithHeaders(ctx, url, map[string]string{"X-Custom": "value"}, &resp)
//
// Returns the same errors as [Get].
// This method is safe for concurrent use on the same Client.
func (c *Client) GetWithHeaders(ctx context.Context, url string, headers map[string]string, v any) error {
	body, err := c.doRequest(ctx, url, headers)
	if err != nil {
		return err
	}
	defer body.Close()
	return json.NewDecoder(body).Decode(v)
}

// GetText performs an HTTP GET request and returns the response body as a string.
// Useful for non-JSON endpoints like go.mod files or plain text responses.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - url: Full URL to request (must be absolute URL with scheme)
//
// The entire response body is read into memory. Use caution with large responses.
// For files larger than a few MB, consider streaming with a custom implementation.
//
// Returns:
//   - The response body as a string
//   - [ErrNotFound] for HTTP 404 responses
//   - [ErrNetwork] for connection failures, timeouts, and HTTP 5xx responses
//   - io errors if reading the response body fails
//
// This method is safe for concurrent use on the same Client.
func (c *Client) GetText(ctx context.Context, url string) (string, error) {
	body, err := c.doRequest(ctx, url, nil)
	if err != nil {
		return "", err
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	return string(data), err
}

func (c *Client) doRequest(ctx context.Context, url string, headers map[string]string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, httputil.Retryable(fmt.Errorf("%w: %v", ErrNetwork, err))
	}

	if err := checkStatus(resp.StatusCode); err != nil {
		resp.Body.Close()
		return nil, err
	}
	return resp.Body, nil
}

func checkStatus(code int) error {
	switch {
	case code == http.StatusOK:
		return nil
	case code == http.StatusNotFound:
		return ErrNotFound
	case code >= 500:
		return httputil.Retryable(fmt.Errorf("%w: status %d", ErrNetwork, code))
	default:
		return fmt.Errorf("%w: status %d", ErrNetwork, code)
	}
}
