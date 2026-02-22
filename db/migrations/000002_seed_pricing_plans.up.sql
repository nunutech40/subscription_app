-- =============================================
-- SAINS API — Seed Pricing Plans
-- Migration: 000002_seed_pricing_plans
-- =============================================

-- 3 segments × 4 durations = 12 plans untuk product 'atomic'

-- ─── GLOBAL SEGMENT ──────────────────────────────────────────────────
INSERT INTO pricing_plans (product_id, segment, duration, duration_days, price_idr, label, is_active) VALUES
  ('atomic', 'global', 'monthly',  30,  49000,  'Bulanan',              TRUE),
  ('atomic', 'global', '3month',   90,  129000, '3 Bulan — Hemat 12%',  TRUE),
  ('atomic', 'global', '6month',   180, 239000, '6 Bulan — Hemat 19%',  TRUE),
  ('atomic', 'global', 'yearly',   365, 399000, 'Tahunan — Hemat 32%',  TRUE);

-- ─── STUDENT SEGMENT ─────────────────────────────────────────────────
INSERT INTO pricing_plans (product_id, segment, duration, duration_days, price_idr, label, is_active) VALUES
  ('atomic', 'student', 'monthly',  30,  29000,  'Bulanan (Pelajar)',              TRUE),
  ('atomic', 'student', '3month',   90,  79000,  '3 Bulan (Pelajar) — Hemat 9%',  TRUE),
  ('atomic', 'student', '6month',   180, 149000, '6 Bulan (Pelajar) — Hemat 14%', TRUE),
  ('atomic', 'student', 'yearly',   365, 249000, 'Tahunan (Pelajar) — Hemat 28%', TRUE);

-- ─── PARENT SEGMENT ──────────────────────────────────────────────────
INSERT INTO pricing_plans (product_id, segment, duration, duration_days, price_idr, label, is_active) VALUES
  ('atomic', 'parent', 'monthly',  30,  59000,  'Bulanan (Orang Tua)',              TRUE),
  ('atomic', 'parent', '3month',   90,  159000, '3 Bulan (Orang Tua) — Hemat 10%', TRUE),
  ('atomic', 'parent', '6month',   180, 299000, '6 Bulan (Orang Tua) — Hemat 16%', TRUE),
  ('atomic', 'parent', 'yearly',   365, 499000, 'Tahunan (Orang Tua) — Hemat 30%', TRUE);
