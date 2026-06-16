-- name: insert-reminder
-- $1 user_id, $2 conversation_id (lookup by UUID upstream), $3 remind_at, $4 note
INSERT INTO conversation_reminders (user_id, conversation_id, remind_at, note)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: get-reminder
SELECT * FROM conversation_reminders WHERE id = $1;

-- name: list-pending-for-conversation
-- Reminders for a specific conversation that the requesting user owns and
-- that haven't fired yet. Used to render the "Pending reminders on this
-- ticket" list under the ReminderButton popover.
SELECT
	r.*,
	c.uuid::text       AS conversation_uuid,
	c.reference_number::text AS conversation_reference_number,
	c.subject          AS conversation_subject
FROM conversation_reminders r
JOIN conversations c ON c.id = r.conversation_id
WHERE r.user_id = $1
  AND c.uuid = $2
  AND r.fired_at IS NULL
ORDER BY r.remind_at ASC;

-- name: list-pending-for-user
-- All pending reminders for the requesting user across every conversation.
-- Powers a future "My reminders" sidebar view.
SELECT
	r.*,
	c.uuid::text       AS conversation_uuid,
	c.reference_number::text AS conversation_reference_number,
	c.subject          AS conversation_subject
FROM conversation_reminders r
JOIN conversations c ON c.id = r.conversation_id
WHERE r.user_id = $1
  AND r.fired_at IS NULL
ORDER BY r.remind_at ASC;

-- name: delete-reminder
-- Only the owning user may delete their reminder.
DELETE FROM conversation_reminders
WHERE id = $1 AND user_id = $2;

-- name: select-due-reminders
-- Pull every reminder whose remind_at has passed and which hasn't fired yet,
-- joined with the data needed to build a notification (conversation UUID +
-- ref, user email + name). Limit keeps a single tick bounded if a backlog
-- builds up.
SELECT
	r.*,
	c.uuid::text       AS conversation_uuid,
	c.reference_number::text AS conversation_reference_number,
	c.subject          AS conversation_subject,
	u.email            AS user_email,
	u.first_name       AS user_first_name,
	u.last_name        AS user_last_name
FROM conversation_reminders r
JOIN conversations c ON c.id = r.conversation_id
JOIN users u ON u.id = r.user_id
WHERE r.remind_at <= NOW()
  AND r.fired_at IS NULL
ORDER BY r.remind_at ASC
LIMIT 200;

-- name: mark-reminder-fired
UPDATE conversation_reminders
SET fired_at = NOW()
WHERE id = $1;
