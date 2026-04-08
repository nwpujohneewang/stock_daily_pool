-- migrations/016_add_market_value_to_stock_basic.sql

ALTER TABLE stock_basic_info
    ADD COLUMN IF NOT EXISTS total_mv NUMERIC(16,2),
    ADD COLUMN IF NOT EXISTS circ_mv  NUMERIC(16,2);
