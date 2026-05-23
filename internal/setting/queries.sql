-- name: get-all
SELECT JSON_OBJECT_AGG(key, value) AS settings FROM (SELECT * FROM settings ORDER BY key) t;

-- name: update
UPDATE settings AS s
SET value = c.value,
    updated_at = now()
FROM (SELECT * FROM jsonb_each($1)) AS c(key, value)
WHERE s.key = c.key;

-- name: get-by-prefix
-- COALESCE to '{}' because JSON_OBJECT_AGG returns NULL (not an empty object)
-- when no rows match the prefix. A bare NULL flows into decryptSettings as an
-- empty body and fails with "unexpected end of JSON input" (500). Returning
-- '{}' lets callers get an empty settings object cleanly.
SELECT COALESCE(JSON_OBJECT_AGG(key, value), '{}'::json) AS settings
FROM settings
WHERE key LIKE $1 || '%';

-- name: get
SELECT value FROM settings WHERE key = $1;

-- name: get-inbox-ai-settings
SELECT id, created_at, updated_at, inbox_id, system_prompt, max_context_chunks,
    similarity_threshold, external_search_enabled, external_search_url,
    external_search_max_results, external_search_endpoints, external_search_headers,
    external_search_mode, external_search_max_chars, external_search_timeout_ms,
    knowledge_source_ids
FROM inbox_ai_settings WHERE inbox_id = $1;

-- name: upsert-inbox-ai-settings
INSERT INTO inbox_ai_settings (inbox_id, system_prompt, max_context_chunks, similarity_threshold,
    external_search_enabled, external_search_url, external_search_max_results,
    external_search_endpoints, external_search_headers, knowledge_source_ids,
    external_search_mode, external_search_max_chars, external_search_timeout_ms, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW())
ON CONFLICT (inbox_id) DO UPDATE SET
    system_prompt = EXCLUDED.system_prompt,
    max_context_chunks = EXCLUDED.max_context_chunks,
    similarity_threshold = EXCLUDED.similarity_threshold,
    external_search_enabled = EXCLUDED.external_search_enabled,
    external_search_url = EXCLUDED.external_search_url,
    external_search_max_results = EXCLUDED.external_search_max_results,
    external_search_endpoints = EXCLUDED.external_search_endpoints,
    external_search_headers = EXCLUDED.external_search_headers,
    knowledge_source_ids = EXCLUDED.knowledge_source_ids,
    external_search_mode = EXCLUDED.external_search_mode,
    external_search_max_chars = EXCLUDED.external_search_max_chars,
    external_search_timeout_ms = EXCLUDED.external_search_timeout_ms,
    updated_at = NOW()
RETURNING id, created_at, updated_at, inbox_id, system_prompt, max_context_chunks,
    similarity_threshold, external_search_enabled, external_search_url,
    external_search_max_results, external_search_endpoints, external_search_headers,
    external_search_mode, external_search_max_chars, external_search_timeout_ms,
    knowledge_source_ids;

-- name: delete-inbox-ai-settings
DELETE FROM inbox_ai_settings WHERE inbox_id = $1;