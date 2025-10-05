package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"stacktower/pkg/httputil"
)

type BaseClient struct {
	HTTP  *http.Client
	Cache *httputil.Cache
}

func (c *BaseClient) FetchWithCache(ctx context.Context, key string, refresh bool, fetch func() error, v any) error {
	if !refresh {
		if ok, _ := c.Cache.Get(key, v); ok {
			return nil
		}
	}

	if err := httputil.RetryWithBackoff(ctx, fetch); err != nil {
		return err
	}

	_ = c.Cache.Set(key, v)
	return nil
}

func (c *BaseClient) DoRequest(ctx context.Context, url string, headers map[string]string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return &httputil.RetryableError{Err: fmt.Errorf("%w: %v", ErrNetwork, err)}
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case resp.StatusCode >= 500:
		return &httputil.RetryableError{Err: fmt.Errorf("%w: %d", ErrNetwork, resp.StatusCode)}
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("%w: %d", ErrNetwork, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	return nil
}
