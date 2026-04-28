package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_2_1 extends the upstream conversation_status_category enum (added in
// v2.2.0 as 'open','waiting','resolved') with our fork-specific categories
// for Spam and Trashed, and assigns the existing rows to those categories.
//
// Without this, our pre-existing Spam and Trashed conversation_statuses rows
// (added in v2.1.2) would inherit the v2.2.0 default of 'open' which is wrong
// — they would then leak back into the open queue once any code path filters
// by category.
func V2_2_1(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`ALTER TYPE conversation_status_category ADD VALUE IF NOT EXISTS 'spam';`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TYPE conversation_status_category ADD VALUE IF NOT EXISTS 'trashed';`); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE conversation_statuses SET category = 'spam' WHERE name = 'Spam';`); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE conversation_statuses SET category = 'trashed' WHERE name = 'Trashed';`); err != nil {
		return err
	}
	return nil
}
