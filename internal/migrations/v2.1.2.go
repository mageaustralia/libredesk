package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_1_2 adds the Spam & Trash feature: new conversation statuses, the
// trashed_at column + index, and the auto-cleanup retention settings.
func V2_1_2(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	stmts := []string{
		`ALTER TABLE conversations ADD COLUMN IF NOT EXISTS trashed_at TIMESTAMPTZ NULL`,
		`CREATE INDEX IF NOT EXISTS index_conversations_trashed_at
			ON conversations (trashed_at) WHERE trashed_at IS NOT NULL`,
		`INSERT INTO conversation_statuses (name) VALUES ('Spam') ON CONFLICT (name) DO NOTHING`,
		`INSERT INTO conversation_statuses (name) VALUES ('Trashed') ON CONFLICT (name) DO NOTHING`,
		`INSERT INTO settings (key, value) VALUES ('trash.auto_trash_resolved_days', '90'::jsonb)
			ON CONFLICT (key) DO NOTHING`,
		`INSERT INTO settings (key, value) VALUES ('trash.auto_trash_spam_days', '30'::jsonb)
			ON CONFLICT (key) DO NOTHING`,
		`INSERT INTO settings (key, value) VALUES ('trash.auto_delete_days', '30'::jsonb)
			ON CONFLICT (key) DO NOTHING`,
	}

	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
