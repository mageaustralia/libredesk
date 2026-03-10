-- name: get-status
select id, 
    created_at,
    name,
    sort_order,
    show_on_send
from conversation_statuses
where id = $1;

-- name: get-all-statuses
select id, 
    created_at,
    name,
    sort_order,
    show_on_send
from conversation_statuses
ORDER BY sort_order, id;

-- name: insert-status
INSERT into conversation_statuses(name, sort_order) values ($1, (SELECT COALESCE(MAX(sort_order), 0) + 1 FROM conversation_statuses)) RETURNING *;

-- name: delete-status
DELETE from conversation_statuses where id = $1;

-- name: update-status
UPDATE conversation_statuses set name = $2 where id = $1 RETURNING *;

-- name: toggle-show-on-send
UPDATE conversation_statuses SET show_on_send = $2 WHERE id = $1;

-- name: reorder-statuses
UPDATE conversation_statuses SET sort_order = $2 WHERE id = $1;