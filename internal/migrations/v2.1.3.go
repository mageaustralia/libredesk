package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_1_3 adds ticket merging support: the merged_into_conversation_id
// column tracks which primary a merged-secondary points to. Deleting a
// primary unsets the link on its secondaries (ON DELETE SET NULL) so we
// never cascade-delete merged history.
func V2_1_3(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	stmts := []string{
		`ALTER TABLE conversations ADD COLUMN IF NOT EXISTS merged_into_conversation_id BIGINT
			REFERENCES conversations(id) ON DELETE SET NULL`,
		`ALTER TABLE conversations ADD COLUMN IF NOT EXISTS merged_at TIMESTAMPTZ NULL`,
		`CREATE INDEX IF NOT EXISTS index_conversations_merged_into
			ON conversations (merged_into_conversation_id)
			WHERE merged_into_conversation_id IS NOT NULL`,
	}
	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
