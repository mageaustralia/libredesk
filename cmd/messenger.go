package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/messenger"
	"github.com/zerodha/fastglue"
)

// handleMetaWebhookVerify handles the GET webhook verification from Meta.
// Meta sends hub.mode=subscribe, hub.verify_token, hub.challenge.
// We check verify_token against all messenger/instagram inboxes and echo back the challenge.
func handleMetaWebhookVerify(r *fastglue.Request) error {
	app := r.Context.(*App)

	mode := string(r.RequestCtx.QueryArgs().Peek("hub.mode"))
	token := string(r.RequestCtx.QueryArgs().Peek("hub.verify_token"))
	challenge := string(r.RequestCtx.QueryArgs().Peek("hub.challenge"))

	if mode != "subscribe" || token == "" || challenge == "" {
		return r.SendErrorEnvelope(http.StatusBadRequest, "invalid verification request", nil, envelope.InputError)
	}

	// Check verify_token against all messenger/instagram inboxes.
	inboxes, err := app.inbox.GetAll()
	if err != nil {
		app.lo.Error("error fetching inboxes for webhook verification", "error", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, "internal error", nil, envelope.GeneralError)
	}

	for _, inb := range inboxes {
		if inb.Channel != inbox.ChannelMessenger && inb.Channel != inbox.ChannelInstagram {
			continue
		}
		var cfg messenger.MetaConfig
		if err := json.Unmarshal(inb.Config, &cfg); err != nil {
			continue
		}
		if cfg.VerifyToken == token {
			// Verification successful - echo back the challenge.
			r.RequestCtx.Response.Header.Set("Content-Type", "text/plain")
			r.RequestCtx.SetBodyString(challenge)
			app.lo.Info("meta webhook verified", "inbox_id", inb.ID, "channel", inb.Channel)
			return nil
		}
	}

	app.lo.Warn("meta webhook verification failed: no matching verify_token")
	return r.SendErrorEnvelope(http.StatusForbidden, "verification failed", nil, envelope.InputError)
}

// handleMetaWebhook handles incoming POST webhook events from Meta (Messenger + Instagram).
func handleMetaWebhook(r *fastglue.Request) error {
	app := r.Context.(*App)

	body := r.RequestCtx.PostBody()
	if len(body) == 0 {
		return r.SendErrorEnvelope(http.StatusBadRequest, "empty body", nil, envelope.InputError)
	}

	// Parse the webhook payload.
	var payload messenger.WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		app.lo.Error("error parsing meta webhook payload", "error", err)
		return r.SendErrorEnvelope(http.StatusBadRequest, "invalid payload", nil, envelope.InputError)
	}

	// Load all messenger/instagram inboxes for matching.
	inboxes, err := app.inbox.GetAll()
	if err != nil {
		app.lo.Error("error fetching inboxes for webhook", "error", err)
		// Return 200 anyway - Meta will retry on non-2xx.
		r.RequestCtx.SetStatusCode(http.StatusOK)
		return nil
	}

	// Build lookup maps: page_id -> inbox, ig_account_id -> inbox.
	type inboxMatch struct {
		inboxID   int
		appSecret string
		channel   string
	}
	lookup := make(map[string]inboxMatch)

	for _, inb := range inboxes {
		if inb.Channel != inbox.ChannelMessenger && inb.Channel != inbox.ChannelInstagram {
			continue
		}
		var cfg messenger.MetaConfig
		if err := json.Unmarshal(inb.Config, &cfg); err != nil {
			continue
		}
		if cfg.PageID != "" {
			lookup[cfg.PageID] = inboxMatch{inboxID: inb.ID, appSecret: cfg.AppSecret, channel: inb.Channel}
		}
		if cfg.IGAccountID != "" {
			lookup[cfg.IGAccountID] = inboxMatch{inboxID: inb.ID, appSecret: cfg.AppSecret, channel: inb.Channel}
		}
	}

	// Process each entry.
	for _, entry := range payload.Entry {
		match, ok := lookup[entry.ID]
		if !ok {
			app.lo.Warn("meta webhook: no inbox found for entry ID", "entry_id", entry.ID)
			continue
		}

		// Verify signature using the matched inbox's app_secret.
		sig := string(r.RequestCtx.Request.Header.Peek("X-Hub-Signature-256"))
		if match.appSecret != "" && sig != "" {
			if !messenger.VerifySignature(match.appSecret, body, sig) {
				app.lo.Warn("meta webhook: signature verification failed", "entry_id", entry.ID)
				continue
			}
		}

		// Get the initialized Messenger inbox instance.
		inboxInstance, err := app.inbox.Get(match.inboxID)
		if err != nil {
			app.lo.Error("error getting inbox instance", "inbox_id", match.inboxID, "error", err)
			continue
		}

		messengerInbox, ok := inboxInstance.(*messenger.Messenger)
		if !ok {
			app.lo.Error("inbox is not a messenger instance", "inbox_id", match.inboxID)
			continue
		}

		// Process each messaging event.
		for _, event := range entry.Messaging {
			if err := messengerInbox.ProcessWebhookEvent(event); err != nil {
				app.lo.Error("error processing messenger event",
					"inbox_id", match.inboxID,
					"sender", event.Sender.ID,
					"error", err)
			}
		}
	}

	// Always return 200 to Meta (they retry on non-2xx).
	r.RequestCtx.SetStatusCode(http.StatusOK)
	fmt.Fprint(r.RequestCtx, "EVENT_RECEIVED")
	return nil
}
