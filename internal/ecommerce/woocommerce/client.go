// Package woocommerce implements the ecommerce.Provider interface for
// WooCommerce stores via the REST API v3.
//
// Auth model: HTTP Basic with consumer_key / consumer_secret. WooCommerce
// generates these in WooCommerce → Settings → Advanced → REST API. Both
// must be presented in the Authorization header for every request; there
// is no token exchange.
//
// API surface: /wp-json/wc/v3/customers, /wp-json/wc/v3/orders. Both
// support filtering with email/search params and standard pagination.
//
// HelperIQ Pro extension — license-gated in cmd/ecommerce.go.
package woocommerce

import (
	"context"
	"encoding/base64"
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

// Client implements ecommerce.Provider for WooCommerce.
type Client struct {
	baseURL   string // store base URL, e.g. https://spinfiresport.com (no trailing slash)
	basicAuth string // pre-computed "Basic <base64(ck:cs)>"
	http      *http.Client
	userAgent string
	lo        *logf.Logger
}

// New constructs a WooCommerce REST client.
//
// Config mapping:
//   - BaseURL      → store URL
//   - ClientID     → consumer_key
//   - ClientSecret → consumer_secret (encrypted at rest)
func New(config ecommerce.ProviderConfig, lo *logf.Logger) (*Client, error) {
	if config.BaseURL == "" || config.ClientID == "" || config.ClientSecret == "" {
		return nil, fmt.Errorf("woocommerce: baseURL, clientID (consumer_key), and clientSecret (consumer_secret) are required")
	}
	baseURL := strings.TrimSuffix(strings.TrimSpace(config.BaseURL), "/")
	creds := config.ClientID + ":" + config.ClientSecret
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(creds))
	return &Client{
		baseURL:   baseURL,
		basicAuth: authHeader,
		http:      &http.Client{Timeout: 20 * time.Second},
		userAgent: ecommerce.UserAgent(),
		lo:        lo,
	}, nil
}

// Name implements ecommerce.Provider.
func (c *Client) Name() string { return "woocommerce" }

// do issues an authenticated GET against the WooCommerce REST API.
func (c *Client) do(ctx context.Context, path string, query url.Values) (io.ReadCloser, error) {
	u := c.baseURL + "/wp-json/wc/v3" + path
	if query != nil {
		if encoded := query.Encode(); encoded != "" {
			u += "?" + encoded
		}
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("woocommerce: build request: %w", err)
	}
	req.Header.Set("Authorization", c.basicAuth)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("woocommerce: http: %w", err)
	}
	return ecommerce.ClassifyResponse(resp, "woocommerce")
}

// TestConnection implements ecommerce.Provider — /system_status is the
// canonical "are credentials valid + can we read?" endpoint; returns a
// modest JSON blob even on stores with no orders.
func (c *Client) TestConnection(ctx context.Context) error {
	body, err := c.do(ctx, "/system_status", nil)
	if err != nil {
		return err
	}
	body.Close()
	return nil
}

// --- Customer endpoints ----------------------------------------------------

