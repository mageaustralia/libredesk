package ecommerce

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"

	"github.com/zerodha/logf"
)

// Manager handles ecommerce provider operations with multi-stage context gathering
type Manager struct {
	// provider is the GLOBAL fallback provider — used when an inbox has
	// no per-inbox ecommerce config (the legacy single-store deployment
	// shape). May be nil if the global setting is unconfigured.
	provider Provider
	lo       logf.Logger

	// providerFactory constructs an ecommerce.Provider from a
	// ProviderConfig. Wired by the cmd layer so the Manager doesn't
	// have to import the concrete provider packages (which would create
	// a cycle since the providers depend on this package). nil-safe:
	// when unset, ForInbox() returns the global provider only.
	providerFactory func(ProviderConfig) (Provider, error)

	// Per-inbox provider cache. Keyed by inboxID; entries are evicted
	// when InvalidateInbox is called (admin saved a config change).
	mu             sync.RWMutex
	inboxProviders map[int]*cachedInboxProvider
}

// cachedInboxProvider pairs a built Provider with the configHash it was
// built from. On lookup, if the inbox's current configHash differs from
// the cached one, we rebuild — covers the case where a config change
// landed but InvalidateInbox wasn't called (defence in depth).
type cachedInboxProvider struct {
	provider   Provider
	configHash string
}

// NewManager creates a new ecommerce manager. The factory parameter lets
// the cmd layer inject a constructor that knows about the concrete
// provider implementations (maho, magento2, shopify, woocommerce)
// without forcing this package to import them.
//
// Pass a nil factory if you only need the global provider (no per-inbox
// support) — useful for tests.
func NewManager(provider Provider, factory func(ProviderConfig) (Provider, error), lo logf.Logger) *Manager {
	return &Manager{
		provider:        provider,
		providerFactory: factory,
		lo:              lo,
		inboxProviders:  make(map[int]*cachedInboxProvider),
	}
}

// IsConfigured returns true if the GLOBAL provider is configured.
// Per-inbox providers are checked separately via ProviderForInbox.
func (m *Manager) IsConfigured() bool {
	return m.provider != nil
}

// IsConfiguredForInbox reports whether the manager can produce a
// provider for the given inbox — either via a per-inbox config or the
// global fallback. Used by the agent UI to decide whether the
// "+ Orders" button should be enabled for a specific conversation.
//
// inboxConfig is the inbox's per-inbox ecommerce config (may be nil).
// The caller resolves the inbox row and passes the relevant subset; the
// manager doesn't reach into the inbox store directly to keep this
// package free of inbox dependencies.
func (m *Manager) IsConfiguredForInbox(inboxConfig *ProviderConfig) bool {
	if inboxConfig != nil && inboxConfig.Type != "" {
		return true
	}
	return m.provider != nil
}

// ProviderForInbox returns the provider to use for ecommerce operations
// scoped to the given inbox. If inboxConfig is set, returns a cached
// provider built from it; otherwise returns the global fallback. Returns
// nil when neither is configured (caller should treat this as "no
// ecommerce available for this inbox").
//
// configHash should be a stable hash of inboxConfig — passed in from
// the caller (cmd layer) so this package doesn't have to compute it
// (and so the cmd layer can use cheaper inputs like the inbox row's
// updated_at as a hash if it prefers).
func (m *Manager) ProviderForInbox(inboxID int, inboxConfig *ProviderConfig) (Provider, error) {
	if inboxConfig == nil || inboxConfig.Type == "" {
		return m.provider, nil
	}
	if m.providerFactory == nil {
		// No factory wired — fall back to global silently. Hit when
		// tests build a Manager without injecting one.
		return m.provider, nil
	}

	hash := configHash(inboxConfig)

	m.mu.RLock()
	cached, ok := m.inboxProviders[inboxID]
	m.mu.RUnlock()
	if ok && cached.configHash == hash {
		return cached.provider, nil
	}

	// Build a fresh provider. Hold the write lock for the construction
	// — provider factories are expected to be fast (constructor only,
	// no network round-trips); if they ever become heavyweight, switch
	// to a singleflight pattern here.
	m.mu.Lock()
	defer m.mu.Unlock()
	// Re-check under write lock (lost-update race).
	if cached, ok := m.inboxProviders[inboxID]; ok && cached.configHash == hash {
		return cached.provider, nil
	}

	provider, err := m.providerFactory(*inboxConfig)
	if err != nil {
		return nil, fmt.Errorf("build per-inbox ecommerce provider: %w", err)
	}
	if provider == nil {
		// Factory returned nil for an unknown provider type. Treat as
		// "no provider" rather than caching nil — next call will retry
		// in case the type becomes valid after a code update.
		return nil, fmt.Errorf("unknown ecommerce provider type %q", inboxConfig.Type)
	}
	m.inboxProviders[inboxID] = &cachedInboxProvider{
		provider:   provider,
		configHash: hash,
	}
	return provider, nil
}

