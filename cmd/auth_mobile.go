package main

import (
	"context"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"google.golang.org/api/idtoken"
)

type googleMobileAuthReq struct {
	IDToken string `json:"id_token"`
}

// googleIssuers are the only issuers we accept an ID token from. idtoken
// verifies the signature against Google's certs, but not the iss claim.
var googleIssuers = map[string]bool{
	"accounts.google.com":         true,
	"https://accounts.google.com": true,
}

// handleGoogleMobileAuth exchanges a Google ID token for an API key pair.
//
// SECURITY: the token MUST be validated against our own OAuth client IDs.
// Passing an empty audience to idtoken.Validate skips the aud check entirely
// (see google.golang.org/api/idtoken/validate.go: `if audience != "" && ...`),
// which would accept any unexpired Google-signed ID token minted by ANY OAuth
// client on the internet — i.e. any third-party site with a "Sign in with
// Google" button could replay an agent's token here and mint a long-lived API
// key for that agent. The audience check is what binds a token to this app.
//
// Config (config.toml):
//
//	[app.google_mobile_auth]
//	client_ids     = ["<mobile-oauth-client-id>.apps.googleusercontent.com"]
//	allowed_domains = ["example.com"]   # optional: require the Workspace `hd` claim
//
// With no client_ids configured the endpoint is disabled — fail closed, so an
// install that never set this up cannot be used as a login bypass.
func handleGoogleMobileAuth(r *fastglue.Request) error {
	app := r.Context.(*App)

	clientIDs := ko.Strings("app.google_mobile_auth.client_ids")
	if len(clientIDs) == 0 {
		app.lo.Warn("google mobile auth attempted but no client_ids configured; endpoint disabled")
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Not found", nil, envelope.GeneralError)
	}

	var req googleMobileAuthReq
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request", nil, envelope.InputError)
	}

	if req.IDToken == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "id_token is required", nil, envelope.InputError)
	}

	// Validate against each configured client ID (Android/iOS/web clients each
	// have their own). A token is accepted only if its aud matches one of ours.
	var payload *idtoken.Payload
	for _, clientID := range clientIDs {
		clientID = strings.TrimSpace(clientID)
		if clientID == "" {
			continue
		}
		p, err := idtoken.Validate(context.Background(), req.IDToken, clientID)
		if err == nil {
			payload = p
			break
		}
	}
	if payload == nil {
		app.lo.Warn("google mobile auth: id token failed validation for all configured client ids")
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Invalid Google token", nil, envelope.GeneralError)
	}

	// idtoken checks the signature and expiry but not the issuer.
	if !googleIssuers[payload.Issuer] {
		app.lo.Warn("google mobile auth: unexpected issuer", "issuer", payload.Issuer)
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Invalid Google token", nil, envelope.GeneralError)
	}

	email, _ := payload.Claims["email"].(string)
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Email not found in token", nil, envelope.InputError)
	}

	emailVerified, _ := payload.Claims["email_verified"].(bool)
	if !emailVerified {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Email not verified", nil, envelope.InputError)
	}

	// Optional Workspace restriction. `hd` is only present on Workspace
	// accounts, so a consumer gmail.com token can never satisfy this.
	if allowed := ko.Strings("app.google_mobile_auth.allowed_domains"); len(allowed) > 0 {
		hd, _ := payload.Claims["hd"].(string)
		hd = strings.TrimSpace(strings.ToLower(hd))
		ok := false
		for _, d := range allowed {
			if hd != "" && hd == strings.TrimSpace(strings.ToLower(d)) {
				ok = true
				break
			}
		}
		if !ok {
			app.lo.Warn("google mobile auth: hosted domain not allowed", "email", email)
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Account not permitted", nil, envelope.GeneralError)
		}
	}

	// Look up the agent by email.
	user, err := app.user.GetAgent(0, email)
	if err != nil {
		app.lo.Warn("google mobile auth: agent not found", "email", email)
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "No agent account found for this email", nil, envelope.GeneralError)
	}

	if !user.Enabled {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Account is disabled", nil, envelope.GeneralError)
	}

	// Generate API key for this agent.
	apiKey, apiSecret, err := app.user.GenerateAPIKey(user.ID)
	if err != nil {
		app.lo.Error("error generating API key for mobile auth", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to generate credentials", nil, envelope.GeneralError)
	}

	return r.SendEnvelope(map[string]interface{}{
		"api_key":    apiKey,
		"api_secret": apiSecret,
		"user": map[string]interface{}{
			"id":         user.ID,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"email":      user.Email.String,
		},
	})
}
