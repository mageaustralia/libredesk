package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_2_3 seeds the "New reply on conversation" email template, used by the
// new notifyParticipants fan-out (assignee + watchers on customer reply).
//
// The body and subject use {{ if .Recipient.IsAssignee }}…{{ else }}…{{ end }}
// so a follower sees Freshdesk-style "New activity / you are watching"
// language while the ticket owner keeps the existing "New reply" wording.
//
// Idempotent: ON CONFLICT (name) DO NOTHING means an admin who has already
// customised this template (or migrated an export from v1.0.3) keeps their
// version untouched.
func V2_2_3(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	const subject = `{{ if .Recipient.IsAssignee }}New reply on #{{ .Conversation.ReferenceNumber }} - {{ .Conversation.Subject }}{{ else }}New activity [#{{ .Conversation.ReferenceNumber }}] {{ .Conversation.Subject }}{{ end }}`
	const body = `<p>Hi {{ .Recipient.FirstName }},</p>
{{ if .Recipient.IsAssignee }}
<p><strong>{{ .Author.FullName }}</strong> replied to ticket <a href="{{ RootURL }}/inboxes/assigned/conversation/{{ .Conversation.UUID }}"><strong>#{{ .Conversation.ReferenceNumber }}</strong></a>.</p>
{{ else }}
<p>The customer has responded to a ticket <strong>you are watching</strong>: <a href="{{ RootURL }}/inboxes/assigned/conversation/{{ .Conversation.UUID }}"><strong>#{{ .Conversation.ReferenceNumber }}</strong></a>.</p>
{{ end }}
<p><strong>Subject:</strong> {{ .Conversation.Subject }}</p>
{{ if .Message.Content }}
<div style="border-top: 1px solid #e0e0e0; margin-top: 16px; padding-top: 16px;">
{{ .Message.Content }}
</div>
{{ end }}
<p style="margin-top: 16px;">
    <a href="{{ RootURL }}/inboxes/assigned/conversation/{{ .Conversation.UUID }}">View Conversation</a>
</p>`

	// `templates.name` has no unique constraint, so we can't use ON CONFLICT.
	// Insert only if a row with the same name doesn't already exist — matches
	// the no-op outcome admins expect when re-running migrations or upgrading
	// from a deployment that already created the template by hand.
	_, err := db.Exec(`
		INSERT INTO templates ("type", name, subject, body, is_default, is_builtin)
		SELECT 'email_notification'::template_type, $1, $2, $3, false, true
		WHERE NOT EXISTS (SELECT 1 FROM templates WHERE name = $1)
	`, "New reply on conversation", subject, body)
	return err
}