// InvalidateInbox evicts the cached provider for the given inbox. Call
// from cmd/inboxes.go after a successful ecommerce-config PUT so the
// next ProviderForInbox call rebuilds with the new credentials.
func (m *Manager) InvalidateInbox(inboxID int) {
	m.mu.Lock()
	delete(m.inboxProviders, inboxID)
	m.mu.Unlock()
}

// configHash returns a stable SHA-256 prefix of the provider config.
// Used as the cache key suffix; differences in any field invalidate the
// cached provider. Truncated to 16 hex chars — collisions are
// astronomically unlikely for the small set of configs we handle.
func configHash(c *ProviderConfig) string {
	b, _ := json.Marshal(c)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:8])
}

// GatherFullContext performs multi-stage context gathering against the
// GLOBAL provider. Kept for back-compat callers (admin "test" endpoint,
// pre-per-inbox callers that haven't been updated). New code should use
// GatherFullContextForInbox so multi-store deployments work.
//
// Stage 1: Fetch customer + recent orders by email
// Stage 2: Scan all provided messages for order numbers
// Stage 3: Fetch full details for mentioned orders
func (m *Manager) GatherFullContext(ctx context.Context, email string, messages []string, maxOrders int) (*EcommerceContext, error) {
	return m.gatherFullContextWith(ctx, m.provider, email, messages, maxOrders)
}

// GatherFullContextForInbox is the inbox-scoped variant. Uses the inbox's
// per-inbox ecommerce config if set, otherwise falls back to the global
// provider. inboxConfig may be nil — that's the "no per-inbox config"
// case which falls back to global.
func (m *Manager) GatherFullContextForInbox(ctx context.Context, inboxID int, inboxConfig *ProviderConfig, email string, messages []string, maxOrders int) (*EcommerceContext, error) {
	provider, err := m.ProviderForInbox(inboxID, inboxConfig)
	if err != nil {
		return nil, err
	}
	return m.gatherFullContextWith(ctx, provider, email, messages, maxOrders)
}

