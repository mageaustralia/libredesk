-- name: search-conversations-by-reference-number
SELECT
    conversations.created_at,
    conversations.uuid,
    conversations.reference_number,
    conversations.subject,
    cs.name AS status
FROM conversations
LEFT JOIN conversation_statuses cs ON conversations.status_id = cs.id
WHERE reference_number::text = $1;

-- name: search-conversations-by-contact-email
SELECT
    conversations.created_at,
    conversations.uuid,
    conversations.reference_number,
    conversations.subject,
    cs.name AS status
FROM conversations
JOIN users ON conversations.contact_id = users.id
LEFT JOIN conversation_statuses cs ON conversations.status_id = cs.id
WHERE users.email = $1
ORDER BY conversations.created_at DESC
LIMIT 1000;

-- name: search-messages
SELECT
    c.created_at as "conversation_created_at",
    c.reference_number as "conversation_reference_number",
    c.uuid as "conversation_uuid",
    LEFT(m.text_content, 200) AS text_content,
    cs.name as "conversation_status"
FROM conversation_messages m
    JOIN conversations c ON m.conversation_id = c.id
    LEFT JOIN conversation_statuses cs ON c.status_id = cs.id
-- T3y H4: skip messages flagged as containing card data so a raw PAN is
-- never returned (or made findable) via search before it's redacted.
WHERE m.type != 'activity' and m.has_pci_data = false and m.text_content ILIKE '%' || $1 || '%'
LIMIT 30;

-- name: search-contacts
SELECT
    id,
    created_at,
    first_name,
    last_name,
    email,
    external_user_id
FROM users
WHERE type = 'contact'
AND deleted_at IS NULL
AND (
    email ILIKE '%' || $1 || '%'
    OR first_name ILIKE '%' || $1 || '%'
    OR last_name ILIKE '%' || $1 || '%'
)
LIMIT 15;

-- name: search-unified
-- FS8: Single-pass search across reference number, subject, contact email,
-- and message text content. Returns one row per conversation with the most
-- relevant snippet. Uses DISTINCT ON to dedupe (a conversation can match
-- multiple predicates) and a window count for total pagination metadata.
SELECT *, COUNT(*) OVER() AS total FROM (
    SELECT DISTINCT ON (c.id)
        c.created_at,
        c.uuid,
        c.reference_number,
        c.subject,
        cs.name AS status,
        COALESCE(
            -- Prefer the latest message containing the search term (highlights
            -- the actual hit when the user searched message body).
            -- T3y H4: never surface a PCI-flagged message as the snippet
            -- (would leak the raw PAN into search results).
            (SELECT m.text_content FROM conversation_messages m
             WHERE m.conversation_id = c.id AND m.type != 'activity'
             AND m.has_pci_data = false
             AND m.text_content ILIKE '%' || $1 || '%'
             ORDER BY m.created_at DESC LIMIT 1),
            -- Fall back to the first incoming message (gives subject/refnum
            -- matches a body snippet to display).
            (SELECT m.text_content FROM conversation_messages m
             WHERE m.conversation_id = c.id AND m.type = 'incoming' AND m.sender_type = 'contact'
             AND m.has_pci_data = false
             ORDER BY m.id ASC LIMIT 1),
            ''
        ) AS snippet
    FROM conversations c
    LEFT JOIN users u ON c.contact_id = u.id
    LEFT JOIN conversation_statuses cs ON c.status_id = cs.id
    WHERE c.reference_number::text = $1
       OR c.subject ILIKE '%' || $1 || '%'
       OR u.email = $1
       OR c.id IN (
           SELECT m.conversation_id FROM conversation_messages m
           WHERE m.type != 'activity' AND m.has_pci_data = false AND m.text_content ILIKE '%' || $1 || '%'
       )
    ORDER BY c.id
) sub
ORDER BY
    CASE WHEN reference_number::text = $1 THEN 0 ELSE 1 END,
    created_at DESC
LIMIT $2 OFFSET $3;


-- name: search-unified-contacts
-- Trigram match on full name (typo-tolerant via % operator) plus substring
-- match on name/email. Only contacts with at least one conversation surface —
-- pure directory lookups belong in the Contacts area, not ticket search.
-- $4 is the fuzzy term: the query with any @domain stripped, so shared
-- "@gmail.com" trigrams can't score against contacts whose stored name is an
-- email address. Exact and substring matching keep the full query ($1).
SELECT *, COUNT(*) OVER() AS total FROM (
    SELECT
        u.id,
        u.created_at,
        u.first_name,
        u.last_name,
        COALESCE(u.email::text, '') AS email,
        GREATEST(
            similarity(u.first_name || ' ' || u.last_name, $4),
            similarity(COALESCE(u.email::text, ''), $1)
        ) AS sim
    FROM users u
    WHERE u.type = 'contact'
      AND u.deleted_at IS NULL
      AND EXISTS (SELECT 1 FROM conversations c WHERE c.contact_id = u.id)
      AND (
        (u.first_name || ' ' || u.last_name) % $4
        OR (u.first_name || ' ' || u.last_name) ILIKE '%' || $1 || '%'
        OR u.email ILIKE '%' || $1 || '%'
      )
) sub
ORDER BY sim DESC
LIMIT $2 OFFSET $3;

