package ecommerce

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// IsAuthError reports whether a status code from an upstream ecommerce
// platform indicates bad credentials. 401 and 403 are both treated as auth
// failures — most platforms (Maho, Shopify, WooCommerce, Magento 2) return
// 403 when the credentials are valid but lack a specific scope, which is
// indistinguishable from "wrong key" from our side and produces the same
// user-facing remediation ("re-check your API key / token scopes").
func IsAuthError(statusCode int) bool {
	return statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden
}

// ClassifyResponse maps a provider HTTP response to the shared error
// vocabulary (ErrNotFound, ErrUnauthorized) or a wrapped fallthrough for
// other non-2xx codes. On the 2xx path it returns the body untouched —
// caller is responsible for closing it. On the error path the body is
// closed before returning.
//
// providerName is used as the prefix on the fallthrough error message so
// logs stay attributable when the same handler chains through multiple
// providers (e.g. per-inbox routing).
//
// 429 is intentionally NOT handled here — rate-limit retry policy is
// provider-specific (Retry-After parsing for Shopify, backoff jitter
// elsewhere), so callers must handle 429 BEFORE delegating to this helper.
func ClassifyResponse(resp *http.Response, providerName string) (io.ReadCloser, error) {
	switch resp.StatusCode {
	case http.StatusOK:
		return resp.Body, nil
	case http.StatusNotFound:
		resp.Body.Close()
		return nil, ErrNotFound
	}
	if IsAuthError(resp.StatusCode) {
		resp.Body.Close()
		return nil, ErrUnauthorized
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	resp.Body.Close()
	return nil, fmt.Errorf("%s: HTTP %d: %s", providerName, resp.StatusCode, strings.TrimSpace(string(body)))
}
