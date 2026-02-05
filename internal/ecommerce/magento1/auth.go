package magento1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type tokenResponse struct {
	Token     string `json:"token"`
	TokenType string `json:"token_type"`
	ExpiresIn int    `json:"expires_in"`
}

type authClient struct {
	baseURL      string
	clientID     string
	clientSecret string

	mu          sync.RWMutex
	token       string
	tokenExpiry time.Time
}

func newAuthClient(baseURL, clientID, clientSecret string) *authClient {
	return &authClient{
		baseURL:      baseURL,
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

func (a *authClient) getToken() (string, error) {
	a.mu.RLock()
	if a.token != "" && time.Now().Before(a.tokenExpiry) {
		token := a.token
		a.mu.RUnlock()
		return token, nil
	}
	a.mu.RUnlock()
	return a.refreshToken()
}

func (a *authClient) refreshToken() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Double-check after lock
	if a.token != "" && time.Now().Before(a.tokenExpiry) {
		return a.token, nil
	}

	payload := map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     a.clientID,
		"client_secret": a.clientSecret,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(a.baseURL+"/api/auth/token", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request returned status %d", resp.StatusCode)
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token: %w", err)
	}

	a.token = tokenResp.Token
	// Refresh 5 minutes before expiry
	a.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-300) * time.Second)
	return a.token, nil
}
