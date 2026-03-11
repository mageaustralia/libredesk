package main

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/ai"
	"github.com/abhinavxd/libredesk/internal/envelope"
	settingmodels "github.com/abhinavxd/libredesk/internal/setting/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type aiCompletionReq struct {
	PromptKey string `json:"prompt_key"`
	Content   string `json:"content"`
}

type providerUpdateReq struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
}

type setDefaultProviderReq struct {
	Provider string `json:"provider"`
}

type testProviderReq struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
}

// handleAICompletion handles AI completion requests
func handleAICompletion(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = aiCompletionReq{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil))
	}

	resp, err := app.ai.Completion(req.PromptKey, req.Content)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(resp)
}

// handleGetAIPrompts returns AI prompts
func handleGetAIPrompts(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	resp, err := app.ai.GetPrompts()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(resp)
}

// handleGetAIProviders returns configured AI providers
func handleGetAIProviders(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	resp, err := app.ai.GetProviders()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(resp)
}

// handleGetAvailableModels returns available models for OpenRouter
func handleGetAvailableModels(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	models := app.ai.GetAvailableModels()
	return r.SendEnvelope(models)
}

// handleUpdateAIProvider updates the AI provider
func handleUpdateAIProvider(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req providerUpdateReq
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil))
	}
	if err := app.ai.UpdateProvider(req.Provider, req.APIKey, req.Model); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope("Provider updated successfully")
}

// handleSetDefaultAIProvider sets the default AI provider
func handleSetDefaultAIProvider(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req setDefaultProviderReq
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil))
	}
	if err := app.ai.SetDefaultProvider(req.Provider); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope("Default provider updated successfully")
}

// handleTestAIProvider tests the AI provider connection
func handleTestAIProvider(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req testProviderReq
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil))
	}
	if err := app.ai.TestProvider(req.Provider, req.APIKey, req.Model); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope("Connection successful")
}

// handleGetSupportedProviders returns list of supported AI provider types
func handleGetSupportedProviders(r *fastglue.Request) error {
	return r.SendEnvelope(ai.SupportedProviders)
}

// handleGetInboxAISettings returns AI settings for a specific inbox.
func handleGetInboxAISettings(r *fastglue.Request) error {
	app := r.Context.(*App)

	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid inbox ID", nil, envelope.InputError)
	}

	out, err := app.setting.GetInboxAISettings(id)
	if err != nil {
		// Not found — return empty struct so the frontend knows to show global defaults
		return r.SendEnvelope(settingmodels.InboxAISettings{InboxID: id})
	}
	return r.SendEnvelope(out)
}

// handleUpdateInboxAISettings creates or updates AI settings for an inbox.
func handleUpdateInboxAISettings(r *fastglue.Request) error {
	app := r.Context.(*App)

	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid inbox ID", nil, envelope.InputError)
	}

	var req settingmodels.InboxAISettings
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil))
	}
	req.InboxID = id

	out, err := app.setting.UpsertInboxAISettings(req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(out)
}

// handleDeleteInboxAISettings removes per-inbox AI settings (falls back to global).
func handleDeleteInboxAISettings(r *fastglue.Request) error {
	app := r.Context.(*App)

	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid inbox ID", nil, envelope.InputError)
	}

	if err := app.setting.DeleteInboxAISettings(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope("Inbox AI settings deleted")
}
