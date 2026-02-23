-- ─── Admin Audit Log ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS admin_audit_logs (
  id         BIGSERIAL PRIMARY KEY,
  admin_email TEXT NOT NULL,
  action     TEXT NOT NULL,           -- 'update_config', 'toggle_product', 'lock_user', etc.
  resource   TEXT NOT NULL,           -- 'system_config', 'product', 'user', 'pricing_plan', etc.
  resource_id TEXT,                   -- the specific ID of the resource modified
  detail     TEXT,                    -- JSON or human-readable detail of the change
  ip         INET,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_admin_audit_created ON admin_audit_logs(created_at DESC);
CREATE INDEX idx_admin_audit_action ON admin_audit_logs(action);
