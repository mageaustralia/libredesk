package ecommerce

import "runtime/debug"

// UserAgent returns "libredesk/<version>" derived from the embedded build info
// so remote ecommerce platforms (Maho/Magento, Shopify, WooCommerce, etc.) can
// identify libredesk traffic in their access logs. Falls back to the VCS
// revision when the binary was built without a version tag, and finally to
// "libredesk/unknown" if no build info is available at all.
func UserAgent() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "libredesk/unknown"
	}
	v := info.Main.Version
	if v != "" && v != "(devel)" {
		return "libredesk/" + v
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && s.Value != "" {
			rev := s.Value
			if len(rev) > 12 {
				rev = rev[:12]
			}
			return "libredesk/" + rev
		}
	}
	return "libredesk/devel"
}
