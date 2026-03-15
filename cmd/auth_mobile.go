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

// handleGoogleMobileAuth exchanges a Google ID token for an API key pair.
func handleGoogleMobileAuth(r *fastglue.Request) error {
	app := r.Context.(*App)

	var req googleMobileAuthReq
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request", nil, envelope.InputError)
	}

	if req.IDToken == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "id_token is required", nil, envelope.InputError)
	}

	// Verify the Google ID token (audience="" accepts any client ID from our project).
	payload, err := idtoken.Validate(context.Background(), req.IDToken, "")
	if err != nil {
		app.lo.Error("error validating Google ID token", "error", err)
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
