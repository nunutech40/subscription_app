-- =============================================
-- SAINS API — Initial Schema
-- Migration: 000001_init_schema
-- Date: 2026-02-22
-- Ref: BACKEND_PLAN.md v1.1, Section 6 + 16.2
-- =============================================

-- ─── PRODUCTS ─────────────────────────────────────────────────────────
CREATE TABLE products (
  id          TEXT PRIMARY KEY,          -- 'atomic', 'energi', 'biologi'
  name        TEXT NOT NULL,             -- 'Atomic — Interactive 3D Periodic Table'
  description TEXT,
  is_active   BOOLEAN DEFAULT TRUE,
  created_at  TIMESTAMPTZ DEFAULT now()
);

-- ─── PRICING PLANS ────────────────────────────────────────────────────
CREATE TABLE pricing_plans (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  product_id   TEXT REFERENCES products(id),
  segment      TEXT NOT NULL,            -- 'global' | 'student' | 'parent'
  duration     TEXT NOT NULL,            -- 'monthly' | '3month' | '6month' | 'yearly'
  duration_days INT NOT NULL,            -- 30 | 90 | 180 | 365
  price_idr    INTEGER NOT NULL,         -- harga dalam rupiah (atau cent USD untuk global)
  label        TEXT,                     -- "Tahunan — Hemat 40%" (untuk UI)
  is_active    BOOLEAN DEFAULT TRUE,
  created_at   TIMESTAMPTZ DEFAULT now()
);

-- ─── USERS ─────────────────────────────────────────────────────────────
CREATE TABLE users (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email         TEXT UNIQUE NOT NULL,
  name          TEXT,
  password_hash TEXT NOT NULL,           -- bcrypt, saltRounds=12
  role          TEXT DEFAULT 'subscriber', -- 'subscriber' | 'admin'
  is_active     BOOLEAN DEFAULT FALSE,
  anomaly_score INTEGER DEFAULT 0,
  created_at    TIMESTAMPTZ DEFAULT now()
);

-- ─── GUEST CODES ───────────────────────────────────────────────────────
-- Admin generate 1 code → bisa dishare ke banyak orang
CREATE TABLE guest_codes (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  code                  TEXT UNIQUE NOT NULL,    -- short code, e.g. "ATOM-A7X2"
  product_id            TEXT REFERENCES products(id),
  label                 TEXT,                    -- label internal admin ("Promo Februari")
  max_logins_per_email  INTEGER DEFAULT 2,       -- max login per email per code
  expires_at            TIMESTAMPTZ NOT NULL,    -- adjustable: 1d / 2d / 7d
  is_active             BOOLEAN DEFAULT TRUE,
  generated_by          UUID REFERENCES users(id),  -- admin yang generate
  created_at            TIMESTAMPTZ DEFAULT now()
);

-- ─── GUEST LOGINS ──────────────────────────────────────────────────────
-- Track per-email usage per guest code
CREATE TABLE guest_logins (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  guest_code_id   UUID REFERENCES guest_codes(id) ON DELETE CASCADE,
  email           TEXT NOT NULL,
  login_count     INTEGER DEFAULT 1,
  last_login_at   TIMESTAMPTZ DEFAULT now(),
  created_at      TIMESTAMPTZ DEFAULT now(),
  UNIQUE(guest_code_id, email)   -- 1 record per email per code
);

-- ─── SUBSCRIPTIONS ─────────────────────────────────────────────────────
CREATE TABLE subscriptions (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id             UUID REFERENCES users(id) ON DELETE CASCADE,
  product_id          TEXT REFERENCES products(id),
  plan_id             UUID REFERENCES pricing_plans(id),
  segment             TEXT NOT NULL,     -- 'global' | 'student' | 'parent'
  xendit_invoice_id   TEXT UNIQUE,
  xendit_payment_id   TEXT,
  amount_paid_idr     INTEGER,
  status              TEXT DEFAULT 'pending',  -- 'pending' | 'active' | 'expired' | 'cancelled'
  paid_at             TIMESTAMPTZ,
  starts_at           TIMESTAMPTZ,
  expires_at          TIMESTAMPTZ NOT NULL,
  created_at          TIMESTAMPTZ DEFAULT now()
);

