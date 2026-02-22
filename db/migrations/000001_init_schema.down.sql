-- =============================================
-- SAINS API — Rollback Initial Schema
-- Migration: 000001_init_schema (DOWN)
-- =============================================

-- Drop in reverse order (respect foreign keys)
DROP TABLE IF EXISTS access_logs CASCADE;
DROP TABLE IF EXISTS anomaly_logs CASCADE;
DROP TABLE IF EXISTS sessions CASCADE;
DROP TABLE IF EXISTS subscriptions CASCADE;
DROP TABLE IF EXISTS guest_logins CASCADE;
DROP TABLE IF EXISTS guest_codes CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS pricing_plans CASCADE;
DROP TABLE IF EXISTS products CASCADE;
DROP TABLE IF EXISTS system_config CASCADE;
