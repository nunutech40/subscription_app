-- Rollback: remove seeded pricing plans
DELETE FROM pricing_plans WHERE product_id = 'atomic';
