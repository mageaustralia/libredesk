package maho

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/ecommerce"
	"github.com/zerodha/logf"
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
	userAgent    string

	httpClient *http.Client

	mu          sync.RWMutex
	token       string
	tokenExpiry time.Time

	lo *logf.Logger
}

// jwtPattern matches a JWT-shaped string (three base64-url chunks separated by dots).
var jwtPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`)

func newAuthClient(baseURL, clientID, clientSecret, userAgent string, httpClient *http.Client, lo *logf.Logger) *authClient {
	return &authClient{
		baseURL:      baseURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		userAgent:    userAgent,
		httpClient:   ecommerce.HTTPClientOrDefault(httpClient, 30*time.Second),
		lo:           lo,
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

	// Maho API Platform v2 supports OAuth2 client_credentials grant for
	// service integrations (the human-user customer grant uses email/password).
	// LibreDesk is an integration, so we use client_credentials.
	payload := map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     a.clientID,
		"client_secret": a.clientSecret,
	}
	body, _ := json.Marshal(payload)

	tokenURL := a.baseURL + "/api/rest/v2/auth/token"
	a.lo.Debug("requesting token", "url", tokenURL)

	req, err := http.NewRequest(http.MethodPost, tokenURL, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("build POST %s failed: %w", tokenURL, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", a.userAgent)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("POST %s failed: %w", tokenURL, err)
	}
	defer resp.Body.Close()

	// Cap the read: the token endpoint is admin-configured, so a hostile
	// or compromised target could otherwise stream an unbounded body.
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, ecommerce.MaxResponseBytes))

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyStr := string(respBody)
		if len(bodyStr) > 200 {
			bodyStr = bodyStr[:200] + "..."
		}
		a.lo.Warn("token request failed", "status", resp.StatusCode)
		return "", fmt.Errorf("POST %s returned %d: %s", tokenURL, resp.StatusCode, bodyStr)
	}

	// NOTE: never log respBody here — on the success path it contains the
	// bearer JWT, which must not land in logs.
	var tokenResp tokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	tokenStr := tokenResp.Token

	// Fallback: if the documented field name changed, scan the JSON for any
	// string value that looks like a JWT (three base64-url chunks split by dots).
	if tokenStr == "" {
		var raw map[string]interface{}
		if err := json.Unmarshal(respBody, &raw); err == nil {
			for k, v := range raw {
				if s, ok := v.(string); ok && jwtPattern.MatchString(s) {
					a.lo.Warn("token field 'token' empty; using JWT-shaped value from alternate field", "field", k)
					tokenStr = s
					break
				}
			}
		}
	}

	if tokenStr == "" {
		preview := string(respBody)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return "", fmt.Errorf("token response did not contain a JWT-shaped token; body=%s", strings.TrimSpace(preview))
	}

	a.lo.Info("token obtained", "expires_in_seconds", tokenResp.ExpiresIn)
	a.token = tokenStr

	// Default to a 1-hour TTL if expires_in isn't provided; refresh 5 minutes
	// before expiry. (Many JWT auth endpoints omit expires_in.)
	expSec := tokenResp.ExpiresIn
	if expSec <= 0 {
		expSec = 3600
	}
	a.tokenExpiry = time.Now().Add(time.Duration(expSec-300) * time.Second)
	return a.token, nil
}
