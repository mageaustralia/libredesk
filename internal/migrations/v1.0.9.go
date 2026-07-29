package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V1_0_9 adds the send_failure user notification type, raised for the
// sending agent when an outgoing message fails to deliver.
func V1_0_9(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`ALTER TYPE user_notification_type ADD VALUE IF NOT EXISTS 'send_failure'`)
	return err
}
