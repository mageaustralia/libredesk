//go:build helperiqpro

package main

import (
	"github.com/abhinavxd/libredesk/internal/ecommerce"
	"github.com/abhinavxd/libredesk/internal/ecommerce/magento2"
	"github.com/abhinavxd/libredesk/internal/ecommerce/shopify"
	"github.com/abhinavxd/libredesk/internal/ecommerce/woocommerce"
	"github.com/abhinavxd/libredesk/internal/license"
	"github.com/zerodha/logf"
)

// Pro-edition implementation of createProEcommerceProvider. Compiled only
// when `-tags=helperiqpro` is set; the CE build uses ecommerce_pro_ce.go
// which returns ErrLicenseRequired without ever importing these packages.
//
// Every type still goes through a license.RequireFeature() check — the
// runtime license is the second gate on top of the compile-time gate, so
// a Pro-built binary can't activate features without a paid license key.
func createProEcommerceProvider(config ecommerce.ProviderConfig, lo *logf.Logger) (ecommerce.Provider, error) {
	switch config.Type {
	case "magento2":
		if err := license.RequireFeature(license.FeatureEcommerceMagento2); err != nil {
			return nil, err
		}
		return magento2.New(config, lo)
	case "shopify":
		if err := license.RequireFeature(license.FeatureEcommerceShopify); err != nil {
			return nil, err
		}
		return shopify.New(config, lo)
	case "woocommerce":
		if err := license.RequireFeature(license.FeatureEcommerceWooCommerce); err != nil {
			return nil, err
		}
		return woocommerce.New(config, lo)
	default:
		// Fell through from the main switch — shouldn't happen.
		return nil, nil
	}
}
