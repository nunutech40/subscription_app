-- =============================================
-- SAINS API — Tambah Products: Kontrak & Surat + Pricing Plans Kontrak
-- Migration: 000010_seed_kontrak_surat
-- Date: 2026-03-25
--
-- Yang SUDAH ada (jangan disentuh):
--   - atomic (product + 12 plans) ✅
--   - dokumen-template (product + 3 plans) ✅
--   - klinik (product + 2 plans) ✅
--
-- Yang DITAMBAHKAN di sini:
--   - kontrak (product baru + pricing plans)
--   - surat (product baru saja, plans pakai dokumen-template)
-- =============================================

-- ─── PRODUCTS BARU ───────────────────────────────────────────────────

INSERT INTO products (id, name, description) VALUES
  ('kontrak', 'KontrakPro — Proposal, Kontrak & Invoice Generator', 'Platform all-in-one untuk freelancer Indonesia. Buat proposal, generate kontrak otomatis, kirim invoice.')
ON CONFLICT (id) DO NOTHING;

INSERT INTO products (id, name, description) VALUES
  ('surat', 'Surat — Generator Surat Resmi Indonesia', 'Generator surat resmi Indonesia dengan 40+ template. Surat lamaran, keterangan, pengantar, kuasa, dan lainnya.')
ON CONFLICT (id) DO NOTHING;

-- ─── PRICING PLANS: KONTRAK ─────────────────────────────────────────
-- Harga sesuai landing page kontrak-v1

INSERT INTO pricing_plans (product_id, segment, duration, duration_days, price_idr, label, is_active) VALUES
  ('kontrak', 'global', 'monthly',  30,  199000,  'Starter — Bulanan',    TRUE),
  ('kontrak', 'global', '3month',   90,  499000,  'Starter — 3 Bulan',    TRUE),
  ('kontrak', 'global', '6month',   180, 899000,  'Starter — 6 Bulan',    TRUE),
  ('kontrak', 'global', 'yearly',   365, 1499000, 'Starter — Tahunan',    TRUE);
