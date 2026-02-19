package urlutil

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// IsInternalURL checks if a URL points to an internal/private network address.
// Returns true if the URL should be blocked to prevent SSRF attacks.
func IsInternalURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return true
	}

	host := u.Hostname()
	if host == "" {
		return true
	}

	// Block well-known internal hostnames.
	lower := strings.ToLower(host)
	if lower == "localhost" ||
		strings.HasSuffix(lower, ".local") ||
		strings.HasSuffix(lower, ".internal") ||
		lower == "metadata.google.internal" {
		return true
	}

	// Resolve and check IP addresses.
	ips, err := net.LookupIP(host)
	if err != nil {
		// If we can't resolve, check if it's a direct IP.
		ip := net.ParseIP(host)
		if ip == nil {
			return true // Can't resolve hostname
		}
		return isPrivateIP(ip)
	}

	for _, ip := range ips {
		if isPrivateIP(ip) {
			return true
		}
	}

	return false
}

func isPrivateIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified()
}

// ValidateExternalURL returns an error if the URL is internal or invalid.
func ValidateExternalURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("empty URL")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("only http/https URLs are allowed")
	}
	if IsInternalURL(rawURL) {
		return fmt.Errorf("URL points to an internal or private network address")
	}
	return nil
}
