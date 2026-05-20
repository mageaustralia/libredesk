package main

import (
	"encoding/json"
	"strings"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

const maxMetaSize = 32 * 1024 // 32KB

type draftReq struct {
	Content string          `json:"content"`
	Meta    json.RawMessage `json:"meta"`
	// MessageType discriminates a reply draft from a private-note draft
	// so an agent can have one of each on the same conversation without
	// them clobbering each other. Empty string = "reply" (back-compat
	// for any older client that doesn't know about per-type drafts).
	MessageType string `json:"message_type"`
}

// validDraftMessageType reports whether the supplied type is one the
// schema permits. Empty is allowed because the manager defaults to
// "reply" — handy for clients that pre-date the per-type drafts work.
func validDraftMessageType(t string) bool {
	return t == "" || t == "reply" || t == "private_note"
}

// handleUpsertConversationDraft saves or updates a draft for a conversation.
func handleUpsertConversationDraft(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		req   = draftReq{}
	)

	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check access to conversation.
	conv, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error decoding draft request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	if len(req.Meta) > maxMetaSize {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "meta"), nil, envelope.InputError)
	}

	if !validDraftMessageType(req.MessageType) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "message_type"), nil, envelope.InputError)
	}

	// Validate content is not empty
	if strings.TrimSpace(req.Content) == "" && (len(req.Meta) == 0 || string(req.Meta) == "{}") {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "content"), nil, envelope.InputError)
	}

	draft, err := app.conversation.UpsertConversationDraft(conv.ID, user.ID, req.Content, req.Meta, req.MessageType)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(draft)
}

// handleGetAllDrafts retrieves all drafts for the current user.
func handleGetAllDrafts(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)

	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	drafts, err := app.conversation.GetAllUserDrafts(user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(drafts)
}

// handleDeleteConversationDraft deletes a draft for a conversation.
//
// Optional ?message_type=reply|private_note query param scopes the delete
// to a single tab's draft. Omitted = delete BOTH types — used by the
// post-send path that wants to clear everything on the conversation.
func handleDeleteConversationDraft(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		auser       = r.RequestCtx.UserValue("user").(amodels.User)
		uuid        = r.RequestCtx.UserValue("uuid").(string)
		messageType = string(r.RequestCtx.QueryArgs().Peek("message_type"))
	)

	if !validDraftMessageType(messageType) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "message_type"), nil, envelope.InputError)
	}

	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := app.conversation.DeleteConversationDraft(0, uuid, user.ID, messageType); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}
