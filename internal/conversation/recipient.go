package conversation

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
)

// makeRecipients computes the recipients for a given conversation ID using the
// last human message in the conversation. Auto-generated diagnostic mail (DSN
// bounces from mailer-daemon/postmaster) is skipped so a bounce sender never
// becomes the reply "To".
func (m *Manager) makeRecipients(conversationID int, contactEmail, inboxEmail, inboxReplyTo string) (to, cc, bcc []string, err error) {
	lastMessage, err := m.getLatestMessageForRecipients(conversationID, []string{models.MessageIncoming, models.MessageOutgoing}, []string{models.MessageStatusReceived, models.MessageStatusSent}, true)
	if err != nil {
		// Only bounces (or no messages) to derive from — fall back to the
		// conversation contact rather than failing the reply.
		if err == sql.ErrNoRows {
			if contactEmail != "" {
				return []string{contactEmail}, nil, nil, nil
			}
			return nil, nil, nil, nil
		}
		return nil, nil, nil, fmt.Errorf("fetching message for makeRecipients: %w", err)
	}

	var meta struct {
		From []string `json:"from"`
		To   []string `json:"to"`
		CC   []string `json:"cc"`
		BCC  []string `json:"bcc"`
	}
	if err = json.Unmarshal(lastMessage.Meta, &meta); err != nil {
		return nil, nil, nil, err
	}

	isIncoming := lastMessage.Type == models.MessageIncoming
	to, cc, bcc = stringutil.ComputeRecipients(
		meta.From, meta.To, meta.CC, meta.BCC, contactEmail, inboxEmail, inboxReplyTo, isIncoming,
	)
	return
}
