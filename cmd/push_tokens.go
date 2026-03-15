package main

import (
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type pushTokenReq struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

// handleRegisterPushToken registers an FCM push token for the current user.
func handleRegisterPushToken(r *fastglue.Request) error {
	app := r.Context.(*App)
	auser := r.RequestCtx.UserValue("user").(amodels.User)

	var req pushTokenReq
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request", nil, envelope.InputError)
	}

	if req.Token == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Token is required", nil, envelope.InputError)
	}
	if req.Platform != "android" && req.Platform != "ios" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Platform must be 'android' or 'ios'", nil, envelope.InputError)
	}

	_, err := app.db.Exec(
		"INSERT INTO user_push_tokens (user_id, token, platform, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW()) ON CONFLICT (user_id, token) DO UPDATE SET platform = EXCLUDED.platform, updated_at = NOW()",
		auser.ID, req.Token, req.Platform,
	)
	if err != nil {
		app.lo.Error("error registering push token", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to register token", nil, envelope.GeneralError)
	}

	return r.SendEnvelope("ok")
}

// handleUnregisterPushToken removes an FCM push token.
func handleUnregisterPushToken(r *fastglue.Request) error {
	app := r.Context.(*App)
	auser := r.RequestCtx.UserValue("user").(amodels.User)

	var req struct {
		Token string `json:"token"`
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request", nil, envelope.InputError)
	}

	_, _ = app.db.Exec("DELETE FROM user_push_tokens WHERE user_id = $1 AND token = $2", auser.ID, req.Token)
	return r.SendEnvelope("ok")
}
