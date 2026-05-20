// Package magento2 implements the ecommerce.Provider interface for
// Magento 2 / Adobe Commerce stores via the REST V1 API.
//
// Auth model: backend Integration token. The admin creates an Integration
// under System → Extensions → Integrations, grants the resources
// (Sales/Customers read), and we use the resulting access token as a
// bearer token. This is preferred over admin user tokens (which expire
// every 4 hours and tie to a human account).
//
// API shape: searchCriteria query parameters are the universal filter
// mechanism — searchCriteria[filterGroups][0][filters][0][field]=email
// &searchCriteria[filterGroups][0][filters][0][value]=foo@bar.com
//
// HelperIQ Pro extension — license-gated in cmd/ecommerce.go.
package magento2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/ecommerce"
	"github.com/zerodha/logf"
)

// Client implements ecommerce.Provider for Magento 2.
type Client struct {
	baseURL     string // e.g. https://store.example.com (no trailing slash)
	accessToken string
	http        *http.Client
	userAgent   string
	lo          *logf.Logger
}

// New constructs a Magento 2 REST client.
//
// Config mapping:
//   - BaseURL      → store base URL
//   - ClientSecret → Integration access token (encrypted at rest)
//   - ClientID     → unused for integration-token auth; left for future
//                    OAuth1.0a flow if ever needed
func New(config ecommerce.ProviderConfig, lo *logf.Logger) (*Client, error) {
	if config.BaseURL == "" || config.ClientSecret == "" {
		return nil, fmt.Errorf("magento2: baseURL and clientSecret (integration token) are required")
	}
	baseURL := strings.TrimSuffix(strings.TrimSpace(config.BaseURL), "/")
	return &Client{
		baseURL:     baseURL,
		accessToken: config.ClientSecret,
		http:        &http.Client{Timeout: 20 * time.Second},
		userAgent:   userAgentString(),
		lo:          lo,
	}, nil
}

// Name implements ecommerce.Provider.
func (c *Client) Name() string { return "magento2" }

func userAgentString() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		v := info.Main.Version
		if v == "" || v == "(devel)" {
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
		return "libredesk/" + v
	}
	return "libredesk/unknown"
}

