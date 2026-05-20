// Package shopify implements the ecommerce.Provider interface for Shopify
// stores via the Admin REST API.
//
// Auth model: a single private-app / custom-app Admin API access token,
// passed verbatim in the X-Shopify-Access-Token header on every request.
// There is no separate OAuth refresh flow — the token is long-lived and
// rotated only when the admin re-generates it in the Shopify admin UI.
//
// API surface used: customers, orders, fulfillments. We map Shopify's
// rest shape onto the shared ecommerce.Customer / Order / Shipment types
// in ../models.go so the rest of the helpdesk doesn't know or care which
// platform the data came from.
//
// NOTE: ships as a HelperIQ Pro extension — runtime gate in cmd/ecommerce.go's
// createEcommerceProvider verifies a valid license via internal/license
// before constructing the client.
package shopify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/ecommerce"
	"github.com/zerodha/logf"
)

// Client implements ecommerce.Provider for Shopify.
type Client struct {
	// shopDomain is the {shop}.myshopify.com hostname (no scheme).
	shopDomain string
	// apiVersion is the Shopify Admin REST API version, e.g. "2024-10".
	// Pinned per-deployment via ProviderConfig.ExtraConfig["api_version"];
	// defaults to a known-good recent version if omitted.
	apiVersion string
	accessToken string
	http        *http.Client
	userAgent   string
	lo          *logf.Logger
}

const defaultAPIVersion = "2024-10"

// New constructs a Shopify Admin API client.
//
// Config mapping (from ecommerce.ProviderConfig):
//   - BaseURL      → shop domain (with or without https://)
//   - ClientSecret → Admin API access token (encrypted at rest)
//   - ClientID     → unused (Shopify private apps don't have a separate ID)
//   - ExtraConfig["api_version"] → optional, e.g. "2024-10"
func New(config ecommerce.ProviderConfig, lo *logf.Logger) (*Client, error) {
	if config.BaseURL == "" || config.ClientSecret == "" {
		return nil, fmt.Errorf("shopify: baseURL (shop domain) and clientSecret (access token) are required")
	}

	// Normalise the shop domain: strip scheme, strip trailing slash, lowercase.
	domain := strings.TrimSpace(config.BaseURL)
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimSuffix(domain, "/")
	domain = strings.ToLower(domain)
	if !strings.HasSuffix(domain, ".myshopify.com") && !strings.Contains(domain, ".") {
		// Allow bare shop names like "tenniswarehouse" → ".myshopify.com" suffix.
		domain = domain + ".myshopify.com"
	}

	apiVersion := config.ExtraConfig["api_version"]
	if apiVersion == "" {
		apiVersion = defaultAPIVersion
	}

	return &Client{
		shopDomain:  domain,
		apiVersion:  apiVersion,
		accessToken: config.ClientSecret,
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
		userAgent: ecommerce.UserAgent(),
		lo:        lo,
	}, nil
}

// Name implements ecommerce.Provider.
func (c *Client) Name() string { return "shopify" }

// do issues an authenticated GET against the Shopify Admin REST API.
// Caller is responsible for closing the response body via the returned
// io.ReadCloser; do() returns the body so we can stream parse on the
// rare large-response paths (orders list with line items).
//
// 429 handling: Shopify uses a leaky-bucket rate limit and returns
// `Retry-After`. We retry up to 2 times with that delay, then surface
// the error. Not exposed as ecommerce.ErrTooManyRequests because the
// helpdesk treats it the same as any other transient error.
func (c *Client) do(ctx context.Context, path string, query url.Values) (io.ReadCloser, error) {
	if query == nil {
		query = url.Values{}
	}
	u := "https://" + c.shopDomain + "/admin/api/" + c.apiVersion + path
	if encoded := query.Encode(); encoded != "" {
		u += "?" + encoded
	}

	const maxRetries = 2
	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
		if err != nil {
			return nil, fmt.Errorf("shopify: build request: %w", err)
		}
		req.Header.Set("X-Shopify-Access-Token", c.accessToken)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", c.userAgent)

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("shopify: http: %w", err)
		}

		// 429 is the only retryable code; everything else delegates to the
		// shared classifier. Shopify uses a leaky-bucket rate limit and
		// always sends Retry-After on 429, so we honour that delay and
		// retry up to maxRetries before giving up.
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			if attempt >= maxRetries {
				return nil, fmt.Errorf("shopify: rate limited after %d retries", attempt)
			}
			delay := time.Second
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, err := strconv.ParseFloat(ra, 64); err == nil {
					delay = time.Duration(secs * float64(time.Second))
				}
			}
			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		return ecommerce.ClassifyResponse(resp, "shopify")
	}
	return nil, fmt.Errorf("shopify: exhausted retries unexpectedly")
}

