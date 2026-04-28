package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_2_2 is a no-op for schema. EC14 (per-inbox From switcher / aliases) stores
// aliases inside the existing inboxes.config JSONB column rather than adding a
// new column, so there is nothing to migrate at the SQL layer. The migration
// entry is registered to keep upgrade ordering stable and to leave a hook for
// any future backfill (e.g. seeding aliases from From parsing) without bumping
// the version.
func V2_2_2(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	return nil
}