// gatherFullContextWith is the shared implementation. Takes the provider
// as a parameter so the inbox-scoped and global-scoped public methods
// can share the multi-stage scan + warning-dedup logic without
// copy-paste drift.
func (m *Manager) gatherFullContextWith(ctx context.Context, provider Provider, email string, messages []string, maxOrders int) (*EcommerceContext, error) {
	if provider == nil {
		return nil, nil
	}

	result := &EcommerceContext{}

	// Stage 1: Fetch customer and recent orders
	customer, err := provider.GetCustomerByEmail(ctx, email)
	if err != nil && err != ErrNotFound {
		m.lo.Warn("failed to get customer", "email", email, "error", err)
		result.Warnings = append(result.Warnings, fmt.Sprintf("Customer lookup failed: %v", err))
	} else if err == nil {
		result.Customer = customer
	}

	orders, err := provider.GetOrdersByEmail(ctx, email, maxOrders)
	if err != nil && err != ErrNotFound {
		m.lo.Warn("failed to get orders", "email", email, "error", err)
		result.Warnings = append(result.Warnings, fmt.Sprintf("Recent orders lookup failed: %v", err))
	} else {
		result.RecentOrders = orders
	}

	// Stage 2: Scan ALL messages for order numbers
	m.lo.Info("scanning messages for order numbers", "message_count", len(messages))
	var foundOrderNumbers []string
	for _, msg := range messages {
		nums := extractAllOrderNumbers(msg)
		if len(nums) > 0 {
			m.lo.Info("found order numbers in message", "numbers", nums)
		}
		foundOrderNumbers = append(foundOrderNumbers, nums...)
	}
	m.lo.Info("order number scan complete", "found", foundOrderNumbers)

	// Deduplicate
	seen := make(map[string]bool)
	var uniqueOrders []string
	for _, num := range foundOrderNumbers {
		if !seen[num] {
			seen[num] = true
			uniqueOrders = append(uniqueOrders, num)
		}
	}

	// Stage 3: Fetch full details for mentioned orders (limit to first 3)
	m.lo.Info("Stage 3: fetching mentioned orders", "unique_orders", uniqueOrders)
	for i, orderNum := range uniqueOrders {
		if i >= 3 {
			break
		}
		// Skip if already in recent orders
		alreadyHave := false
		for _, ro := range result.RecentOrders {
			if ro.IncrementID == orderNum {
				// Promote to matched order with full data
				o := ro
				result.MatchedOrders = append(result.MatchedOrders, &o)
				alreadyHave = true
				break
			}
		}
		if alreadyHave {
			continue
		}
		order, err := provider.GetOrderByNumber(ctx, orderNum)
		if err == nil {
			result.MatchedOrders = append(result.MatchedOrders, order)
			m.lo.Debug("found order in conversation", "order_number", orderNum)
		} else if err != ErrNotFound {
			m.lo.Warn("failed to lookup order", "order_number", orderNum, "error", err)
			result.Warnings = append(result.Warnings, fmt.Sprintf("Order #%s lookup failed: %v", orderNum, err))
		}
	}

	// Deduplicate warnings — same auth/network failure cascades across stages
	if len(result.Warnings) > 0 {
		seen := make(map[string]bool, len(result.Warnings))
		deduped := result.Warnings[:0]
		for _, w := range result.Warnings {
			if !seen[w] {
				seen[w] = true
				deduped = append(deduped, w)
			}
		}
		result.Warnings = deduped
	}

	return result, nil
}

// FormatContextForPrompt formats ecommerce context as text for AI prompt
func (m *Manager) FormatContextForPrompt(eCtx *EcommerceContext) string {
	if eCtx == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n## Customer Ecommerce Data\n\n")

	if eCtx.Customer != nil {
		sb.WriteString(fmt.Sprintf("**Customer:** %s %s (%s)\n",
			eCtx.Customer.FirstName, eCtx.Customer.LastName, eCtx.Customer.Email))
		if eCtx.Customer.Telephone != "" {
			sb.WriteString(fmt.Sprintf("**Phone:** %s\n", eCtx.Customer.Telephone))
		}
		if !eCtx.Customer.CreatedAt.IsZero() {
			sb.WriteString(fmt.Sprintf("**Customer since:** %s\n", eCtx.Customer.CreatedAt.Format("2006-01-02")))
		}
	}

	// Show matched orders (mentioned in conversation) with FULL details
	if len(eCtx.MatchedOrders) > 0 {
		sb.WriteString("\n### Orders Mentioned in Conversation\n")
		for _, order := range eCtx.MatchedOrders {
			sb.WriteString(formatOrderFull(order))
			sb.WriteString("\n")
		}
	}

	// Show recent orders as summary only
	if len(eCtx.RecentOrders) > 0 {
		sb.WriteString("\n### Recent Orders (Summary)\n")
		for _, order := range eCtx.RecentOrders {
			// Skip if already shown in matched orders
			alreadyShown := false
			for _, matched := range eCtx.MatchedOrders {
				if matched.IncrementID == order.IncrementID {
					alreadyShown = true
					break
				}
			}
			if !alreadyShown {
				sb.WriteString(formatOrderSummary(&order))
			}
		}
	}

	return sb.String()
}

