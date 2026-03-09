-- name: RecordPageView :exec
INSERT INTO analytics_pageviews (
    page_url, referrer, ip_hash, user_agent, utm_source, utm_medium, utm_campaign
) VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetTotalVisitors :one
SELECT COUNT(DISTINCT ip_hash)::bigint FROM analytics_pageviews;

-- name: GetTotalPageViews :one
SELECT COUNT(*)::bigint FROM analytics_pageviews;

-- name: GetDailyVisitors :many
SELECT 
    DATE_TRUNC('day', created_at)::date AS date,
    COUNT(DISTINCT ip_hash)::bigint AS count
FROM analytics_pageviews
WHERE created_at >= NOW() - INTERVAL '30 days'
GROUP BY 1
ORDER BY 1 ASC;

-- name: GetPageViewsByUrl :many
SELECT 
    page_url,
    COUNT(*)::bigint AS views,
    COUNT(DISTINCT ip_hash)::bigint AS unique_visitors
FROM analytics_pageviews
WHERE created_at >= NOW() - INTERVAL '30 days'
GROUP BY page_url
ORDER BY views DESC;

-- name: GetRecentPageViews :many
SELECT * FROM analytics_pageviews 
ORDER BY created_at DESC 
LIMIT $1 OFFSET $2;