// TestConnection implements ecommerce.Provider — hits /shop.json which is
// the lightest authenticated endpoint Shopify exposes. Returns ErrUnauthorized
// for bad tokens (mapped from 401/403 in do()) so the admin UI can show a
// useful error.
func (c *Client) TestConnection(ctx context.Context) error {
	body, err := c.do(ctx, "/shop.json", nil)
	if err != nil {
		return err
	}
	body.Close()
	return nil
}

// --- Customer endpoints ----------------------------------------------------

type shopifyCustomerEnvelope struct {
	Customers []shopifyCustomer `json:"customers"`
}
type shopifyCustomerSingle struct {
	Customer shopifyCustomer `json:"customer"`
}
type shopifyCustomer struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"created_at"`
}

func (sc shopifyCustomer) toEcommerce() *ecommerce.Customer {
	return &ecommerce.Customer{
		ID:        strconv.FormatInt(sc.ID, 10),
		Email:     sc.Email,
		FirstName: sc.FirstName,
		LastName:  sc.LastName,
		Telephone: sc.Phone,
		CreatedAt: sc.CreatedAt,
	}
}

// GetCustomerByEmail implements ecommerce.Provider. Uses /customers/search.json
// rather than /customers.json?email= because the latter was deprecated in
// 2022-07 — search.json supports the email: filter syntax that's stable
// across API versions.
func (c *Client) GetCustomerByEmail(ctx context.Context, email string) (*ecommerce.Customer, error) {
	if email == "" {
		return nil, fmt.Errorf("shopify: email required")
	}
	q := url.Values{}
	q.Set("query", "email:"+email)
	q.Set("limit", "1")
	body, err := c.do(ctx, "/customers/search.json", q)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	var env shopifyCustomerEnvelope
	if err := json.NewDecoder(body).Decode(&env); err != nil {
		return nil, fmt.Errorf("shopify: decode customers: %w", err)
	}
	if len(env.Customers) == 0 {
		return nil, ecommerce.ErrNotFound
	}
	return env.Customers[0].toEcommerce(), nil
}

// --- Order endpoints -------------------------------------------------------

type shopifyOrderEnvelope struct {
	Orders []shopifyOrder `json:"orders"`
}
type shopifyOrderSingle struct {
	Order shopifyOrder `json:"order"`
}
type shopifyOrder struct {
	ID                int64                `json:"id"`
	Name              string               `json:"name"`               // Display number e.g. "#1001"
	OrderNumber       int64                `json:"order_number"`       // Numeric form
	Email             string               `json:"email"`
	FinancialStatus   string               `json:"financial_status"`   // paid, pending, refunded …
	FulfillmentStatus string               `json:"fulfillment_status"` // fulfilled, partial, null
	Currency          string               `json:"currency"`
	TotalPrice        string               `json:"total_price"`
	SubtotalPrice     string               `json:"subtotal_price"`
	TotalShipping     string               `json:"total_shipping_price_set.shop_money.amount"` // best-effort
	CreatedAt         time.Time            `json:"created_at"`
	LineItems         []shopifyLineItem    `json:"line_items"`
	ShippingAddress   *shopifyAddress      `json:"shipping_address"`
	BillingAddress    *shopifyAddress      `json:"billing_address"`
	Fulfillments      []shopifyFulfillment `json:"fulfillments"`
	Customer          *shopifyCustomer     `json:"customer"`
}

type shopifyLineItem struct {
	SKU         string `json:"sku"`
	Name        string `json:"name"`
	Quantity    int    `json:"quantity"`
	Price       string `json:"price"`
	TotalDiscount string `json:"total_discount"`
}

type shopifyAddress struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Address1  string `json:"address1"`
	Address2  string `json:"address2"`
	City      string `json:"city"`
	Province  string `json:"province"`
	Zip       string `json:"zip"`
	Country   string `json:"country"`
	Phone     string `json:"phone"`
}

type shopifyFulfillment struct {
	ID             int64     `json:"id"`
	Status         string    `json:"status"`
	TrackingNumber string    `json:"tracking_number"`
	TrackingCompany string   `json:"tracking_company"`
	CreatedAt      time.Time `json:"created_at"`
}

func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func (sa *shopifyAddress) toEcommerce() *ecommerce.Address {
	if sa == nil {
		return nil
	}
	return ecommerce.NewAddress(ecommerce.AddressInput{
		FirstName:   sa.FirstName,
		LastName:    sa.LastName,
		StreetLines: []string{sa.Address1, sa.Address2},
		City:        sa.City,
		Region:      sa.Province,
		PostCode:    sa.Zip,
		Country:     sa.Country,
		Telephone:   sa.Phone,
	})
}

func (so shopifyOrder) toEcommerce() ecommerce.Order {
	items := make([]ecommerce.OrderItem, 0, len(so.LineItems))
	for _, li := range so.LineItems {
		price := parseFloat(li.Price)
		items = append(items, ecommerce.OrderItem{
			SKU:      li.SKU,
			Name:     li.Name,
			Qty:      li.Quantity,
			Price:    price,
			RowTotal: price * float64(li.Quantity),
		})
	}
	shipments := make([]ecommerce.Shipment, 0, len(so.Fulfillments))
	for _, f := range so.Fulfillments {
		shipments = append(shipments, ecommerce.Shipment{
			ID:             strconv.FormatInt(f.ID, 10),
			TrackingNumber: f.TrackingNumber,
			Carrier:        f.TrackingCompany,
			Status:         f.Status,
			CreatedAt:      f.CreatedAt,
		})
	}
	customerName := ""
	if so.Customer != nil {
		customerName = strings.TrimSpace(so.Customer.FirstName + " " + so.Customer.LastName)
	}
	// Map Shopify's financial+fulfillment status pair onto a single string
	// the helpdesk UI can render. Keep both in `state` for AI context.
	status := so.FinancialStatus
	if so.FulfillmentStatus != "" {
		status += "/" + so.FulfillmentStatus
	}
	return ecommerce.Order{
		ID:              strconv.FormatInt(so.ID, 10),
		IncrementID:     so.Name,
		CustomerEmail:   so.Email,
		CustomerName:    customerName,
		Status:          status,
		State:           so.FinancialStatus,
		Items:           items,
		Subtotal:        parseFloat(so.SubtotalPrice),
		GrandTotal:      parseFloat(so.TotalPrice),
		Currency:        so.Currency,
		ShippingAddress: so.ShippingAddress.toEcommerce(),
		BillingAddress:  so.BillingAddress.toEcommerce(),
		Shipments:       shipments,
		CreatedAt:       so.CreatedAt,
	}
}

// GetOrdersByEmail implements ecommerce.Provider.
func (c *Client) GetOrdersByEmail(ctx context.Context, email string, limit int) ([]ecommerce.Order, error) {
	if limit <= 0 || limit > 250 {
		limit = 10
	}
	q := url.Values{}
	q.Set("email", email)
	q.Set("status", "any")
	q.Set("limit", strconv.Itoa(limit))
	body, err := c.do(ctx, "/orders.json", q)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	var env shopifyOrderEnvelope
	if err := json.NewDecoder(body).Decode(&env); err != nil {
		return nil, fmt.Errorf("shopify: decode orders: %w", err)
	}
	out := make([]ecommerce.Order, 0, len(env.Orders))
	for _, o := range env.Orders {
		out = append(out, o.toEcommerce())
	}
	return out, nil
}

// GetOrderByNumber implements ecommerce.Provider. Shopify exposes the
// display order number as `name` (e.g. "#1001") on the order resource —
// we filter via the `name` query param and accept inputs with or without
// the leading "#".
func (c *Client) GetOrderByNumber(ctx context.Context, orderNumber string) (*ecommerce.Order, error) {
	if orderNumber == "" {
		return nil, fmt.Errorf("shopify: order number required")
	}
	name := orderNumber
	if !strings.HasPrefix(name, "#") {
		name = "#" + name
	}
	q := url.Values{}
	q.Set("name", name)
	q.Set("status", "any")
	q.Set("limit", "1")
	body, err := c.do(ctx, "/orders.json", q)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	var env shopifyOrderEnvelope
	if err := json.NewDecoder(body).Decode(&env); err != nil {
		return nil, fmt.Errorf("shopify: decode orders: %w", err)
	}
	if len(env.Orders) == 0 {
		return nil, ecommerce.ErrNotFound
	}
	out := env.Orders[0].toEcommerce()
	return &out, nil
}

// GetOrderByID implements ecommerce.Provider — direct lookup by Shopify's
// internal numeric ID (the `id` field, not `order_number`).
func (c *Client) GetOrderByID(ctx context.Context, orderID string) (*ecommerce.Order, error) {
	if orderID == "" {
		return nil, fmt.Errorf("shopify: order id required")
	}
	body, err := c.do(ctx, "/orders/"+orderID+".json", nil)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	var env shopifyOrderSingle
	if err := json.NewDecoder(body).Decode(&env); err != nil {
		return nil, fmt.Errorf("shopify: decode order: %w", err)
	}
	out := env.Order.toEcommerce()
	return &out, nil
}
