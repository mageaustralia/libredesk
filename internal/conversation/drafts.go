package conversation

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
)

// Draft meta contract — kept in lockstep with the frontend's expectations.
//
// The `meta` column on conversation_drafts is opaque JSON to this package
// (we just round-trip it as a raw message), but BOTH the frontend
// (frontend/apps/main/src/composables/useDraftManager.js) AND any
// post-send handlers that hydrate a draft into a real message need to
// agree on the shape. If you add or rename a field here, update the
// useDraftManager.js companion doc AND the validateMacroActions /
// validateAttachments / isDraftEmpty helpers in that file.
//
//	{
//	  "attachments":   [{ id, size, uuid, filename, content_type }, ...],  // optional
//	  "macro_actions": [{ type, value: [...], display_value: [...] }, ...], // optional
//	}
//
// Both fields are optional. An empty/missing meta is fine — it means
// the draft is text-only. isDraftEmpty (frontend) treats a draft with no
// text AND no attachments AND no macro_actions as deletable.

// UpsertConversationDraft saves or updates a draft for a conversation.
// messageType is "reply" or "private_note"; callers without a preference
// (older API consumers) pass "" and we default to "reply" for back-compat.
func (m *Manager) UpsertConversationDraft(conversationID, userID int, content string, meta json.RawMessage, messageType string) (models.ConversationDraft, error) {
	var draft models.ConversationDraft
	content = rewriteInlineImagesToCID(content)

	if messageType == "" {
		messageType = "reply"
	}

	if err := m.q.UpsertConversationDraft.Get(&draft, conversationID, userID, content, meta, messageType); err != nil {
		m.lo.Error("error upserting conversation draft", "conversation_id", conversationID, "user_id", userID, "message_type", messageType, "error", err)
		return draft, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	draft.Content = m.resolveDraftInlineCIDs(draft.Content)
	return draft, nil
}

func (m *Manager) GetAllUserDrafts(userID int) ([]models.ConversationDraft, error) {
	var drafts = make([]models.ConversationDraft, 0)
	if err := m.q.GetAllUserDrafts.Select(&drafts, userID); err != nil {
		m.lo.Error("error fetching user drafts", "user_id", userID, "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	for i := range drafts {
		drafts[i].Content = m.resolveDraftInlineCIDs(drafts[i].Content)
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
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
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

func (m *Manager) resolveDraftInlineCIDs(content string) string {
	cids := extractInlineContentIDs(content)
	for _, cid := range cids {
		uuid := strings.TrimPrefix(cid, "ldsk-")
		if uuid == "" {
			continue
		}
		media, err := m.mediaStore.Get(0, uuid)
		if err != nil {
			continue
		}
		content = strings.ReplaceAll(content, "cid:"+cid, m.mediaStore.GetURL(media.UUID, media.ContentType, media.Filename))
	}
	return content
}
