// Package reminder manages per-agent, per-conversation follow-up reminders.
// A reminder is a personal nudge: when remind_at passes, the worker dispatches
// an in-app + WS + email notification to the agent who set it. The
// conversation's state is never touched.
package reminder

import (
	"context"
	"embed"
	"fmt"
	"time"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	notifier "github.com/abhinavxd/libredesk/internal/notification"
	nmodels "github.com/abhinavxd/libredesk/internal/notification/models"
	"github.com/abhinavxd/libredesk/internal/reminder/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Dispatcher is the subset of notifier.Dispatcher this package needs. Kept as
// an interface so tests can substitute a no-op.
type Dispatcher interface {
	Send(n notifier.Notification)
}

type Manager struct {
	q          queries
	db         *sqlx.DB
	dispatcher Dispatcher
	lo         *logf.Logger
	i18n       *i18n.I18n
}

type Opts struct {
	DB         *sqlx.DB
	Dispatcher Dispatcher
	Lo         *logf.Logger
	I18n       *i18n.I18n
}

type queries struct {
	InsertReminder            *sqlx.Stmt `query:"insert-reminder"`
	GetReminder               *sqlx.Stmt `query:"get-reminder"`
	ListPendingForConversation *sqlx.Stmt `query:"list-pending-for-conversation"`
	ListPendingForUser        *sqlx.Stmt `query:"list-pending-for-user"`
	DeleteReminder            *sqlx.Stmt `query:"delete-reminder"`
	SelectDueReminders        *sqlx.Stmt `query:"select-due-reminders"`
	MarkReminderFired         *sqlx.Stmt `query:"mark-reminder-fired"`
}

func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:          q,
		db:         opts.DB,
		dispatcher: opts.Dispatcher,
		lo:         opts.Lo,
		i18n:       opts.I18n,
	}, nil
}

// Create inserts a new reminder for the given user against the given
// conversation. RemindAt must be in the future; the worker tolerates past
// times (fires immediately) but the API layer should reject them so the
// agent doesn't accidentally pick a stale preset.
func (m *Manager) Create(userID, conversationID int, remindAt time.Time, note string) (models.Reminder, error) {
	var r models.Reminder
	if err := m.q.InsertReminder.Get(&r, userID, conversationID, remindAt, note); err != nil {
		m.lo.Error("error inserting reminder", "user_id", userID, "conversation_id", conversationID, "error", err)
		return r, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "reminder"), nil)
	}
	return r, nil
}

// ListForConversation returns the requesting user's pending reminders for a
// specific conversation (by UUID), oldest due first.
func (m *Manager) ListForConversation(userID int, conversationUUID string) ([]models.PendingReminder, error) {
	out := make([]models.PendingReminder, 0)
	if err := m.q.ListPendingForConversation.Select(&out, userID, conversationUUID); err != nil {
		m.lo.Error("error listing reminders for conversation", "user_id", userID, "conversation_uuid", conversationUUID, "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "reminders"), nil)
	}
	return out, nil
}

// ListForUser returns every pending reminder owned by the user across all
// conversations, oldest due first.
func (m *Manager) ListForUser(userID int) ([]models.PendingReminder, error) {
	out := make([]models.PendingReminder, 0)
	if err := m.q.ListPendingForUser.Select(&out, userID); err != nil {
		m.lo.Error("error listing reminders for user", "user_id", userID, "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "reminders"), nil)
	}
	return out, nil
}

// Delete removes a reminder. Only the owning user may delete it — the query
// filters on user_id so a wrong caller silently no-ops.
func (m *Manager) Delete(userID, reminderID int) error {
	if _, err := m.q.DeleteReminder.Exec(reminderID, userID); err != nil {
		m.lo.Error("error deleting reminder", "user_id", userID, "reminder_id", reminderID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "reminder"), nil)
	}
	return nil
}

// ResolveConversationIDByUUID maps a conversation UUID to its internal
// integer ID. Used by the HTTP handler before calling Create. Lives here so
// the reminder package owns its own DB access rather than depending on the
// (already huge) conversation manager.
func (m *Manager) ResolveConversationIDByUUID(uuid string) (int, error) {
	var id int
	if err := m.db.Get(&id, `SELECT id FROM conversations WHERE uuid = $1`, uuid); err != nil {
		return 0, err
	}
	return id, nil
}

// RunFirer is the periodic worker. Every `interval`, it pulls due reminders,
// dispatches one notification per row, and marks fired_at. Run as a goroutine
// from cmd/main.go.
func (m *Manager) RunFirer(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.fireDue(ctx)
		}
	}
}

func (m *Manager) fireDue(ctx context.Context) {
	due := make([]models.DueReminder, 0)
	if err := m.q.SelectDueReminders.SelectContext(ctx, &due); err != nil {
		m.lo.Error("error selecting due reminders", "error", err)
		return
	}
	for _, r := range due {
		m.dispatchOne(r)
		if _, err := m.q.MarkReminderFired.ExecContext(ctx, r.ID); err != nil {
			m.lo.Error("error marking reminder fired", "reminder_id", r.ID, "error", err)
		}
	}
	if len(due) > 0 {
		m.lo.Info("fired reminders", "count", len(due))
	}
}

func (m *Manager) dispatchOne(r models.DueReminder) {
	title := fmt.Sprintf("Reminder: #%s %s", r.ConversationReferenceNumber, firstLine(r.ConversationSubject.String))
	body := r.Note
	if body == "" {
		body = "(no note)"
	}
	n := notifier.Notification{
		Type:             nmodels.NotificationTypeAssignment, // re-use existing in-app type; UI treats by title
		RecipientIDs:     []int{r.UserID},
		Title:            title,
		Body:             null.StringFrom(body),
		ConversationUUID: r.ConversationUUID,
	}
	if r.UserEmail.Valid && r.UserEmail.String != "" {
		n.Email = &notifier.EmailNotification{
			Recipients: []string{r.UserEmail.String},
			Subject:    title,
			Content:    fmt.Sprintf("<p>Hi %s,</p><p>You set a reminder on this ticket:</p><p><strong>%s</strong></p><p>Reminder note: %s</p>", htmlEscape(r.UserFirstName), htmlEscape(r.ConversationSubject.String), htmlEscape(body)),
		}
	}
	m.dispatcher.Send(n)
}

func firstLine(s string) string {
	for i, c := range s {
		if c == '\n' || c == '\r' {
			return s[:i]
		}
	}
	return s
}

// htmlEscape is a minimal HTML escape for the four characters that matter
// inside a paragraph. Avoids pulling html/template just for one email.
func htmlEscape(s string) string {
	out := make([]rune, 0, len(s))
	for _, c := range s {
		switch c {
		case '&':
			out = append(out, []rune("&amp;")...)
		case '<':
			out = append(out, []rune("&lt;")...)
		case '>':
			out = append(out, []rune("&gt;")...)
		case '"':
			out = append(out, []rune("&quot;")...)
		default:
			out = append(out, c)
		}
	}
	return string(out)
}