func formatOrderFull(o *Order) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n**Order #%s**\n", o.IncrementID))
	sb.WriteString(fmt.Sprintf("- Status: %s\n", o.Status))
	sb.WriteString(fmt.Sprintf("- Date: %s\n", o.CreatedAt.Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("- Total: $%.2f %s\n", o.GrandTotal, o.Currency))

	if o.TotalPaid > 0 {
		sb.WriteString(fmt.Sprintf("- Paid: $%.2f\n", o.TotalPaid))
	}
	if o.TotalRefunded > 0 {
		sb.WriteString(fmt.Sprintf("- Refunded: $%.2f\n", o.TotalRefunded))
	}
	if o.PaymentMethod != "" {
		sb.WriteString(fmt.Sprintf("- Payment: %s\n", o.PaymentMethod))
	}
	if o.ShippingMethod != "" {
		sb.WriteString(fmt.Sprintf("- Shipping: %s\n", o.ShippingMethod))
	}
	if o.ShippingAddress != nil {
		parts := []string{}
		if o.ShippingAddress.City != "" {
			parts = append(parts, o.ShippingAddress.City)
		}
		if o.ShippingAddress.Region != "" {
			parts = append(parts, o.ShippingAddress.Region)
		}
		if o.ShippingAddress.PostCode != "" {
			parts = append(parts, o.ShippingAddress.PostCode)
		}
		if o.ShippingAddress.Country != "" {
			parts = append(parts, o.ShippingAddress.Country)
		}
		if len(parts) > 0 {
			sb.WriteString(fmt.Sprintf("- Shipping to: %s\n", strings.Join(parts, ", ")))
		}
	}

	if len(o.Items) > 0 {
		sb.WriteString("- Items:\n")
		for _, item := range o.Items {
			line := fmt.Sprintf("  - %s (SKU: %s) x%d @ $%.2f = $%.2f",
				item.Name, item.SKU, item.Qty, item.Price, item.RowTotal)
			if item.QtyRefunded > 0 {
				line += fmt.Sprintf(" [REFUNDED x%d]", item.QtyRefunded)
			}
			if item.QtyShipped > 0 {
				line += fmt.Sprintf(" [SHIPPED x%d]", item.QtyShipped)
			}
			sb.WriteString(line + "\n")
		}
	}

	if len(o.Shipments) > 0 {
		sb.WriteString("- Shipments:\n")
		for _, ship := range o.Shipments {
			trackURL := trackingURL(ship.Carrier, ship.TrackingNumber)
			if trackURL != "" {
				sb.WriteString(fmt.Sprintf("  - %s Tracking: %s ( %s )\n", ship.Carrier, ship.TrackingNumber, trackURL))
			} else {
				sb.WriteString(fmt.Sprintf("  - %s Tracking: %s\n", ship.Carrier, ship.TrackingNumber))
			}
		}
	}

	if o.ShippingAddress != nil {
		sb.WriteString(fmt.Sprintf("- Ship to: %s %s, %s, %s %s %s\n",
			o.ShippingAddress.FirstName, o.ShippingAddress.LastName,
			o.ShippingAddress.Street,
			o.ShippingAddress.City, o.ShippingAddress.Region, o.ShippingAddress.PostCode))
	}

	// Status history - include all notes (most recent first for relevance).
	// Timestamps from the API (and courier times embedded in note text) are
	// UTC; convert to local Australian time so the AI quotes correct times.
	if len(o.StatusHistory) > 0 {
		sb.WriteString("- Order History (times are local Australian time):\n")
		for _, entry := range o.StatusHistory {
			if entry.Note != "" {
				sb.WriteString(fmt.Sprintf("  - [%s] %s\n", toLocalTime(entry.CreatedAt), localiseTimestamps(entry.Note)))
			}
		}
	}

	return sb.String()
}