type wcCustomer struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Billing   wcAddress `json:"billing"`
	CreatedAt string    `json:"date_created"`
}
type wcAddress struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Address1  string `json:"address_1"`
	Address2  string `json:"address_2"`
	City      string `json:"city"`
	State     string `json:"state"`
	Postcode  string `json:"postcode"`
	Country   string `json:"country"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
}

func (wc wcCustomer) toEcommerce() *ecommerce.Customer {
	created, _ := time.Parse(time.RFC3339, wc.CreatedAt)
	return &ecommerce.Customer{
		ID:        strconv.Itoa(wc.ID),
		Email:     wc.Email,
		FirstName: wc.FirstName,
		LastName:  wc.LastName,
		Telephone: wc.Billing.Phone,
		CreatedAt: created,
	}
}

// GetCustomerByEmail implements ecommerce.Provider. WooCommerce exposes
// ?email=... as the canonical lookup; we get back a list (the API
// returns an array even for unique fields).
func (c *Client) GetCustomerByEmail(ctx context.Context, email string) (*ecommerce.Customer, error) {
	if email == "" {
		return nil, fmt.Errorf("woocommerce: email required")
	}
	q := url.Values{}
	q.Set("email", email)
	q.Set("per_page", "1")
	body, err := c.do(ctx, "/customers", q)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	var customers []wcCustomer
	if err := json.NewDecoder(body).Decode(&customers); err != nil {
		return nil, fmt.Errorf("woocommerce: decode customers: %w", err)
	}
	if len(customers) == 0 {
		return nil, ecommerce.ErrNotFound
	}
	return customers[0].toEcommerce(), nil
}

// --- Order endpoints -------------------------------------------------------

type wcOrder struct {
	ID                int          `json:"id"`
	Number            string       `json:"number"`
	OrderKey          string       `json:"order_key"`
	Status            string       `json:"status"` // processing, completed, refunded, …
	Currency          string       `json:"currency"`
	Total             string       `json:"total"`
	Subtotal          string       `json:"-"` // computed from line_items below
	TotalTax          string       `json:"total_tax"`
	ShippingTotal     string       `json:"shipping_total"`
	PaymentMethodTitle string      `json:"payment_method_title"`
	DateCreated       string       `json:"date_created"`
	Billing           wcAddress    `json:"billing"`
	Shipping          wcAddress    `json:"shipping"`
	LineItems         []wcLineItem `json:"line_items"`
	ShippingLines     []wcShipping `json:"shipping_lines"`
}
type wcLineItem struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
	Price    string `json:"price"`
	Subtotal string `json:"subtotal"`
	Total    string `json:"total"`
}
type wcShipping struct {
	ID          int    `json:"id"`
	MethodTitle string `json:"method_title"`
	Total       string `json:"total"`
}

func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func (wa wcAddress) toEcommerce() *ecommerce.Address {
	return ecommerce.NewAddress(ecommerce.AddressInput{
		FirstName:   wa.FirstName,
		LastName:    wa.LastName,
		StreetLines: []string{wa.Address1, wa.Address2},
		City:        wa.City,
		Region:      wa.State,
		PostCode:    wa.Postcode,
		Country:     wa.Country,
		Telephone:   wa.Phone,
	})
}

func (wo wcOrder) toEcommerce() ecommerce.Order {
	created, _ := time.Parse(time.RFC3339, wo.DateCreated)
	items := make([]ecommerce.OrderItem, 0, len(wo.LineItems))
	var subtotal float64
	for _, li := range wo.LineItems {
		row := parseFloat(li.Total)
		subtotal += row
		items = append(items, ecommerce.OrderItem{
			SKU:      li.SKU,
			Name:     li.Name,
			Qty:      li.Quantity,
			Price:    parseFloat(li.Price),
			RowTotal: row,
		})
	}
	// WooCommerce doesn't return shipments as a top-level resource on the
	// order — fulfillment is plugin-dependent. We surface the chosen
	// shipping method as a single "shipment" with no tracking, so the
	// helpdesk UI at least shows the carrier/method. Real tracking
	// requires reading from whichever shipment plugin the store uses.
	shipments := make([]ecommerce.Shipment, 0, len(wo.ShippingLines))
	for _, s := range wo.ShippingLines {
		shipments = append(shipments, ecommerce.Shipment{
			ID:      strconv.Itoa(s.ID),
			Carrier: s.MethodTitle,
			Status:  wo.Status, // no separate fulfillment status in core Woo
		})
	}
	return ecommerce.Order{
		ID:              strconv.Itoa(wo.ID),
		IncrementID:     wo.Number,
		CustomerEmail:   wo.Billing.Email,
		CustomerName:    strings.TrimSpace(wo.Billing.FirstName + " " + wo.Billing.LastName),
		Status:          wo.Status,
		State:           wo.Status,
		Items:           items,
		Subtotal:        subtotal,
		GrandTotal:      parseFloat(wo.Total),
		ShippingAmount:  parseFloat(wo.ShippingTotal),
		Currency:        wo.Currency,
		PaymentMethod:   wo.PaymentMethodTitle,
		ShippingAddress: wo.Shipping.toEcommerce(),
		BillingAddress:  wo.Billing.toEcommerce(),
		Shipments:       shipments,
		CreatedAt:       created,
	}
}

// GetOrdersByEmail implements ecommerce.Provider. WooCommerce supports
// ?customer=ID for a known customer, or ?search=email — the latter
// works without a prior customer lookup.
func (c *Client) GetOrdersByEmail(ctx context.Context, email string, limit int) ([]ecommerce.Order, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	q := url.Values{}
	q.Set("search", email)
	q.Set("per_page", strconv.Itoa(limit))
	q.Set("orderby", "date")
	q.Set("order", "desc")
	body, err := c.do(ctx, "/orders", q)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	var orders []wcOrder
	if err := json.NewDecoder(body).Decode(&orders); err != nil {
		return nil, fmt.Errorf("woocommerce: decode orders: %w", err)
	}
	out := make([]ecommerce.Order, 0, len(orders))
	for _, o := range orders {
		out = append(out, o.toEcommerce())
	}
	return out, nil
}

// GetOrderByNumber implements ecommerce.Provider — the human order number
// in Woo is the `number` field. Some stores customise it via plugins;
// we filter via ?search=… which matches the number too.
func (c *Client) GetOrderByNumber(ctx context.Context, orderNumber string) (*ecommerce.Order, error) {
	q := url.Values{}
	q.Set("search", orderNumber)
	q.Set("per_page", "1")
	body, err := c.do(ctx, "/orders", q)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	var orders []wcOrder
	if err := json.NewDecoder(body).Decode(&orders); err != nil {
		return nil, fmt.Errorf("woocommerce: decode orders: %w", err)
	}
	if len(orders) == 0 {
		return nil, ecommerce.ErrNotFound
	}
	out := orders[0].toEcommerce()
	return &out, nil
}

// GetOrderByID implements ecommerce.Provider — /orders/{id}, returns a
// single object (not wrapped in an array).
func (c *Client) GetOrderByID(ctx context.Context, orderID string) (*ecommerce.Order, error) {
	body, err := c.do(ctx, "/orders/"+orderID, nil)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	var wo wcOrder
	if err := json.NewDecoder(body).Decode(&wo); err != nil {
		return nil, fmt.Errorf("woocommerce: decode order: %w", err)
	}
	out := wo.toEcommerce()
	return &out, nil
}
