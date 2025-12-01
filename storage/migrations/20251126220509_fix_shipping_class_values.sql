-- +goose Up
-- Fix shipping class values to match expected categories (small/medium/large/xlarge)
-- The original migration used First/Priority which don't match the CASE statements in shipping queries

UPDATE size_charts SET default_shipping_class = 'small' WHERE id = 'chart_small';
UPDATE size_charts SET default_shipping_class = 'medium' WHERE id = 'chart_medium';
UPDATE size_charts SET default_shipping_class = 'large' WHERE id = 'chart_large';
UPDATE size_charts SET default_shipping_class = 'xlarge' WHERE id = 'chart_xlarge';

-- +goose Down
-- Revert to original values (not recommended)
UPDATE size_charts SET default_shipping_class = 'First' WHERE id = 'chart_small';
UPDATE size_charts SET default_shipping_class = 'First' WHERE id = 'chart_medium';
UPDATE size_charts SET default_shipping_class = 'Priority' WHERE id = 'chart_large';
UPDATE size_charts SET default_shipping_class = 'Priority' WHERE id = 'chart_xlarge';
