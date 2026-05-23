package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_2_21 seeds the pci.* notification settings (T3y PCI redaction). These
// were added to the settings page + GetByPrefix("pci.") path but never given
// schema defaults, so existing installs had zero pci.* rows. GetByPrefix then
// returned NULL (JSON_OBJECT_AGG over no rows) and the settings page 500'd on
// decrypt. They're also needed as real rows because the settings UPDATE only
// touches existing keys — without them, saving PCI settings silently no-ops.
//
// ON CONFLICT DO NOTHING so re-runs and installs that already have the rows
// (e.g. set via the UI) are untouched. Defaults match the model: agent 0 =
// notifications disabled, method "both".
func V2_2_21(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		INSERT INTO settings ("key", value) VALUES
		  ('pci.notify_agent_id', '0'::jsonb),
		  ('pci.notify_method', '"both"'::jsonb)
		ON CONFLICT (key) DO NOTHING;
	`); err != nil {
		return err
	}
	return nil
}
