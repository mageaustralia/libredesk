package conversation

import (
	"context"
	"fmt"
	"time"

	pciscrub "github.com/mageaustralia/go-pci-scrub"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	notifier "github.com/abhinavxd/libredesk/internal/notification"
	nmodels "github.com/abhinavxd/libredesk/internal/notification/models"
	"github.com/volatiletech/null/v9"
)

// PCIRedactMessage holds the info needed to redact a message.
type PCIRedactMessage struct {
	ID               int         `db:"id"`
	UUID             string      `db:"uuid"`
	Content          string      `db:"content"`
	TextContent      string      `db:"text_content"`
	SourceID         null.String `db:"source_id"`
	ConversationID   int         `db:"conversation_id"`
	ConversationUUID string      `db:"conversation_uuid"`
	InboxID          int         `db:"inbox_id"`
}

// RedactMessagePCI scrubs PCI data from a message's content and text_content.
// Returns the redacted message info (including inbox_id and source_id for IMAP deletion).
func (m *Manager) RedactMessagePCI(msgUUID string) (PCIRedactMessage, error) {
	// Get the message with inbox info.
	var msg PCIRedactMessage
	if err := m.q.GetMessageForRedact.Get(&msg, msgUUID); err != nil {
		return msg, envelope.NewError(envelope.GeneralError, "Message not found", nil)
	}

	// Scrub the content.
	scrubbedContent := pciscrub.Scrub(msg.Content)
	scrubbedText := pciscrub.Scrub(msg.TextContent)

	// Update the message in DB.
	var result struct {
		ID             int         `db:"id"`
		ConversationID int         `db:"conversation_id"`
		SourceID       null.String `db:"source_id"`
		Type           string      `db:"type"`
	}
	if err := m.q.RedactMessagePCI.Get(&result, msgUUID, scrubbedContent, scrubbedText); err != nil {
		m.lo.Error("error redacting PCI data from message", "error", err, "message_uuid", msgUUID)
		return msg, envelope.NewError(envelope.GeneralError, "Failed to redact message", nil)
	}

	m.lo.Info("PCI data redacted from message", "message_uuid", msgUUID, "conversation_uuid", msg.ConversationUUID)
	return msg, nil
}

// InsertPCIRedactActivityNote adds an activity note about PCI redaction to the conversation.
func (m *Manager) InsertPCIRedactActivityNote(conversationUUID string, actorName string, success bool, detail string) {
	noteContent := fmt.Sprintf("PCI data redacted by %s", actorName)
	if !success {
		noteContent = fmt.Sprintf("PCI data redacted by %s. %s", actorName, detail)
	}
	msg := &models.Message{
		Type:             models.MessageActivity,
		Status:           models.MessageStatusSent,
		ConversationUUID: conversationUUID,
		Content:          noteContent,
		TextContent:      noteContent,
		ContentType:      models.ContentTypeText,
		Private:          true,
		SenderID:         1, // system user
		SenderType:       models.SenderTypeAgent,
	}
	m.InsertMessage(msg)
}


