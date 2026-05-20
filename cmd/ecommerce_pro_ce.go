//go:build !helperiqpro

package main

import (
	"github.com/abhinavxd/libredesk/internal/ecommerce"
	"github.com/abhinavxd/libredesk/internal/license"
	"github.com/zerodha/logf"
)

// CE-edition stub for createProEcommerceProvider. The Pro-build variant
// in ecommerce_pro.go imports magento2/shopify/woocommerce + does the
// license-gated dispatch; the CE build can't even reference those
// packages, so we just return ErrLicenseRequired (typed) so the rest of
// the codebase can treat "Pro not built in" and "Pro built but
// unlicensed" identically — same HTTP response, same admin UI message.
//
// The unused-param lints are deliberate: this signature must match the
// Pro variant.
func createProEcommerceProvider(_ ecommerce.ProviderConfig, _ *logf.Logger) (ecommerce.Provider, error) {
	return nil, license.ErrLicenseRequired
}
