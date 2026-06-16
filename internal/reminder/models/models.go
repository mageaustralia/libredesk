package models

import (
	"time"

	"github.com/volatiletech/null/v9"
)

// Reminder is a personal, per-agent, per-conversation follow-up nudge. When
// remind_at passes, the reminder worker dispatches a notification to user_id
// and marks fired_at. Reminders never change the conversation's state — they
// are purely a notification to the agent who set them.
type Reminder struct {
	ID             int       `db:"id" json:"id"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UserID         int       `db:"user_id" json:"user_id"`
	ConversationID int       `db:"conversation_id" json:"conversation_id"`
	RemindAt       time.Time `db:"remind_at" json:"remind_at"`
	FiredAt        null.Time `db:"fired_at" json:"fired_at"`
	Note           string    `db:"note" json:"note"`
}

// DueReminder is a row hydrated for the firing worker — joined with
// conversation and user fields needed to build the notification.
type DueReminder struct {
	Reminder
	ConversationUUID            string      `db:"conversation_uuid"`
	ConversationReferenceNumber string      `db:"conversation_reference_number"`
	ConversationSubject         null.String `db:"conversation_subject"`
	UserEmail                   null.String `db:"user_email"`
	UserFirstName               string      `db:"user_first_name"`
	UserLastName                string      `db:"user_last_name"`
}

// PendingReminder is the shape returned to the frontend when listing
// reminders for a conversation — includes the conversation ref so the
// "My reminders" view (future) can render labels without extra joins.
type PendingReminder struct {
	Reminder
	ConversationUUID            string      `db:"conversation_uuid" json:"conversation_uuid"`
	ConversationReferenceNumber string      `db:"conversation_reference_number" json:"conversation_reference_number"`
	ConversationSubject         null.String `db:"conversation_subject" json:"conversation_subject"`
}
