-- +migrate Up
-- Landing page tracking: know which landing page brought each user & subscription

-- Track where each user registered from
ALTER TABLE users ADD COLUMN IF NOT EXISTS utm_source TEXT;

-- Track which landing page led to each subscription
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS utm_source TEXT;

-- Index for analytics queries
CREATE INDEX IF NOT EXISTS idx_users_utm_source ON users(utm_source) WHERE utm_source IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_subscriptions_utm_source ON subscriptions(utm_source) WHERE utm_source IS NOT NULL;
