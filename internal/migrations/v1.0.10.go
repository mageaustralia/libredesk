package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V1_0_10 adds the api_request activity log type (audit trail for API-key
// calls) and the conversations:delete permission, split out of
// conversations:update_status so a role can mark spam / trash / restore
// without being able to permanently delete. Roles that could delete before
// (anything holding update_status) keep that ability.
func V1_0_10(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`ALTER TYPE activity_log_type ADD VALUE IF NOT EXISTS 'api_request'`); err != nil {
		return err
	}
	_, err := db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'conversations:delete')
		WHERE 'conversations:update_status' = ANY(permissions)
		  AND NOT ('conversations:delete' = ANY(permissions))`)
	return err
}
