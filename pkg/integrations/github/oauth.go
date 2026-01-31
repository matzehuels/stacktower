package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultClientID is the OAuth App Client ID for Stacktower.
// This is public and safe to commit - only the Client Secret must be kept private.
// The Device Flow doesn't require a secret, only the Client ID.
//
// To use your own OAuth App, set GITHUB_CLIENT_ID env var.
const DefaultClientID = "Ov23liyPM58WU6hMeP7E"

// OAuthClient handles GitHub OAuth operations.
type OAuthClient struct {
	config     OAuthConfig
	httpClient *http.Client
}

// NewOAuthClient creates a new OAuth client.
func NewOAuthClient(config OAuthConfig) *OAuthClient {
	return &OAuthClient{
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// AuthorizationURL returns the GitHub OAuth authorization URL.
func (c *OAuthClient) AuthorizationURL(state string) string {
	params := url.Values{
		"client_id":    {c.config.ClientID},
		"redirect_uri": {c.config.RedirectURI},
		"scope":        {"read:user user:email repo"},
		"state":        {state},
	}
	return "https://github.com/login/oauth/authorize?" + params.Encode()
}

// ExchangeCode exchanges an authorization code for an access token.
func (c *OAuthClient) ExchangeCode(code string) (*OAuthToken, error) {
	data := url.Values{
		"client_id":     {c.config.ClientID},
		"client_secret": {c.config.ClientSecret},
		"code":          {code},
		"redirect_uri":  {c.config.RedirectURI},
	}

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("%s: %s", result.Error, result.ErrorDesc)
	}

	return &OAuthToken{
		AccessToken: result.AccessToken,
		TokenType:   result.TokenType,
		Scope:       result.Scope,
	}, nil
}

// DeviceCodeResponse contains the response from requesting a device code.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// RequestDeviceCode initiates the device authorization flow.
// The user must visit the VerificationURI and enter the UserCode.
func (c *OAuthClient) RequestDeviceCode(ctx context.Context) (*DeviceCodeResponse, error) {
	data := url.Values{
		"client_id": {c.config.ClientID},
		"scope":     {"read:user user:email repo"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://github.com/login/device/code", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	var result DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// PollForToken polls GitHub for the access token after user authorization.
// It respects the interval from the device code response.
// Returns the token when authorized, or an error if expired/denied.
func (c *OAuthClient) PollForToken(ctx context.Context, deviceCode string, interval int) (*OAuthToken, error) {
	if interval < 5 {
		interval = 5 // GitHub minimum interval
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			token, err := c.checkDeviceToken(ctx, deviceCode)
			if err != nil {
				// Check if it's a "still waiting" error
				if strings.Contains(err.Error(), "authorization_pending") {
					continue // Keep polling
				}
				if strings.Contains(err.Error(), "slow_down") {
					// Increase interval and continue
					ticker.Reset(time.Duration(interval+5) * time.Second)
					continue
				}
				return nil, err // Real error (expired, denied, etc.)
			}
			return token, nil
		}
	}
}

// checkDeviceToken attempts to exchange the device code for a token.
func (c *OAuthClient) checkDeviceToken(ctx context.Context, deviceCode string) (*OAuthToken, error) {
	data := url.Values{
		"client_id":   {c.config.ClientID},
		"device_code": {deviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("%s: %s", result.Error, result.ErrorDesc)
	}

	return &OAuthToken{
		AccessToken: result.AccessToken,
		TokenType:   result.TokenType,
		Scope:       result.Scope,
	}, nil
}
