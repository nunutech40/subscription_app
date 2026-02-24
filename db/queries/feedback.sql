-- name: CreateFeedback :one
INSERT INTO feedback (user_email, user_role, category, rating, message, page_url, user_agent, ip)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListFeedback :many
SELECT * FROM feedback ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: ListFeedbackByCategory :many
SELECT * FROM feedback WHERE category = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: ListUnreadFeedback :many
SELECT * FROM feedback WHERE is_read = FALSE ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountUnreadFeedback :one
SELECT COUNT(*) FROM feedback WHERE is_read = FALSE;

-- name: MarkFeedbackRead :exec
UPDATE feedback SET is_read = TRUE WHERE id = $1;

-- name: MarkAllFeedbackRead :exec
UPDATE feedback SET is_read = TRUE WHERE is_read = FALSE;

-- name: GetFeedbackStats :one
SELECT
  COUNT(*) AS total,
  COUNT(*) FILTER (WHERE is_read = FALSE) AS unread,
  COUNT(*) FILTER (WHERE category = 'saran') AS saran_count,
  COUNT(*) FILTER (WHERE category = 'bug') AS bug_count,
  COUNT(*) FILTER (WHERE category = 'pertanyaan') AS tanya_count,
  ROUND(AVG(rating)::numeric, 1) AS avg_rating
FROM feedback;
