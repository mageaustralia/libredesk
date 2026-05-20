// Package license implements offline, signature-verified license keys
// for HelperIQ Pro extensions.
//
// Why offline: B2B customers self-host. Pinging a license server would
// (a) introduce a network dependency they often firewall against, and
// (b) leak usage data we have no business collecting. Offline verification
// using a public key embedded in the binary gives us the same gate
// with none of those costs.
//
// Key shape: a base64url-encoded payload + Ed25519 signature, joined by
// a dot, similar to a JWT but trimmed of all the parts we don't need.
//
//   <base64url(payload)>.<base64url(sig)>
//
// Payload is JSON:
//
//   {
//     "customer":  "Mage Australia",
//     "email":     "support@mageaustralia.com.au",
//     "issued_at": 1747353600,
//     "expires_at": 1778889600,  // optional — 0 means perpetual
//     "features":  ["ecommerce.shopify", "ecommerce.magento2", "ai.bring_your_own"],
//     "instance_id": ""          // optional — bind to a specific install
//   }
//
// Verification flow:
//  1. Admin pastes the license key into Admin → License (or sets
//     LIBREDESK_LICENSE env var, or licenses.toml in config dir).
//  2. On startup AND on settings.license_key change, package license
//     parses the key, verifies the Ed25519 signature against the
//     compiled-in public key, and caches the decoded payload.
//  3. Feature-gated code calls RequireFeature(license.FeatureFoo).
//     Returns nil if the feature is in the current license's `features`
//     list, ErrLicenseRequired otherwise.
//
// Threat model: this is a *commercial honesty* gate, not security. The
// source is AGPL; a determined customer can patch RequireFeature to
// `return nil` in 30 seconds. The goal is to make the paying path the
// easy path. The vast majority of customers run unmodified releases.
//
// Public key generation (one-off, store private key in 1Password or
// equivalent — losing it means re-issuing every license):
//
//	priv, pub, _ := ed25519.GenerateKey(rand.Reader)
//	fmt.Printf("PRIVATE: %s\n", base64.StdEncoding.EncodeToString(priv))
//	fmt.Printf("PUBLIC:  %s\n", base64.StdEncoding.EncodeToString(pub))
//
// Replace the publicKeyB64 constant below with the public key. Sign new
// licenses with: tools/license-sign/main.go (separate CLI, kept in a
// private repo so the private key is never in the OSS tree).
package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// publicKeyB64 is the Ed25519 public key compiled into every HelperIQ
// release. The matching private key lives in HelperIQ Inc.'s vault. To
// rotate, generate a new pair, replace this constant, recompile the
// release, and re-issue every active license — there's no in-band key
// rotation.
//
// THIS IS A PLACEHOLDER. Replace with the real public key before any
// release that gates features. With this empty value, every
// RequireFeature() call will return ErrLicenseRequired (which is the
// safe fallback — features stay off until a real key is wired in).
const publicKeyB64 = "" // ed25519 public key, base64

// Feature constants — string-typed so license payloads remain
// human-readable and forward-compatible. Adding a new feature is
// strictly additive: old licenses keep working, they just don't grant
// the new feature.
type Feature string

const (
	FeatureEcommerceShopify     Feature = "ecommerce.shopify"
	FeatureEcommerceMagento2    Feature = "ecommerce.magento2"
	FeatureEcommerceWooCommerce Feature = "ecommerce.woocommerce"
	FeatureAIBringYourOwn       Feature = "ai.bring_your_own"
	FeatureSSO                  Feature = "auth.sso"
	FeatureAuditLogExport       Feature = "audit.export"
)

// ErrLicenseRequired is returned by RequireFeature when no valid license
// covers the requested feature. Callers should surface this to the admin
// UI with a "Upgrade to HelperIQ Pro to enable X" message — NOT as a
// generic 500.
var ErrLicenseRequired = errors.New("helperiq pro license required for this feature")

// ErrLicenseInvalid is returned by Load() when the supplied key doesn't
// parse or fails signature verification. Used at admin-UI save time so
// the bad key can be rejected with a clean error instead of silently
// failing later.
var ErrLicenseInvalid = errors.New("license key is invalid or tampered with")

