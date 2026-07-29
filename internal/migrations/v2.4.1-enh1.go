package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_4_1_Enh1 adds the send_failure user notification type, raised for the
// sending agent when an outgoing message fails to deliver.
//
// Fork-only enhancement with no upstream counterpart. Versioned as the semver
// pre-release v2.4.1-enh.1 so it sorts after the v2.4.0 base but before a
// future upstream v2.4.1 hop, avoiding a collision at merge time.
func V2_4_1_Enh1(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`ALTER TYPE user_notification_type ADD VALUE IF NOT EXISTS 'send_failure'`)
	return err
}
