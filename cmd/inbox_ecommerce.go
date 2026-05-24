package main

import (
	"encoding/json"
	"strconv"

	"github.com/abhinavxd/libredesk/internal/ecommerce"
	"github.com/abhinavxd/libredesk/internal/envelope"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// Per-inbox ecommerce configuration handlers + helpers.
//
// Data model: the per-inbox ecommerce config lives in inboxes.config JSONB
// under the `ecommerce` key (see imodels.Config.Ecommerce). This file
// adapts that storage to the ecommerce.ProviderConfig the manager expects,
// and exposes 4 REST handlers under /api/v1/inboxes/{id}/ecommerce.
//
// The global ecommerce settings remain as a fallback — inboxes with no
// `ecommerce` block in their config use them. See
// docs/superpowers/specs/2026-05-20-per-inbox-ecommerce.md for the
// full design.

// resolveInboxEcommerceConfig reads the inbox row, parses its config, and
// returns the ecommerce.ProviderConfig representation of the inbox's
// `ecommerce` block (or nil if absent — caller falls back to global).
//
// Errors are logged and swallowed: a malformed inbox config shouldn't
// kill the "+ Orders" code path — degrading to "no per-inbox provider,
// use global" is the right behaviour.
func (app *App) resolveInboxEcommerceConfig(inboxID int) *ecommerce.ProviderConfig {
	if inboxID <= 0 {
		return nil
	}
	row, err := app.inbox.GetDBRecord(inboxID)
	if err != nil {
		app.lo.Debug("resolveInboxEcommerceConfig: inbox row not found", "inbox_id", inboxID, "error", err)
		return nil
	}
	if len(row.Config) == 0 {
		return nil
	}
	var parsed struct {
		Ecommerce *imodels.EcommerceConfig `json:"ecommerce,omitempty"`
	}
	if err := json.Unmarshal(row.Config, &parsed); err != nil {
		app.lo.Warn("resolveInboxEcommerceConfig: unmarshal inbox config", "inbox_id", inboxID, "error", err)
		return nil
	}
	if parsed.Ecommerce == nil || parsed.Ecommerce.Type == "" {
		return nil
	}
	return &ecommerce.ProviderConfig{
		Type:         parsed.Ecommerce.Type,
		BaseURL:      parsed.Ecommerce.BaseURL,
		ClientID:     parsed.Ecommerce.ClientID,
		ClientSecret: parsed.Ecommerce.ClientSecret,
		ExtraConfig:  parsed.Ecommerce.ExtraConfig,
	}
}

// handleGetInboxEcommerce returns the inbox's ecommerce config.
// Response shape:
//
//	{ "data": { "type": "...", ..., "inherited": false } }
//
// `inherited=true` means the inbox has no per-inbox config and the
// returned values are the global fallback — useful for the UI to show
// "Using global settings" in the field placeholders.
//
// The client_secret is never returned plaintext — always masked to "***"
// when present, matching the SMTP/IMAP password convention in the inbox
// form so the form can be re-submitted without losing the secret.
func handleGetInboxEcommerce(r *fastglue.Request) error {
	app := r.Context.(*App)
	inboxID, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	cfg := app.resolveInboxEcommerceConfig(inboxID)
	inherited := false
	if cfg == nil {
		inherited = true
		// Return the global config as the inherited default so the UI
		// can show what's currently in effect. Don't leak the global
		// client_secret either.
		// We re-read it from settings each call rather than caching —
		// admin changes propagate immediately on the next view of any
		// inbox.
		out, err := app.setting.GetByPrefix(ecommerceSettingsKey)
		if err == nil {
			var settings map[string]interface{}
			if json.Unmarshal(out, &settings) == nil {
				cfg = &ecommerce.ProviderConfig{
					Type:     getStringFromSettings(settings, "ecommerce.type"),
					BaseURL:  getStringFromSettings(settings, "ecommerce.base_url"),
					ClientID: getStringFromSettings(settings, "ecommerce.client_id"),
				}
			}
		}
	}

	if cfg == nil {
		// Neither per-inbox nor global. Return an empty config so the
		// UI renders an empty form (rather than "null").
		cfg = &ecommerce.ProviderConfig{}
	}

	hasSecret := cfg.ClientSecret != ""
	cfg.ClientSecret = ""
	return r.SendEnvelope(map[string]any{
		"type":              cfg.Type,
		"base_url":          cfg.BaseURL,
		"client_id":         cfg.ClientID,
		"client_secret_set": hasSecret, // UI shows "•••" placeholder when true
		"extra_config":      cfg.ExtraConfig,
		"inherited":         inherited,
	})
}

// handleUpdateInboxEcommerce upserts the inbox's ecommerce config.
//
// Body shape: same as ecommerce.ProviderConfig (type/base_url/client_id/
// client_secret/extra_config). If client_secret is empty AND the inbox
// already had a config, the existing secret is preserved — matches the
// SMTP/IMAP password-preservation pattern in internal/inbox/inbox.go.
//
// Empty `type` clears the per-inbox config (inbox falls back to global).
//
// On success, the manager's per-inbox provider cache is invalidated so
// the next "+ Orders" / RAG-generate call rebuilds with the new creds.
func handleUpdateInboxEcommerce(r *fastglue.Request) error {
	app := r.Context.(*App)
	inboxID, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	var req imodels.EcommerceConfig
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	// Validate type if non-empty.
	if req.Type != "" {
		switch req.Type {
		case "magento1", "magento2", "shopify", "woocommerce":
			// ok
		default:
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "{globals.terms.provider}"), nil, envelope.InputError)
		}
		if req.BaseURL == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "base_url"), nil, envelope.InputError)
		}
		if err := validateEcommerceBaseURL(req.BaseURL); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, envelope.InputError)
		}
	}

	// Read the current inbox row, modify the ecommerce block in its
	// config, write it back via the inbox manager's Update path. Update()
	// is the right entry point (rather than a direct DB write) because
	// it preserves SMTP/IMAP/ecommerce passwords and re-encrypts at rest.
	row, err := app.inbox.GetDBRecord(inboxID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	var existing map[string]any
	if len(row.Config) > 0 {
		_ = json.Unmarshal(row.Config, &existing)
	}
	if existing == nil {
		existing = make(map[string]any)
	}

	if req.Type == "" {
		// Clear per-inbox config — drop the key entirely so resolve
		// returns nil and we fall back to global.
		delete(existing, "ecommerce")
	} else {
		existing["ecommerce"] = map[string]any{
			"type":          req.Type,
			"base_url":      req.BaseURL,
			"client_id":     req.ClientID,
			"client_secret": req.ClientSecret, // may be "" — Update() preserves
			"extra_config":  req.ExtraConfig,
		}
	}

	updatedConfig, err := json.Marshal(existing)
	if err != nil {
		app.lo.Error("inbox_ecommerce: marshal updated config", "inbox_id", inboxID, "error", err)
		return sendErrorEnvelope(r, err)
	}
	row.Config = updatedConfig

	if _, err := app.inbox.Update(inboxID, row); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Evict any cached per-inbox provider so next lookup uses the
	// fresh creds (or falls back to global if cleared).
	if app.ecommerce != nil {
		app.ecommerce.InvalidateInbox(inboxID)
	}

	return r.SendEnvelope(true)
}

