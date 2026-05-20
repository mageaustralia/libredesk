package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_2_20 ports the per-type drafts schema change from v1.0.3's v1.0.7
// migration. Adds message_type to conversation_drafts so reply +
// private_note drafts coexist per (conversation, user) without
// clobbering each other.
//
// Before: drafts were keyed by (conversation_id, user_id) — one slot
// per conversation. Typing in Private Note then later switching to
// Reply made the saved content leak between tabs because both editors
// shared the same draft slot.
//
// After: keyed by (conversation_id, user_id, message_type). Existing
// rows backfill to 'reply' (the old default semantic).
func V2_2_20(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		ALTER TABLE conversation_drafts
		  ADD COLUMN IF NOT EXISTS message_type TEXT NOT NULL DEFAULT 'reply';
	`); err != nil {
		return err
	}
	// The old unique constraint is enforced via an index named
	// `index_uniq_conversation_drafts_on_conversation_id_and_user_id`.
	// Drop it so we can add the wider 3-column unique index.
	if _, err := db.Exec(`
		DROP INDEX IF EXISTS index_uniq_conversation_drafts_on_conversation_id_and_user_id;
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS index_uniq_conversation_drafts_on_conv_user_type
		  ON conversation_drafts(conversation_id, user_id, message_type);
	`); err != nil {
		return err
	}
	return nil
}