-- ─── SESSIONS ──────────────────────────────────────────────────────────
CREATE TABLE sessions (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id              UUID REFERENCES users(id) ON DELETE CASCADE,  -- NULL jika guest
  guest_code_id        UUID REFERENCES guest_codes(id),              -- NULL jika subscriber
  guest_email          TEXT,                                          -- email guest (tracking)
  refresh_token_hash   TEXT NOT NULL,
  device_fingerprint   TEXT,
  ip_at_login          INET,
  user_agent           TEXT,
  country_code         TEXT,             -- dari IP geolocation
  is_active            BOOLEAN DEFAULT TRUE,
  revoked_at           TIMESTAMPTZ,
  revoke_reason        TEXT,             -- 'logout' | 'new_login' | 'admin' | 'expired'
  expires_at           TIMESTAMPTZ NOT NULL,
  created_at           TIMESTAMPTZ DEFAULT now()
);

-- ─── ANOMALY LOGS ──────────────────────────────────────────────────────
CREATE TABLE anomaly_logs (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       UUID REFERENCES users(id) ON DELETE CASCADE,
  event         TEXT NOT NULL,           -- 'session_displaced' | 'country_change' | ...
  score_delta   INTEGER NOT NULL,        -- poin yang ditambahkan
  detail        JSONB,                   -- data tambahan (old_ip, new_ip, dll)
  created_at    TIMESTAMPTZ DEFAULT now()
);

-- ─── ACCESS LOGS ───────────────────────────────────────────────────────
CREATE TABLE access_logs (
  id          BIGSERIAL PRIMARY KEY,
  user_id     UUID,
  session_id  UUID,
  ip          INET,
  endpoint    TEXT,
  method      TEXT,
  status_code INTEGER,
  created_at  TIMESTAMPTZ DEFAULT now()
);

-- ─── SYSTEM CONFIG ────────────────────────────────────────────────────
-- Key-value store untuk konfigurasi dinamis (quota, limits, dll)
CREATE TABLE system_config (
  key         TEXT PRIMARY KEY,
  value       TEXT NOT NULL,
  description TEXT,
  updated_at  TIMESTAMPTZ DEFAULT now(),
  updated_by  UUID REFERENCES users(id)
);

-- ─── INDEXES ───────────────────────────────────────────────────────────
CREATE INDEX idx_subscriptions_user_product ON subscriptions(user_id, product_id);
CREATE INDEX idx_subscriptions_expires ON subscriptions(expires_at);
CREATE INDEX idx_sessions_user ON sessions(user_id) WHERE is_active = TRUE;
CREATE INDEX idx_sessions_token ON sessions(refresh_token_hash);
CREATE INDEX idx_anomaly_user ON anomaly_logs(user_id, created_at DESC);
CREATE INDEX idx_access_logs_user ON access_logs(user_id, created_at DESC);
CREATE INDEX idx_guest_codes_code ON guest_codes(code);
CREATE INDEX idx_guest_logins_code_email ON guest_logins(guest_code_id, email);

-- ─── SEED DATA ─────────────────────────────────────────────────────────

-- Product: Atomic
INSERT INTO products (id, name, description) VALUES
  ('atomic', 'Atomic — Interactive 3D Periodic Table', 'Aplikasi tabel periodik 3D interaktif untuk pelajar dan orang tua');

-- System config: quota defaults
INSERT INTO system_config (key, value, description) VALUES
  ('max_subscribers',       '200',  'Maks total subscriber aktif'),
  ('max_active_guests',     '50',   'Maks guest session aktif bersamaan'),
  ('quota_warning_pct',     '80',   'Notifikasi admin di persentase ini (%)'),
  ('guest_priority_mode',   'off',  'on = guest diblokir total saat > 90% subscriber quota');