// handleDeleteInboxEcommerce clears the per-inbox ecommerce config —
// equivalent to UpdateInboxEcommerce with type="". Convenience endpoint
// so the UI's "Use global settings" toggle has an obvious wire.
func handleDeleteInboxEcommerce(r *fastglue.Request) error {
	app := r.Context.(*App)
	inboxID, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}
	row, err := app.inbox.GetDBRecord(inboxID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	var existing map[string]any
	if len(row.Config) > 0 {
		_ = json.Unmarshal(row.Config, &existing)
	}
	if existing != nil {
		delete(existing, "ecommerce")
	}
	updatedConfig, err := json.Marshal(existing)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	row.Config = updatedConfig
	if _, err := app.inbox.Update(inboxID, row); err != nil {
		return sendErrorEnvelope(r, err)
	}
	if app.ecommerce != nil {
		app.ecommerce.InvalidateInbox(inboxID)
	}
	return r.SendEnvelope(true)
}

// handleTestInboxEcommerce runs a connection test against the supplied
// (proposed) inbox ecommerce config. Mirrors the global
// handleTestEcommerceConnection — used by the "Test connection" button
// on the inbox's Ecommerce tab before save.
//
// We deliberately build a provider from the request body rather than the
// persisted config so the agent can test new credentials before
// committing them. The provider is discarded after the test.
func handleTestInboxEcommerce(r *fastglue.Request) error {
	app := r.Context.(*App)

	var req imodels.EcommerceConfig
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}
	if req.Type == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "{globals.terms.provider}"), nil, envelope.InputError)
	}

	// If client_secret is empty in the request AND we have an existing
	// inbox config, fall back to the stored secret — same pattern as
	// SMTP/IMAP "test" with masked password.
	if req.ClientSecret == "" {
		inboxIDStr, _ := r.RequestCtx.UserValue("id").(string)
		if inboxID, err := strconv.Atoi(inboxIDStr); err == nil {
			if existing := app.resolveInboxEcommerceConfig(inboxID); existing != nil {
				req.ClientSecret = existing.ClientSecret
			}
		}
	}

	cfg := ecommerce.ProviderConfig{
		Type:         req.Type,
		BaseURL:      req.BaseURL,
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
		ExtraConfig:  req.ExtraConfig,
	}
	if err := validateEcommerceBaseURL(req.BaseURL); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, envelope.InputError)
	}
	provider, err := createEcommerceProvider(app, cfg)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if provider == nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "{globals.terms.provider}"), nil, envelope.InputError)
	}
	// Mirror the global handler (SS3): log the full upstream error
	// server-side, return a generic message so internal hostnames /
	// backend details aren't leaked to the client.
	if err := provider.TestConnection(r.RequestCtx); err != nil {
		app.lo.Error("per-inbox ecommerce connection test failed", "inbox_id", r.RequestCtx.UserValue("id"), "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("admin.ecommerce.connectionFailed"), nil, envelope.GeneralError)
	}
	return r.SendEnvelope(true)
}
