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
type Client struct {
	http    *http.Client
	cache   *httputil.Cache
	headers map[string]string
}

// NewClient creates a Client with the given cache and default headers.
// Headers are applied to all requests made through this client.
// Pass nil for headers if no default headers are needed.
func NewClient(cache *httputil.Cache, headers map[string]string) *Client {
	return &Client{
		http:    NewHTTPClient(),
		cache:   cache,
		headers: headers,
	}
}

// Cached retrieves a value from cache or executes fetch and caches the result.
// If refresh is true, the cache is bypassed and fetch is always called.
// The fetch function should populate v; on success, v is stored in the cache.
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
func (c *Client) Get(ctx context.Context, url string, v any) error {
	return c.GetWithHeaders(ctx, url, nil, v)
}

// GetWithHeaders performs an HTTP GET with additional headers merged with defaults.
// Request-specific headers override client defaults for the same key.
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
		return nil, &httputil.RetryableError{Err: fmt.Errorf("%w: %v", ErrNetwork, err)}
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
		return &httputil.RetryableError{Err: fmt.Errorf("%w: status %d", ErrNetwork, code)}
	default:
		return fmt.Errorf("%w: status %d", ErrNetwork, code)
	}
}
