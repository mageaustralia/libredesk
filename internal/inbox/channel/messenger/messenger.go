// Package messenger implements the Meta Messenger and Instagram DM channel
// for Libredesk inboxes.
package messenger

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/inbox"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

// Messenger implements the inbox.Inbox interface for Facebook Messenger and Instagram DMs.
type Messenger struct {
	id      int
	channel string // "messenger" or "instagram"
	config  MetaConfig
	lo      *logf.Logger

	msgStore inbox.MessageStore
	usrStore inbox.UserStore
}

// Opts holds initialization options.
type Opts struct {
	ID      int
	Channel string
	Config  MetaConfig
	Lo      *logf.Logger
}

// New creates a new Messenger inbox.
func New(msgStore inbox.MessageStore, usrStore inbox.UserStore, opts Opts) (*Messenger, error) {
	if opts.Config.PageAccessToken == "" {
		return nil, fmt.Errorf("page_access_token is required for %s inbox", opts.Channel)
	}
	if opts.Config.PageID == "" && opts.Config.IGAccountID == "" {
		return nil, fmt.Errorf("page_id or ig_account_id is required")
	}

	return &Messenger{
		id:       opts.ID,
		channel:  opts.Channel,
		config:   opts.Config,
		lo:       opts.Lo,
		msgStore: msgStore,
		usrStore: usrStore,
	}, nil
}

// Identifier returns the inbox DB ID.
func (m *Messenger) Identifier() int { return m.id }

// Channel returns "messenger" or "instagram".
func (m *Messenger) Channel() string { return m.channel }

// FromAddress returns the Page ID (or IG account ID).
func (m *Messenger) FromAddress() string {
	if m.channel == "instagram" && m.config.IGAccountID != "" {
		return m.config.IGAccountID
	}
	return m.config.PageID
}

// Config returns the MetaConfig.
func (m *Messenger) Config() MetaConfig { return m.config }

// Receive blocks until context is cancelled. Messenger is webhook-driven, no polling.
func (m *Messenger) Receive(ctx context.Context) error {
	m.lo.Info("messenger inbox receiver started (webhook-driven, no polling)", "id", m.id, "channel", m.channel)
	<-ctx.Done()
	return nil
}

// Send sends a message via the Meta Send API.
// HTML is stripped to plain text. Attachments are sent as separate messages.
func (m *Messenger) Send(msg models.Message) error {
	// Extract recipient PSID/IGSID from message meta.
	recipientID, err := m.extractRecipientID(msg)
	if err != nil {
		return fmt.Errorf("extracting recipient ID: %w", err)
	}

	// Strip HTML tags to get plain text.
	text := stripHTML(msg.Content)
	if text == "" && len(msg.Attachments) == 0 {
		return fmt.Errorf("empty message: no text or attachments")
	}

	// Send text message.
	if text != "" {
		msgID, err := SendTextMessage(m.config.PageAccessToken, recipientID, text)
		if err != nil {
			return fmt.Errorf("sending text message: %w", err)
		}
		m.lo.Debug("sent text message", "message_id", msgID, "recipient", recipientID)
	}

	// Send attachments as separate messages.
	for _, att := range msg.Attachments {
		attType := mapAttachmentType(att.ContentType)
		if att.URL == "" {
			continue
		}
		msgID, err := SendAttachment(m.config.PageAccessToken, recipientID, attType, att.URL)
		if err != nil {
			m.lo.Error("error sending attachment", "error", err, "url", att.URL)
			continue
		}
		m.lo.Debug("sent attachment", "message_id", msgID, "type", attType)
	}

	return nil
}

// Close is a no-op for webhook-based channels.
func (m *Messenger) Close() error { return nil }

