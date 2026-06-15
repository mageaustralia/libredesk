package main

import (
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	realip "github.com/ferluci/fast-realip"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

var (
	oidcStateSessKey = "oidc_state"
	oidcNextSessKey  = "oidc_next"
)

// handleOIDCLogin redirects to the OIDC provider for login.
func handleOIDCLogin(r *fastglue.Request) error {
	var (
		app             = r.Context.(*App)
		providerID, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil {
		app.lo.Error("error parsing provider id", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.GeneralError)
	}

	// Set a state and save it in the session, to prevent CSRF attacks.
	state, err := stringutil.RandomAlphanumeric(32)
	if err != nil {
		app.lo.Error("error generating state", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorGenerating", "name", "state"), nil, envelope.GeneralError)
	}

	sessionValues := map[string]any{
		oidcStateSessKey: state,
		// For redirecting after login
		oidcNextSessKey: string(r.RequestCtx.QueryArgs().Peek("next")),
	}

	if err = app.auth.SetSessionValues(r, sessionValues); err != nil {
		app.lo.Error("error saving state in session", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorSaving", "name", "{globals.terms.session}"), nil, envelope.GeneralError)
	}

	authURL, err := app.auth.LoginURL(providerID, state)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.Redirect(authURL, fasthttp.StatusFound, nil, "")
}

// handleOIDCCallback receives the redirect callback from the OIDC provider and completes the handshake.
//
// On any failure we redirect the browser back to the login page with a
// `?login_error=<reason>` query parameter rather than rendering a raw JSON
// envelope, since this endpoint is hit as a top-level browser navigation. The
// SPA login screen then surfaces a friendly message and lets the user retry.
func handleOIDCCallback(r *fastglue.Request) error {
	var (
		app              = r.Context.(*App)
		code             = string(r.RequestCtx.QueryArgs().Peek("code"))
		state            = string(r.RequestCtx.QueryArgs().Peek("state"))
		providerErr      = string(r.RequestCtx.QueryArgs().Peek("error"))
		providerErrDesc  = string(r.RequestCtx.QueryArgs().Peek("error_description"))
		providerIDRaw, _ = r.RequestCtx.UserValue("id").(string)
		ip               = realip.FromRequest(r.RequestCtx)
		country          = string(r.RequestCtx.Request.Header.Peek("CF-IPCountry"))
	)

	// IdP returned an error (user denied consent, app misconfigured, etc.).
	if providerErr != "" {
		app.lo.Error("oidc provider returned error", "error", providerErr, "description", providerErrDesc)
		return redirectToLoginError(r, "cancelled")
	}

	providerID, err := strconv.Atoi(providerIDRaw)
	if err != nil {
		app.lo.Error("error parsing provider id", "error", err, "raw", providerIDRaw)
		return redirectToLoginError(r, "provider_failed")
	}

	// Callback hit without a code/state — most commonly a browser back, refresh,
	// or prefetch of the callback URL after the original handshake already
	// completed. The IdP would otherwise reject with "Missing required parameter: code".
	if code == "" || state == "" {
		app.lo.Error("oidc callback missing required parameters", "have_code", code != "", "have_state", state != "")
		return redirectToLoginError(r, "expired")
	}

	// Compare the state from the session with the state from the query.
	sessionState, err := app.auth.GetSessionValue(r, oidcStateSessKey)
	if err != nil {
		app.lo.Error("error getting state from session", "error", err)
		return redirectToLoginError(r, "expired")
	}
	if state != sessionState {
		app.lo.Error("oidc state mismatch", "session_empty", sessionState == "")
		return redirectToLoginError(r, "expired")
	}

	_, claims, err := app.auth.ExchangeOIDCToken(r.RequestCtx, providerID, code)
	if err != nil {
		app.lo.Error("error exchanging oidc token", "error", err)
		return redirectToLoginError(r, "provider_failed")
	}

	// Lookup the user by email and set the session.
	user, err := app.user.GetAgent(0, claims.Email)
	if err != nil {
		app.lo.Error("oidc user lookup failed", "email", claims.Email, "error", err)
		return redirectToLoginError(r, "unauthorized")
	}

	if err := app.auth.SaveSession(amodels.User{
		ID:        user.ID,
		Email:     user.Email.String,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}, r); err != nil {
		app.lo.Error("oidc save session failed", "user_id", user.ID, "error", err)
		return redirectToLoginError(r, "session_failed")
	}

	// Update last login time.
	if err := app.user.UpdateLastLoginAt(user.ID); err != nil {
		app.lo.Error("oidc update last login failed", "user_id", user.ID, "error", err)
	}

	// Insert activity log.
	if err := app.activityLog.Login(user.ID, user.Email.String, ip, country); err != nil {
		app.lo.Error("error creating login activity log", "error", err)
	}

	// Read the 'next' parameter from session to redirect after login.
	nextParam, _ := app.auth.GetSessionValue(r, oidcNextSessKey)
	redirectURL := "/"
	if nextStr, ok := nextParam.(string); ok && nextStr != "" {
		redirectURL = nextStr
	}

	return r.RedirectURI(redirectURL, fasthttp.StatusFound, nil, "")
}

// redirectToLoginError bounces the browser back to the SPA login screen with a
// machine-readable `login_error` reason. The frontend maps the reason to a
// localised message via auth.loginErrors.*. Keep reason values in sync with the
// switch in UserLoginView.vue.
func redirectToLoginError(r *fastglue.Request, reason string) error {
	return r.RedirectURI("/?login_error="+reason, fasthttp.StatusFound, nil, "")
}
