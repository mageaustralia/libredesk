package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_2_24 migrates two-letter locale codes (en, de, fr, …) to BCP 47
// region codes (en-US, de-DE, fr-FR, …) across the two places they're
// stored:
//
//  1. The `app.lang` setting — affects new-admin default language.
//  2. The per-livechat-inbox `config.language` and
//     `config.fallback_language` JSONB fields — affects which
//     translation file the widget loads.
//
// Folds upstream's e0f362c1 (app.lang) + 1233f2fb (livechat config)
// into a single migration on our v2.2.x timeline. Upstream split these
// across v2.2.2 + v2.3.0; we're already at v2.2.23 (and our own v2.2.2
// is a no-op marker for EC14), so we land both as v2.2.24.
//
// Surgical UPDATEs — only rows whose current value matches the old
// two-letter form get rewritten, so this is idempotent and safe to
// re-run if needed.
func V2_2_24(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	langMap := map[string]string{
		"da": "da-DK",
		"de": "de-DE",
		"en": "en-US",
		"es": "es-ES",
		"fa": "fa-IR",
		"fr": "fr-FR",
		"it": "it-IT",
		"ja": "ja-JP",
		"mr": "mr-IN",
	}

	for localeCode, localeRegion := range langMap {
		if _, err := db.Exec(`
			UPDATE settings SET value = to_jsonb($1::text), updated_at = now()
			WHERE key = 'app.lang' AND value = to_jsonb($2::text);
		`, localeRegion, localeCode); err != nil {
			return err
		}

		if _, err := db.Exec(`
			UPDATE inboxes
			SET config = jsonb_set(config, '{language}', to_jsonb($1::text)), updated_at = now()
			WHERE channel = 'livechat' AND config->>'language' = $2;
		`, localeRegion, localeCode); err != nil {
			return err
		}

		if _, err := db.Exec(`
			UPDATE inboxes
			SET config = jsonb_set(config, '{fallback_language}', to_jsonb($1::text)), updated_at = now()
			WHERE channel = 'livechat' AND config->>'fallback_language' = $2;
		`, localeRegion, localeCode); err != nil {
			return err
		}
	}

	return nil
}
