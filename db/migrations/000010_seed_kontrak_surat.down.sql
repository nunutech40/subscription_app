-- =============================================
-- SAINS API — Rollback: Kontrak & Surat Products + Pricing Plans
-- Migration: 000010_seed_kontrak_surat (DOWN)
-- =============================================

DELETE FROM pricing_plans WHERE product_id = 'kontrak';
DELETE FROM products WHERE id = 'kontrak';
DELETE FROM products WHERE id = 'surat';