// do issues an authenticated GET. Caller closes the returned body.
func (c *Client) do(ctx context.Context, path string, query url.Values) (io.ReadCloser, error) {
	u := c.baseURL + "/rest/V1" + path
	if query != nil {
		if encoded := query.Encode(); encoded != "" {
			u += "?" + encoded
		}
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("magento2: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("magento2: http: %w", err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		return resp.Body, nil
	case http.StatusNotFound:
		resp.Body.Close()
		return nil, ecommerce.ErrNotFound
	case http.StatusUnauthorized, http.StatusForbidden:
		resp.Body.Close()
		return nil, ecommerce.ErrUnauthorized
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		return nil, fmt.Errorf("magento2: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}

// searchCriteria builds the obnoxious nested-query-string format
// Magento 2's API requires for filtering. Example output:
//   searchCriteria[filterGroups][0][filters][0][field]=email
//   searchCriteria[filterGroups][0][filters][0][value]=foo@bar.com
//   searchCriteria[filterGroups][0][filters][0][condition_type]=eq
//   searchCriteria[pageSize]=10
//
// Returning a url.Values keeps things composable with .Add() for extra
// filters at call sites without re-parsing.
func searchCriteria(field, value, condition string, pageSize int) url.Values {
	v := url.Values{}
	v.Set("searchCriteria[filterGroups][0][filters][0][field]", field)
	v.Set("searchCriteria[filterGroups][0][filters][0][value]", value)
	v.Set("searchCriteria[filterGroups][0][filters][0][condition_type]", condition)
	if pageSize > 0 {
		v.Set("searchCriteria[pageSize]", strconv.Itoa(pageSize))
	}
	return v
}

// TestConnection implements ecommerce.Provider — hits the store-config
// endpoint which any integration with even minimal grants can read.
func (c *Client) TestConnection(ctx context.Context) error {
	body, err := c.do(ctx, "/store/storeConfigs", nil)
	if err != nil {
		return err
	}
	body.Close()
	return nil
}

// --- Customer endpoints ----------------------------------------------------

type m2CustomerSearch struct {
	Items []m2Customer `json:"items"`
}
type m2Customer struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	CreatedAt string `json:"created_at"`
}

func (mc m2Customer) toEcommerce() *ecommerce.Customer {
	created, _ := time.Parse("2006-01-02 15:04:05", mc.CreatedAt)
	return &ecommerce.Customer{
		ID:        strconv.Itoa(mc.ID),
		Email:     mc.Email,
		FirstName: mc.FirstName,
		LastName:  mc.LastName,
		CreatedAt: created,
	}
}

// GetCustomerByEmail implements ecommerce.Provider.
func (c *Client) GetCustomerByEmail(ctx context.Context, email string) (*ecommerce.Customer, error) {
	if email == "" {
		return nil, fmt.Errorf("magento2: email required")
	}
	body, err := c.do(ctx, "/customers/search", searchCriteria("email", email, "eq", 1))
	if err != nil {
		return nil, err
	}
	defer body.Close()
	var env m2CustomerSearch
	if err := json.NewDecoder(body).Decode(&env); err != nil {
		return nil, fmt.Errorf("magento2: decode customers: %w", err)
	}
	if len(env.Items) == 0 {
		return nil, ecommerce.ErrNotFound
	}
	return env.Items[0].toEcommerce(), nil
}

// --- Order endpoints -------------------------------------------------------

type m2OrderSearch struct {
	Items []m2Order `json:"items"`
}
type m2Order struct {
	EntityID       int            `json:"entity_id"`
	IncrementID    string         `json:"increment_id"`
	CustomerEmail  string         `json:"customer_email"`
	CustomerFirst  string         `json:"customer_firstname"`
	CustomerLast   string         `json:"customer_lastname"`
	Status         string         `json:"status"`
	State          string         `json:"state"`
	Items          []m2OrderItem  `json:"items"`
	Subtotal       float64        `json:"subtotal"`
	GrandTotal     float64        `json:"grand_total"`
	TotalPaid      float64        `json:"total_paid"`
	TotalRefunded  float64        `json:"total_refunded"`
	ShippingAmount float64        `json:"shipping_amount"`
	OrderCurrency  string         `json:"order_currency_code"`
	PaymentMethod  string         `json:"payment_method"` // best-effort; real path is extension_attributes
	CreatedAt      string         `json:"created_at"`
	ExtensionAttrs map[string]any `json:"extension_attributes,omitempty"`
}
type m2OrderItem struct {
	SKU      string  `json:"sku"`
	Name     string  `json:"name"`
	QtyOrdered float64 `json:"qty_ordered"`
	Price    float64 `json:"price"`
	RowTotal float64 `json:"row_total"`
}

func (mo m2Order) toEcommerce() ecommerce.Order {
	created, _ := time.Parse("2006-01-02 15:04:05", mo.CreatedAt)
	items := make([]ecommerce.OrderItem, 0, len(mo.Items))
	for _, it := range mo.Items {
		items = append(items, ecommerce.OrderItem{
			SKU:      it.SKU,
			Name:     it.Name,
			Qty:      int(it.QtyOrdered),
			Price:    it.Price,
			RowTotal: it.RowTotal,
		})
	}
	return ecommerce.Order{
		ID:             strconv.Itoa(mo.EntityID),
		IncrementID:    mo.IncrementID,
		CustomerEmail:  mo.CustomerEmail,
		CustomerName:   strings.TrimSpace(mo.CustomerFirst + " " + mo.CustomerLast),
		Status:         mo.Status,
		State:          mo.State,
		Items:          items,
		Subtotal:       mo.Subtotal,
		GrandTotal:     mo.GrandTotal,
		TotalPaid:      mo.TotalPaid,
		TotalRefunded:  mo.TotalRefunded,
		ShippingAmount: mo.ShippingAmount,
		Currency:       mo.OrderCurrency,
		PaymentMethod:  mo.PaymentMethod,
		CreatedAt:      created,
	}
}

// GetOrdersByEmail implements ecommerce.Provider.
func (c *Client) GetOrdersByEmail(ctx context.Context, email string, limit int) ([]ecommerce.Order, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	q := searchCriteria("customer_email", email, "eq", limit)
	q.Set("searchCriteria[sortOrders][0][field]", "created_at")
	q.Set("searchCriteria[sortOrders][0][direction]", "DESC")
	body, err := c.do(ctx, "/orders", q)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	var env m2OrderSearch
	if err := json.NewDecoder(body).Decode(&env); err != nil {
		return nil, fmt.Errorf("magento2: decode orders: %w", err)
	}
	out := make([]ecommerce.Order, 0, len(env.Items))
	for _, o := range env.Items {
		out = append(out, o.toEcommerce())
	}
	return out, nil
}

// GetOrderByNumber implements ecommerce.Provider — looks up by increment_id
// which is the human-facing order number Magento exposes in confirmations.
func (c *Client) GetOrderByNumber(ctx context.Context, orderNumber string) (*ecommerce.Order, error) {
	body, err := c.do(ctx, "/orders", searchCriteria("increment_id", orderNumber, "eq", 1))
	if err != nil {
		return nil, err
	}
	defer body.Close()
	var env m2OrderSearch
	if err := json.NewDecoder(body).Decode(&env); err != nil {
		return nil, fmt.Errorf("magento2: decode order: %w", err)
	}
	if len(env.Items) == 0 {
		return nil, ecommerce.ErrNotFound
	}
	out := env.Items[0].toEcommerce()
	return &out, nil
}

// GetOrderByID implements ecommerce.Provider — direct /orders/{entity_id}
// lookup, no searchCriteria envelope.
func (c *Client) GetOrderByID(ctx context.Context, orderID string) (*ecommerce.Order, error) {
	body, err := c.do(ctx, "/orders/"+orderID, nil)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	var mo m2Order
	if err := json.NewDecoder(body).Decode(&mo); err != nil {
		return nil, fmt.Errorf("magento2: decode order: %w", err)
	}
	out := mo.toEcommerce()
	return &out, nil
}
