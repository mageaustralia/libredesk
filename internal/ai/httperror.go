package ai

import "net/http"

// IsAuthError reports whether an HTTP status code from an upstream LLM
// provider indicates a bad/missing API key. OpenAI and OpenRouter both
// return 401 for invalid keys; we don't treat 403 as auth here because
// OpenRouter uses 403 for "model not available on this key" which is
// semantically different (the key is fine, the request isn't).
func IsAuthError(statusCode int) bool {
	return statusCode == http.StatusUnauthorized
}