-- name: search-unified-conversations
-- Ranked tiers: exact ref (0) > contact name (1) > contact email (2) >
-- subject (3), then most recent activity. status_id/inbox_id/assigned 0 = no
-- filter (assigned -1 = unassigned); dates NULL = no filter. $9 is the fuzzy
-- term (query minus any @domain) used for trigram name matching only — see
-- search-unified-contacts.
SELECT *, COUNT(*) OVER() AS total FROM (
    SELECT
        c.created_at,
        c.last_message_at,
        c.last_message_sender,
        c.uuid,
        c.reference_number,
        COALESCE(c.subject, '') AS subject,
        TRIM(u.first_name || ' ' || u.last_name) AS contact_name,
        COALESCE(u.email::text, '') AS contact_email,
        CASE
            WHEN c.reference_number::text = $1 THEN 0
            WHEN (u.first_name || ' ' || u.last_name) ILIKE '%' || $1 || '%'
                 OR (u.first_name || ' ' || u.last_name) % $9 THEN 1
            WHEN u.email ILIKE '%' || $1 || '%' THEN 2
            ELSE 3
        END AS match_rank
    FROM conversations c
    JOIN users u ON c.contact_id = u.id
    WHERE (
        c.reference_number::text = $1
        OR (u.first_name || ' ' || u.last_name) % $9
        OR (u.first_name || ' ' || u.last_name) ILIKE '%' || $1 || '%'
        OR u.email ILIKE '%' || $1 || '%'
        OR c.subject ILIKE '%' || $1 || '%'
    )
    AND ($2::int = 0 OR c.status_id = $2)
    AND ($3::int = 0 OR c.inbox_id = $3)
    AND ($4::timestamptz IS NULL OR COALESCE(c.last_message_at, c.created_at) >= $4)
    AND ($5::timestamptz IS NULL OR COALESCE(c.last_message_at, c.created_at) <= $5)
    AND ($6::int = 0 OR ($6 = -1 AND c.assigned_user_id IS NULL) OR c.assigned_user_id = $6)
) sub
ORDER BY match_rank, COALESCE(last_message_at, created_at) DESC
LIMIT $7 OFFSET $8;

-- name: search-unified-messages
-- FTS over the generated tsvector; one row per conversation (latest matching
-- message). [[[ / ]]] delimit highlights - frontend splits on them (never
-- rendered as HTML). ts_headline is deferred to the outer select so it only
-- runs on the final paginated rows, not every match (it re-parses each full
-- body and is ~100x the cost of the ts_rank/index scan for common words).
-- T3y: never surface a PCI-flagged message (would leak the raw PAN).
SELECT
    ranked.created_at,
    ranked.uuid,
    ranked.reference_number,
    ranked.subject,
    ts_headline('english', ranked.text_content, websearch_to_tsquery('english', $1),
        'StartSel=[[[, StopSel=]]], MaxWords=35, MinWords=15') AS snippet,
    ranked.match_rank,
    ranked.total
FROM (
    SELECT *, COUNT(*) OVER() AS total FROM (
        SELECT DISTINCT ON (m.conversation_id)
            m.created_at,
            m.text_content,
            c.uuid,
            c.reference_number,
            COALESCE(c.subject, '') AS subject,
            ts_rank(m.text_content_tsv, websearch_to_tsquery('english', $1)) AS match_rank
        FROM conversation_messages m
        JOIN conversations c ON m.conversation_id = c.id
        WHERE m.type != 'activity'
          AND m.has_pci_data = false
          AND m.text_content_tsv @@ websearch_to_tsquery('english', $1)
          AND ($2::int = 0 OR c.status_id = $2)
          AND ($3::int = 0 OR c.inbox_id = $3)
          AND ($4::timestamptz IS NULL OR m.created_at >= $4)
          AND ($5::timestamptz IS NULL OR m.created_at <= $5)
          AND ($6::int = 0 OR ($6 = -1 AND c.assigned_user_id IS NULL) OR c.assigned_user_id = $6)
        ORDER BY m.conversation_id, m.created_at DESC
    ) dedup
    ORDER BY match_rank DESC, created_at DESC
    LIMIT $7 OFFSET $8
) ranked
ORDER BY ranked.match_rank DESC, ranked.created_at DESC;
