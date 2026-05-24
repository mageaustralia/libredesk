package ecommerce

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// MaxResponseBytes caps how much of a provider response we read into
// memory. Ecommerce REST payloads (a customer + a few orders) are well
// under this; the cap stops a malicious or compromised store — reachable
// precisely because BaseURL is admin-supplied — from streaming an
// unbounded body and OOM-ing the host (it runs on 1.8GB).
const MaxResponseBytes = 8 << 20 // 8 MiB

// HTTPClientOrDefault returns c when non-nil, otherwise a plain client
// with the given timeout. Providers call this so a caller that injects
// the SSRF-guarded client (production) gets it, while direct callers
// (tests) still get a working client. Production build sites always
// inject, so the fallback is not an SSRF hole in normal operation.
func HTTPClientOrDefault(c *http.Client, timeout time.Duration) *http.Client {
	if c != nil {
		return c
	}
	return &http.Client{Timeout: timeout}
}

// HTTPClient is the small base every provider (Magento 2, Shopify,
// WooCommerce, future ones) plugs into to get a consistent request
// pipeline: URL assembly, provider-supplied auth, User-Agent, Accept
// header, error wrapping. Each provider only customises the parts that
// are genuinely platform-specific — auth header shape and (for Shopify)
// retry policy — and delegates the rest.
//
// Magento 1 doesn't use HTTPClient because its doRequest returns
// (body []byte, statusCode int, error) for legacy reasons (the auth
// flow inspects the status to detect token expiry); reshaping it to
// match the rest is out of scope here.
type HTTPClient struct {
	// Name is used as the prefix on wrapped errors so logs stay
	// attributable when multiple providers chain through a single
	// handler (e.g. per-inbox routing).
	Name string

	// BaseURL must include any platform-specific path prefix so callers
	// only pass the endpoint-specific tail. For example:
	//   magento2:    https://store.example.com/rest/V1
	//   shopify:     https://shop.myshopify.com/admin/api/2024-10
	//   woocommerce: https://store.example.com/wp-json/wc/v3
	BaseURL string

	// UserAgent is set on every outbound request when non-empty.
	// Use ecommerce.UserAgent() to derive the standard "libredesk/<v>"
	// string from build info.
	UserAgent string

	// HTTP is the underlying client; supply with a sensible timeout
	// (15-60s is the usual range for ecommerce REST APIs).
	HTTP *http.Client

	// Auth applies the provider's auth scheme directly to the request
	// (Bearer token, X-Shopify-Access-Token, HTTP Basic, etc.). Called
	// per-request so token refresh can happen transparently if the
	// closure captures a token source.
	Auth func(req *http.Request) error
}

// Do builds, authenticates, and dispatches a request without
// interpreting the response. Returns the raw *http.Response so the
// caller can implement retry policy (Shopify's 429 Retry-After loop)
// or stream large response bodies. Callers MUST close resp.Body.
func (c *HTTPClient) Do(ctx context.Context, method, path string, query url.Values, body io.Reader) (*http.Response, error) {
	u := c.BaseURL + path
	if query != nil {
		if encoded := query.Encode(); encoded != "" {
			u += "?" + encoded
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, fmt.Errorf("%s: build request: %w", c.Name, err)
	}
	if c.Auth != nil {
		if err := c.Auth(req); err != nil {
			return nil, fmt.Errorf("%s: auth: %w", c.Name, err)
		}
	}
	req.Header.Set("Accept", "application/json")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: http: %w", c.Name, err)
	}
	resp.Body = capBody(resp.Body)
	return resp, nil
}

// cappedBody bounds reads to MaxResponseBytes while delegating Close to
// the original body, so every consumer of Do/Get is protected from an
// oversized response without each having to remember an io.LimitReader.
type cappedBody struct {
	io.Reader
	io.Closer
}

func capBody(rc io.ReadCloser) io.ReadCloser {
	return cappedBody{Reader: io.LimitReader(rc, MaxResponseBytes), Closer: rc}
}

// Get is the common-case convenience wrapper around Do+ClassifyResponse:
// authenticated GET, returns the body on 2xx, mapped error otherwise.
// Use Do directly if you need control over the response (retry,
// streaming, custom status handling).
func (c *HTTPClient) Get(ctx context.Context, path string, query url.Values) (io.ReadCloser, error) {
	resp, err := c.Do(ctx, "GET", path, query, nil)
	if err != nil {
		return nil, err
	}
	return ClassifyResponse(resp, c.Name)
}
