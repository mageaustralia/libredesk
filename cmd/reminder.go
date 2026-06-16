package main

import (
	"strconv"
	"time"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// reminderCreateReq is the payload accepted by handleCreateReminder.
type reminderCreateReq struct {
	// ISO 8601 / RFC 3339 timestamp. Must be in the future.
	RemindAt string `json:"remind_at"`
	Note     string `json:"note"`
}

// handleListConversationReminders returns the requesting agent's pending
// reminders for a specific conversation (by UUID). Private to the agent —
// each user only sees their own.
func handleListConversationReminders(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		user = r.RequestCtx.UserValue("user").(amodels.User)
		uuid = r.RequestCtx.UserValue("uuid").(string)
	)
	if uuid == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`uuid`"), nil, envelope.InputError)
	}
	out, err := app.reminder.ListForConversation(user.ID, uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(out)
}

// handleListMyReminders returns every pending reminder owned by the
// requesting agent across all conversations. Used by a future "My
// reminders" sidebar view.
func handleListMyReminders(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		user = r.RequestCtx.UserValue("user").(amodels.User)
	)
	out, err := app.reminder.ListForUser(user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(out)
}

// handleCreateReminder sets a personal reminder on a conversation.
func handleCreateReminder(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		user = r.RequestCtx.UserValue("user").(amodels.User)
		uuid = r.RequestCtx.UserValue("uuid").(string)
		req  reminderCreateReq
	)
	if uuid == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`uuid`"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), err.Error(), envelope.InputError)
	}
	if req.RemindAt == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`remind_at`"), nil, envelope.InputError)
	}

	// Parse remind_at; require strict RFC 3339 so we don't accept timezone-less
	// strings that would later get misinterpreted by the worker's NOW() compare.
	remindAt, err := time.Parse(time.RFC3339, req.RemindAt)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`remind_at`"), nil, envelope.InputError)
	}
	// Reject obviously stale picks — gives a 1-minute grace so a slow agent
	// can still set "in 1 minute" without server-clock skew kicking it back.
	if remindAt.Before(time.Now().Add(-1 * time.Minute)) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`remind_at`"), nil, envelope.InputError)
	}
	if len(req.Note) > 500 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`note`"), nil, envelope.InputError)
	}

	convID, err := app.reminder.ResolveConversationIDByUUID(uuid)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.conversation}"), nil, envelope.InputError)
	}

	created, err := app.reminder.Create(user.ID, convID, remindAt, req.Note)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(created)
}

// handleDeleteReminder removes a reminder. Only the owning user may delete
// theirs — the query enforces user_id == caller.
func handleDeleteReminder(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		user = r.RequestCtx.UserValue("user").(amodels.User)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	if err := app.reminder.Delete(user.ID, id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
