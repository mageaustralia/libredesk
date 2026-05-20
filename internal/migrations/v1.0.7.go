package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V1_0_7 adds message_type to conversation_drafts so an agent can have a
// reply draft AND a private-note draft coexist on the same conversation
// without one overwriting the other.
//
// Before this migration: drafts were keyed by (conversation_id, user_id),
// so typing in the Private Note tab then later switching to Reply (or
// vice versa) made the saved content "leak" into the wrong tab — there
// was only one draft slot per (conversation, user).
//
// After: keyed by (conversation_id, user_id, message_type) where
// message_type is 'reply' or 'private_note'. Existing rows backfill to
// 'reply' which matches their semantics (the old default was reply).
func V1_0_7(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
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
