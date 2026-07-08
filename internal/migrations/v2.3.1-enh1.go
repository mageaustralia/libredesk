package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_3_1_Enh1 backs the scoped/ranked unified search (FS26). It adds:
//   - a generated tsvector column on conversation_messages for word-level
//     full-text search (replacing the unindexed body ILIKE scan),
//   - a GIN index on that column,
//   - trigram GIN indexes on the contact full name and conversation subject.
//
// Fork-only enhancement with no upstream counterpart. Versioned as the semver
// pre-release v2.3.1-enh.1 so it sorts after v2.3.0 but before a future
// upstream v2.3.1 hop, avoiding a collision at merge time.
//
// All statements are IF NOT EXISTS / idempotent. The ALTER rewrites the
// conversation_messages table once (seconds at typical volumes); it runs in the
// upgrade maintenance window, so plain CREATE INDEX (not CONCURRENTLY) is fine.
func V2_3_1_Enh1(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	stmts := []string{
		`ALTER TABLE conversation_messages
		   ADD COLUMN IF NOT EXISTS text_content_tsv TSVECTOR
		   GENERATED ALWAYS AS (to_tsvector('english', left(coalesce(text_content, ''), 100000))) STORED`,
		`CREATE INDEX IF NOT EXISTS index_fts_conversation_messages_on_text_content
		   ON conversation_messages USING GIN (text_content_tsv)`,
		`CREATE INDEX IF NOT EXISTS index_trgm_users_on_full_name
		   ON users USING GIN ((first_name || ' ' || last_name) gin_trgm_ops)`,
		`CREATE INDEX IF NOT EXISTS index_trgm_conversations_on_subject
		   ON conversations USING GIN (subject gin_trgm_ops)`,
	}
	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
