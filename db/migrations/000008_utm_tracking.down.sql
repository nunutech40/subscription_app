-- +migrate Down
DROP INDEX IF EXISTS idx_subscriptions_utm_source;
DROP INDEX IF EXISTS idx_users_utm_source;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS utm_source;
ALTER TABLE users DROP COLUMN IF EXISTS utm_source;
