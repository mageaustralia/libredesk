package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_2_22 adds the external-search "mode" + raw-passthrough guards (T3d2):
// generic GET/POST backends that return arbitrary JSON, not just
// Meilisearch's `hits` shape.
//
//   - ai.external_search_mode: "meilisearch" (default, unchanged typed
//     parse), "generic_get", or "generic_post" (raw JSON into the prompt).
//   - ai.external_search_max_chars: truncates each generic response before
//     it enters the prompt (token guard).
//   - ai.external_search_timeout_ms: per-request cap (was a fixed 10s).
//
// Global settings rows + matching per-inbox columns. Idempotent so existing
// installs upgrade cleanly; defaults preserve current behaviour (meilisearch
// mode), so nothing changes until an admin picks a generic mode.
func V2_2_22(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		INSERT INTO settings ("key", value) VALUES
		  ('ai.external_search_mode', '"meilisearch"'::jsonb),
		  ('ai.external_search_max_chars', '4000'::jsonb),
		  ('ai.external_search_timeout_ms', '1000'::jsonb)
		ON CONFLICT (key) DO NOTHING;
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		ALTER TABLE inbox_ai_settings
		  ADD COLUMN IF NOT EXISTS external_search_mode TEXT NOT NULL DEFAULT 'meilisearch',
		  ADD COLUMN IF NOT EXISTS external_search_max_chars INT NOT NULL DEFAULT 4000,
		  ADD COLUMN IF NOT EXISTS external_search_timeout_ms INT NOT NULL DEFAULT 1000;
	`); err != nil {
		return err
	}
	return nil
}
