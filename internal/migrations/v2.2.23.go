package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_2_23 fixes the watcher branch of the "New reply on conversation" email
// template. The original v2.2.3 seed hardcoded "The customer has responded"
// regardless of who actually replied, so an agent posting a reply still
// triggered a "the customer has responded" notification to other watchers.
// Use {{ .Author.FullName }} (already passed into the template context) so
// the attribution is accurate whether an agent or a contact wrote the reply.
//
// Surgical: only updates rows whose body still contains the buggy snippet,
// so any admin who has manually customised the template keeps their version.
func V2_2_23(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	const oldSnippet = `<p>The customer has responded to a ticket <strong>you are watching</strong>:`
	const newSnippet = `<p><strong>{{ .Author.FullName }}</strong> replied to a ticket <strong>you are watching</strong>:`
	_, err := db.Exec(`
		UPDATE templates
		SET body = replace(body, $1, $2)
		WHERE name = 'New reply on conversation'
		  AND body LIKE '%' || $1 || '%'
	`, oldSnippet, newSnippet)
	return err
}