// NotifyPCIIMAPDeleteFailed sends a notification to the configured PCI admin
// when an IMAP delete fails after PCI redaction.
func (m *Manager) NotifyPCIIMAPDeleteFailed(conversationUUID string, msgUUID string) {
	if m.dispatcher == nil {
		return
	}

	settings, err := m.settingsStore.GetPCISettings()
	if err != nil || settings.NotifyAgentID == 0 {
		return
	}

	agent, err := m.userStore.GetAgent(settings.NotifyAgentID, "")
	if err != nil {
		m.lo.Error("error fetching PCI notify agent", "error", err, "agent_id", settings.NotifyAgentID)
		return
	}

	// Get conversation details for the notification
	rootURL, _ := m.settingsStore.GetAppRootURL()
	ticketLink := fmt.Sprintf("%s/inboxes/all/conversation/%s", rootURL, conversationUUID)
	subject := ""
	contactEmail := ""
	conv, err := m.GetConversation(0, conversationUUID, "")
	if err == nil {
		subject = conv.Subject.String
		if conv.Contact.Email.Valid {
			contactEmail = conv.Contact.Email.String
		}
	}

	title := "PCI: Failed to delete email from Gmail"
	if subject != "" {
		title = fmt.Sprintf("PCI: Failed to delete email — %s", subject)
	}

	bodyText := fmt.Sprintf("Card data was redacted but the original email could not be deleted from Gmail. Please delete it manually.\n\nSubject: %s\nFrom: %s\nTicket: %s", subject, contactEmail, ticketLink)
	bodyHTML := fmt.Sprintf("<p>Card data was redacted but the original email could not be deleted from Gmail. Please delete it manually.</p><p><strong>Subject:</strong> %s<br><strong>From:</strong> %s<br><strong>Ticket:</strong> <a href=\"%s\">%s</a></p>", subject, contactEmail, ticketLink, ticketLink)

	n := notifier.Notification{
		Type:             nmodels.NotificationType("pci_imap_delete_failed"),
		RecipientIDs:     []int{settings.NotifyAgentID},
		Title:            title,
		Body:             null.StringFrom(bodyText),
		ConversationUUID: conversationUUID,
	}

	method := settings.NotifyMethod
	if method == "" {
		method = "both"
	}

	switch method {
	case "in_app":
		m.dispatcher.Send(n)
	case "email":
		if agent.Email.Valid && agent.Email.String != "" {
			n.Email = &notifier.EmailNotification{
				Recipients: []string{agent.Email.String},
				Subject:    title,
				Content:    bodyHTML,
			}
			m.dispatcher.Send(n)
		}
	case "both":
		if agent.Email.Valid && agent.Email.String != "" {
			n.Email = &notifier.EmailNotification{
				Recipients: []string{agent.Email.String},
				Subject:    title,
				Content:    bodyHTML,
			}
		}
		m.dispatcher.Send(n)
	}
}
// RunPCIAutoRedact runs the PCI auto-redaction routine daily.
// Messages with PCI data older than 7 days are automatically scrubbed.
func (m *Manager) RunPCIAutoRedact(ctx context.Context, deleteIMAPFunc func(sourceID string, inboxID int) error) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.runPCIAutoRedactCycle(ctx, deleteIMAPFunc)
		}
	}
}

func (m *Manager) runPCIAutoRedactCycle(ctx context.Context, deleteIMAPFunc func(sourceID string, inboxID int) error) {
	var messages []PCIRedactMessage
	if err := m.q.GetPCIMessagesForAutoRedact.SelectContext(ctx, &messages); err != nil {
		m.lo.Error("error fetching PCI messages for auto-redact", "error", err)
		return
	}

	if len(messages) == 0 {
		return
	}

	m.lo.Info(fmt.Sprintf("auto-redacting PCI data from %d messages", len(messages)))

	for _, msg := range messages {
		scrubbedContent := pciscrub.Scrub(msg.Content)
		scrubbedText := pciscrub.Scrub(msg.TextContent)

		var result struct {
			ID             int         `db:"id"`
			ConversationID int         `db:"conversation_id"`
			SourceID       null.String `db:"source_id"`
			Type           string      `db:"type"`
		}
		if err := m.q.RedactMessagePCI.Get(&result, msg.UUID, scrubbedContent, scrubbedText); err != nil {
			m.lo.Error("error auto-redacting PCI message", "error", err, "message_uuid", msg.UUID)
			continue
		}

		m.lo.Info("auto-redacted PCI data from message", "message_uuid", msg.UUID)

		// Try to delete from IMAP.
		if msg.SourceID.Valid && msg.SourceID.String != "" && deleteIMAPFunc != nil {
			if err := deleteIMAPFunc(msg.SourceID.String, msg.InboxID); err != nil {
				m.lo.Error("failed to delete PCI email from IMAP", "error", err, "source_id", msg.SourceID.String)
				m.InsertPCIRedactActivityNote(msg.ConversationUUID, "System (auto-redact)", false,
					"Card data was redacted but the original email could not be deleted from Gmail. Please delete manually.")
				m.NotifyPCIIMAPDeleteFailed(msg.ConversationUUID, msg.UUID)
			} else {
				m.InsertPCIRedactActivityNote(msg.ConversationUUID, "System (auto-redact)", true, "")
			}
		} else {
			m.InsertPCIRedactActivityNote(msg.ConversationUUID, "System (auto-redact)", true, "")
		}
	}
}
