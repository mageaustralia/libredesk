package conversation

import (
	"context"
	"encoding/json"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
)

// UpsertConversationDraft saves or updates a draft for a conversation.
// messageType is "reply" or "private_note"; callers without a preference
// (older API consumers) pass "" and we default to "reply" for back-compat.
func (m *Manager) UpsertConversationDraft(conversationID, userID int, content string, meta json.RawMessage, messageType string) (models.ConversationDraft, error) {
	var draft models.ConversationDraft

	if messageType == "" {
		messageType = "reply"
	}

	if err := m.q.UpsertConversationDraft.Get(&draft, conversationID, userID, content, meta, messageType); err != nil {
		m.lo.Error("error upserting conversation draft", "conversation_id", conversationID, "user_id", userID, "message_type", messageType, "error", err)
		return draft, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "draft"), nil)
	}

	return draft, nil
}

// GetAllUserDrafts retrieves all drafts for a user.
func (m *Manager) GetAllUserDrafts(userID int) ([]models.ConversationDraft, error) {
	var drafts = make([]models.ConversationDraft, 0)
	if err := m.q.GetAllUserDrafts.Select(&drafts, userID); err != nil {
		m.lo.Error("error fetching user drafts", "user_id", userID, "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "drafts"), nil)
	}
	return drafts, nil
}

// DeleteConversationDraft deletes a draft for a conversation by ID or UUID.
// messageType ("reply", "private_note", or "" for ALL types) lets the
// caller delete either a specific type's draft (the post-send / cancel
// path) or every draft on the conversation (the "clear all" path).
func (m *Manager) DeleteConversationDraft(conversationID int, uuid string, userID int, messageType string) error {
	var uuidParam any
	if uuid != "" {
		uuidParam = uuid
	}

	var typeParam any
	if messageType != "" {
		typeParam = messageType
	}

	if _, err := m.q.DeleteConversationDraft.Exec(conversationID, uuidParam, userID, typeParam); err != nil {
		m.lo.Error("error deleting conversation draft", "conversation_id", conversationID, "uuid", uuid, "user_id", userID, "message_type", messageType, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "draft"), nil)
	}

	return nil
}

// DeleteStaleDrafts deletes drafts older than the specified retention period.
func (m *Manager) DeleteStaleDrafts(ctx context.Context, retentionPeriod time.Duration) error {
	cutoff := time.Now().Add(-retentionPeriod)
	res, err := m.q.DeleteStaleDrafts.ExecContext(ctx, cutoff)
	if err != nil {
		m.lo.Error("error deleting stale drafts", "error", err)
		return err
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected > 0 {
		m.lo.Info("deleted stale drafts", "count", rowsAffected)
	}

	return nil
}
