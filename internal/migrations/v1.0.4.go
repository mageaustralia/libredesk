package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V1_0_4 adds messenger and instagram channel types.
func V1_0_4(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TYPE channels ADD VALUE IF NOT EXISTS 'messenger';
		ALTER TYPE channels ADD VALUE IF NOT EXISTS 'instagram';
	`)
	return err
}
