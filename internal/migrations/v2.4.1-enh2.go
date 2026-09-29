package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_4_1_Enh2 adds conversations.contact_updated_at, which records when an
// agent reassigned a conversation to a different contact.
//
// Reply recipients are derived from the latest message's headers, which is
// wrong for a ticket forwarded in by a third party: the agent repoints the
// conversation at the real customer, but replies keep going to the forwarder.
// The UI compares this timestamp against the latest message to know the agent's
// choice should win. It previously lived in a browser-memory ref, so it was
// lost on reload and invisible to every other agent.
//
// Fork-only enhancement with no upstream counterpart; versioned as the semver
// pre-release v2.4.1-enh.2 so it sorts after enh.1 but before a future upstream
// v2.4.1 hop.
func V2_4_1_Enh2(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`ALTER TABLE conversations ADD COLUMN IF NOT EXISTS contact_updated_at TIMESTAMPTZ NULL`)
	return err
}