var localTZ = func() *time.Location {
	if loc, err := time.LoadLocation("Australia/Melbourne"); err == nil {
		return loc
	}
	return time.FixedZone("AEST", 10*3600)
}()

var isoUTCRegex = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2})?Z`)

// toLocalTime converts an ISO-8601 timestamp string to local Australian time
// for display; unparseable values are returned unchanged.
func toLocalTime(ts string) string {
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, ts); err == nil {
			return t.In(localTZ).Format("2 Jan 2006 3:04 PM MST")
		}
	}
	return ts
}

// localiseTimestamps rewrites ISO-8601 UTC timestamps embedded in free text
// (e.g. courier scan times in shipping-app notes) to local Australian time.
func localiseTimestamps(text string) string {
	return isoUTCRegex.ReplaceAllStringFunc(text, func(m string) string {
		if t, err := time.Parse(time.RFC3339, m); err == nil {
			return t.In(localTZ).Format("2 Jan 2006 3:04 PM MST")
		}
		return m
	})
}

func formatOrderSummary(o *Order) string {
	summary := fmt.Sprintf("- #%s | %s | $%.2f %s | %s",
		o.IncrementID, o.Status, o.GrandTotal, o.Currency, o.CreatedAt.Format("2006-01-02"))
	if o.TotalRefunded > 0 {
		summary += fmt.Sprintf(" | Refunded: $%.2f", o.TotalRefunded)
	}
	return summary + "\n"
}

// trackingURL returns the carrier tracking URL for a given tracking number.
func trackingURL(carrier, trackingNumber string) string {
	c := strings.ToLower(carrier)
	switch {
	case strings.Contains(c, "australia post") || strings.Contains(c, "auspost") || strings.Contains(c, "eparcel"):
		return "https://auspost.com.au/mypost/track/details/" + trackingNumber
	case strings.Contains(c, "couriers please") || strings.Contains(c, "couriersplease"):
		return "https://www.couriersplease.com.au/tools-track/no/" + trackingNumber
	case strings.Contains(c, "team global") || strings.Contains(c, "tge") || strings.Contains(c, "toll"):
		return "https://www.myteamge.com/?externalSearchQuery=" + trackingNumber
	default:
		return ""
	}
}

// Order number patterns for Magento-style IDs (100xxxxxx)
var (
	orderPrefixRegex     = regexp.MustCompile(`(?i)(?:order|#|number)[:\s#]*(\d{9,12})`)
	standaloneOrderRegex = regexp.MustCompile(`\b(1\d{8,11})\b`)
)

func extractAllOrderNumbers(text string) []string {
	var results []string

	// First try prefixed patterns (higher confidence)
	matches := orderPrefixRegex.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) > 1 {
			results = append(results, match[1])
		}
	}

	// Then try standalone numbers
	matches = standaloneOrderRegex.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) > 1 {
			results = append(results, match[1])
		}
	}

	return results
}

// GetOrderByNumber looks up an order by its display number using the
// GLOBAL provider. Used by the admin "test customer/order lookup" tools.
// Production conversation-context paths should use GetOrderByNumberForInbox.
func (m *Manager) GetOrderByNumber(ctx context.Context, orderNumber string) (*Order, error) {
	if m.provider == nil {
		return nil, fmt.Errorf("no provider configured")
	}
	return m.provider.GetOrderByNumber(ctx, orderNumber)
}

// GetOrderByNumberForInbox is the inbox-scoped variant.
func (m *Manager) GetOrderByNumberForInbox(ctx context.Context, inboxID int, inboxConfig *ProviderConfig, orderNumber string) (*Order, error) {
	provider, err := m.ProviderForInbox(inboxID, inboxConfig)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, fmt.Errorf("no ecommerce provider configured for inbox %d", inboxID)
	}
	return provider.GetOrderByNumber(ctx, orderNumber)
}
