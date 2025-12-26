-- name: create-webhook
INSERT INTO webhooks (name, url, status, events, auth_type, auth_basic_user, auth_basic_pass, auth_hmac_secret, max_retries, retry_interval, timeout)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id, uuid;

-- name: get-webhooks
SELECT id, uuid, name, url, status, events, auth_type, auth_basic_user, auth_basic_pass, auth_hmac_secret, max_retries, retry_interval, timeout, created_at, updated_at
FROM webhooks
WHERE ($1 = 0 OR id = $1)
ORDER BY created_at DESC;

-- name: get-webhooks-by-event
SELECT id, uuid, name, url, status, events, auth_type, auth_basic_user, auth_basic_pass, auth_hmac_secret, max_retries, retry_interval, timeout, created_at, updated_at
FROM webhooks
WHERE status = 'enabled' AND $1 = ANY(events)
ORDER BY created_at;

-- name: update-webhook
UPDATE webhooks SET
    name = $2,
    url = $3,
    status = $4,
    events = $5,
    auth_type = $6,
    auth_basic_user = $7,
    auth_basic_pass = (CASE WHEN $8 = '' THEN auth_basic_pass ELSE $8 END),
    auth_hmac_secret = (CASE WHEN $9 = '' THEN auth_hmac_secret ELSE $9 END),
    max_retries = $10,
    retry_interval = $11,
    timeout = $12,
    updated_at = NOW()
WHERE id = $1;

-- name: delete-webhooks
DELETE FROM webhooks WHERE id = ANY($1);

-- name: create-webhook-log
INSERT INTO webhook_logs (webhook_id, event, url, payload, status, next_retry_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: update-webhook-log
UPDATE webhook_logs SET
    status = $2,
    response_code = $3,
    response_body = $4,
    error = $5,
    attempts = $6,
    next_retry_at = $7,
    updated_at = NOW()
WHERE id = $1;

-- name: get-pending-webhook-logs
SELECT l.id, l.webhook_id, l.event, l.url, l.payload, l.status, l.response_code, l.response_body, l.error, l.attempts, l.next_retry_at, l.created_at, l.updated_at,
       w.max_retries, w.timeout, w.auth_type, w.auth_basic_user, w.auth_basic_pass, w.auth_hmac_secret
FROM webhook_logs l
JOIN webhooks w ON w.id = l.webhook_id
WHERE l.status = 'pending' AND (l.next_retry_at IS NULL OR l.next_retry_at <= NOW())
ORDER BY l.created_at
LIMIT $1;

-- name: query-webhook-logs
SELECT COUNT(*) OVER () AS total,
       l.id, l.webhook_id, l.event, l.url, l.payload, l.status, l.response_code, l.response_body, l.error, l.attempts, l.next_retry_at, l.created_at, l.updated_at,
       w.name AS webhook_name
FROM webhook_logs l
LEFT JOIN webhooks w ON w.id = l.webhook_id
WHERE ($1 = 0 OR l.webhook_id = $1)
    AND ($2 = '' OR l.status = $2::webhook_log_status)
    AND ($3 = '' OR l.event = $3)
ORDER BY %order% OFFSET $4 LIMIT (CASE WHEN $5 < 1 THEN NULL ELSE $5 END);

-- name: delete-webhook-logs
DELETE FROM webhook_logs
WHERE ($1 = TRUE)
   OR (id = ANY($2))
   OR ($3 > 0 AND webhook_id = $3)
   OR ($4 != '' AND status = $4::webhook_log_status);