// Payload is the decoded shape of a license. Exported so admin tooling
// can read fields like ExpiresAt to render "expires in N days" warnings.
type Payload struct {
	Customer   string    `json:"customer"`
	Email      string    `json:"email"`
	IssuedAt   int64     `json:"issued_at"`
	ExpiresAt  int64     `json:"expires_at,omitempty"` // 0 = perpetual
	Features   []Feature `json:"features"`
	InstanceID string    `json:"instance_id,omitempty"` // 0-length = bind to anything
}

// IsExpired reports whether the license has passed its ExpiresAt. A
// payload with ExpiresAt == 0 is perpetual (never expires) — the rest
// of the codebase relies on this fact to ship "lifetime" licenses for
// founding customers.
func (p *Payload) IsExpired() bool {
	if p.ExpiresAt == 0 {
		return false
	}
	return time.Now().Unix() > p.ExpiresAt
}

// Has reports whether the license grants a specific feature. Used by
// RequireFeature; also useful in admin UIs to enable/disable settings
// rows.
func (p *Payload) Has(f Feature) bool {
	for _, granted := range p.Features {
		if granted == f {
			return true
		}
	}
	return false
}

// In-process license state. The license is loaded once at startup (and
// on settings change) and cached here. RequireFeature reads from this
// cache — no per-call parse or verify.
var (
	mu      sync.RWMutex
	current *Payload // nil if no license loaded
)

// Load parses, verifies, and installs a license key. Call from startup
// (after reading settings) and from the admin "save license" handler.
//
// An empty key clears the cached license — useful when an admin removes
// the key to fall back to Community features only.
func Load(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		mu.Lock()
		current = nil
		mu.Unlock()
		return nil
	}

	parts := strings.Split(key, ".")
	if len(parts) != 2 {
		return fmt.Errorf("%w: expected <payload>.<sig>", ErrLicenseInvalid)
	}
	payloadB64, sigB64 := parts[0], parts[1]

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return fmt.Errorf("%w: payload not base64url", ErrLicenseInvalid)
	}
	sigBytes, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return fmt.Errorf("%w: signature not base64url", ErrLicenseInvalid)
	}

	if publicKeyB64 == "" {
		// Public key not configured in this build — refuse to "verify"
		// anything. Production binaries will have the real key
		// compiled in.
		return fmt.Errorf("%w: this build has no license public key", ErrLicenseInvalid)
	}
	pub, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("%w: compiled-in public key is malformed (this is a build bug)", ErrLicenseInvalid)
	}

	if !ed25519.Verify(ed25519.PublicKey(pub), payloadBytes, sigBytes) {
		return fmt.Errorf("%w: signature does not verify", ErrLicenseInvalid)
	}

	var p Payload
	if err := json.Unmarshal(payloadBytes, &p); err != nil {
		return fmt.Errorf("%w: payload not valid JSON: %v", ErrLicenseInvalid, err)
	}
	if p.IsExpired() {
		return fmt.Errorf("%w: license expired on %s", ErrLicenseInvalid,
			time.Unix(p.ExpiresAt, 0).Format("2006-01-02"))
	}

	mu.Lock()
	current = &p
	mu.Unlock()
	return nil
}

// Current returns a copy of the current license payload, or nil if no
// valid license is loaded. Returns a copy so callers can read fields
// without holding the package mutex; do not mutate the returned value.
func Current() *Payload {
	mu.RLock()
	defer mu.RUnlock()
	if current == nil {
		return nil
	}
	cp := *current
	cp.Features = append([]Feature(nil), current.Features...)
	return &cp
}

// RequireFeature returns nil if the current license grants the feature,
// ErrLicenseRequired otherwise. The standard pattern at feature
// entry points (e.g. provider construction, admin settings PUT) is:
//
//	if err := license.RequireFeature(license.FeatureEcommerceShopify); err != nil {
//	    return nil, err
//	}
//
// Returning the typed error lets the API layer respond with a clear
// HTTP 402 / 403 + "Upgrade to Pro" message instead of leaking generic
// 500s.
func RequireFeature(f Feature) error {
	mu.RLock()
	p := current
	mu.RUnlock()
	if p == nil {
		return ErrLicenseRequired
	}
	if p.IsExpired() {
		return ErrLicenseRequired
	}
	if !p.Has(f) {
		return ErrLicenseRequired
	}
	return nil
}
