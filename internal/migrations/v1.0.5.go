package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V1_0_5 adds per-agent email signature column.
func V1_0_5(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE users ADD COLUMN IF NOT EXISTS signature TEXT NOT NULL DEFAULT '';
	`)
	return err
}
