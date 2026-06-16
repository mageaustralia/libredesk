package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_2_25 adds the conversation_reminders table for per-agent personal
// follow-up reminders on individual conversations.
func V2_2_25(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS conversation_reminders (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			user_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			remind_at TIMESTAMPTZ NOT NULL,
			fired_at TIMESTAMPTZ NULL,
			note TEXT NOT NULL DEFAULT '',
			CONSTRAINT constraint_conversation_reminders_note_length CHECK (length(note) <= 500)
		);
		CREATE INDEX IF NOT EXISTS index_conversation_reminders_due
			ON conversation_reminders(remind_at) WHERE fired_at IS NULL;
		CREATE INDEX IF NOT EXISTS index_conversation_reminders_on_user_id
			ON conversation_reminders(user_id);
		CREATE INDEX IF NOT EXISTS index_conversation_reminders_on_conversation_id
			ON conversation_reminders(conversation_id);
	`)
	return err
}
