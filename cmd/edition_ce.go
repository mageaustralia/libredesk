//go:build !helperiqpro

package main

// Edition tags the running binary. Set by build constraints in this file
// (CE) and edition_pro.go (Pro). Surfaced over /api/v1/config so the
// frontend can grey out Pro-only options.
//
// To build the Pro binary:
//
//	make BUILD_TAGS=helperiqpro
//	# or
//	go build -tags helperiqpro -o helperiq-pro ./cmd
//
// The default build is CE — no Pro packages are even compiled in, so
// shipping the public CE binary cannot leak Pro provider code. The Pro
// build pulls in internal/ecommerce/{magento2,shopify,woocommerce}
// from this same repo (paid customers can either build themselves or
// pull the pre-built `helperiq-pro` Docker image from the private
// registry).
const Edition = "ce"

// IsPro reports whether the current binary is the Pro edition. Cheap
// compile-time constant — the compiler will dead-code-eliminate any
// `if !IsPro { ... }` branches.
const IsPro = false
