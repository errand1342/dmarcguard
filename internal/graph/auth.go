package graph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// TokenResponse represents the OAuth 2.0 token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// Authenticator handles OAuth 2.0 authentication with Microsoft Graph
type Authenticator struct {
	cfg       *Config
	client    *http.Client
	token     string
	expiresAt time.Time
	mu        sync.RWMutex
}

// NewAuthenticator creates a new Authenticator
func NewAuthenticator(cfg *Config) *Authenticator {
	return &Authenticator{
		cfg: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetToken returns a valid access token, refreshing if necessary
func (a *Authenticator) GetToken() (string, error) {
	a.mu.RLock()
	if a.token != "" && time.Now().Before(a.expiresAt.Add(-30*time.Second)) {
		defer a.mu.RUnlock()
		return a.token, nil
	}
	a.mu.RUnlock()

	// Need to refresh token
	return a.refreshToken()
}

// refreshToken obtains a new access token
func (a *Authenticator) refreshToken() (string, error) {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", a.cfg.TenantID)

	data := url.Values{
		"client_id":     {a.cfg.ClientID},
		"scope":         {"https://graph.microsoft.com/.default"},
		"grant_type":    {"client_credentials"},
		"client_secret": {a.cfg.ClientSecret},
	}

	resp, err := a.client.PostForm(tokenURL, data)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, body)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	a.mu.Lock()
	a.token = tokenResp.AccessToken
	a.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	a.mu.Unlock()

	return tokenResp.AccessToken, nil
}

// MakeRequest performs an authenticated HTTP request to Microsoft Graph
func (a *Authenticator) MakeRequest(method, path string, body interface{}) ([]byte, error) {
	token, err := a.GetToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	url := fmt.Sprintf("https://graph.microsoft.com/v1.0%s", path)

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, respBody)
	}

	return respBody, nil
}