// ProcessWebhookEvent processes a single messaging event from a webhook.
func (m *Messenger) ProcessWebhookEvent(event MessagingEvent) error {
	// Skip non-message events (delivery, read receipts, postbacks).
	if event.Message == nil {
		return nil
	}

	// Skip echo messages (messages sent by the page itself).
	if event.Message.IsEcho {
		return nil
	}

	senderID := event.Sender.ID

	// Check if message already exists (dedup by MID).
	exists, err := m.msgStore.MessageExists(event.Message.MID)
	if err != nil {
		return fmt.Errorf("checking message existence: %w", err)
	}
	if exists {
		return nil
	}

	// Fetch sender profile.
	profile, err := GetUserProfile(m.config.PageAccessToken, senderID)
	if err != nil {
		m.lo.Warn("could not fetch sender profile, using fallback", "sender_id", senderID, "error", err)
		profile = UserProfile{FirstName: "Messenger", LastName: "User"}
	}

	// Build contact. Use PSID/IGSID as synthetic email for contact dedup.
	syntheticEmail := fmt.Sprintf("%s@%s.meta.local", senderID, m.channel)

	contact := umodels.User{
		FirstName:       profile.FirstName,
		LastName:        profile.LastName,
		Email:           null.StringFrom(syntheticEmail),
		AvatarURL:       null.StringFrom(profile.ProfilePic),
		Type:            umodels.UserTypeContact,
		InboxID:         m.id,
		SourceChannelID: null.StringFrom(senderID),
	}

	// Build message content.
	content := event.Message.Text

	// Build message meta with sender/recipient IDs.
	meta, _ := json.Marshal(map[string]interface{}{
		"sender_id":    senderID,
		"recipient_id": event.Recipient.ID,
		"channel":      m.channel,
	})

	msg := models.Message{
		Type:        models.MessageIncoming,
		Status:      models.MessageStatusReceived,
		Content:     content,
		ContentType: "text",
		SourceID:    null.StringFrom(event.Message.MID),
		Channel:     m.channel,
		SenderType:  models.SenderTypeContact,
		InboxID:     m.id,
		Meta:        meta,
	}

	// Handle attachments - download immediately (temporary URLs).
	for _, att := range event.Message.Attachments {
		if att.Payload.URL == "" {
			// For stickers, locations etc, add as text.
			if att.Type == "sticker" {
				msg.Content += " [sticker]"
			} else if att.Type == "location" {
				msg.Content += " [location]"
			}
			continue
		}

		// Download the attachment data.
		data, contentType, dlErr := DownloadAttachment(att.Payload.URL)
		if dlErr != nil {
			m.lo.Error("error downloading attachment", "type", att.Type, "error", dlErr)
			continue
		}

		ext := extensionFromType(att.Type, contentType)
		msg.Attachments = append(msg.Attachments, attachment.Attachment{
			Name:        fmt.Sprintf("attachment%s", ext),
			Content:     data,
			ContentType: contentType,
			Size:        len(data),
		})
	}

	incoming := models.IncomingMessage{
		Message: msg,
		Contact: contact,
		InboxID: m.id,
	}

	if err := m.msgStore.EnqueueIncoming(incoming); err != nil {
		return fmt.Errorf("enqueueing incoming message: %w", err)
	}

	m.lo.Info("enqueued incoming message", "sender", senderID, "mid", event.Message.MID, "channel", m.channel)
	return nil
}

// extractRecipientID gets the PSID/IGSID from the message meta.
func (m *Messenger) extractRecipientID(msg models.Message) (string, error) {
	if len(msg.Meta) == 0 {
		return "", fmt.Errorf("no meta in message")
	}

	var meta map[string]interface{}
	if err := json.Unmarshal(msg.Meta, &meta); err != nil {
		return "", fmt.Errorf("parsing message meta: %w", err)
	}

	// The sender_id of the original incoming message is who we reply to.
	if id, ok := meta["sender_id"].(string); ok && id != "" {
		return id, nil
	}

	// Fallback: extract from the To field.
	if len(msg.To) > 0 {
		return msg.To[0], nil
	}

	return "", fmt.Errorf("could not determine recipient PSID/IGSID")
}

// stripHTML removes HTML tags from content, returning plain text.
func stripHTML(s string) string {
	var out strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			out.WriteRune(r)
		}
	}
	return strings.TrimSpace(out.String())
}

// mapAttachmentType maps MIME content types to Meta attachment types.
func mapAttachmentType(contentType string) string {
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return "image"
	case strings.HasPrefix(contentType, "video/"):
		return "video"
	case strings.HasPrefix(contentType, "audio/"):
		return "audio"
	default:
		return "file"
	}
}

// extensionFromType returns a file extension for an attachment type.
func extensionFromType(attType, contentType string) string {
	switch attType {
	case "image":
		if strings.Contains(contentType, "png") {
			return ".png"
		}
		if strings.Contains(contentType, "gif") {
			return ".gif"
		}
		return ".jpg"
	case "video":
		return ".mp4"
	case "audio":
		return ".mp3"
	default:
		return ".bin"
	}
}
