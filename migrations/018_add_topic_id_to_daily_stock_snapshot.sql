ALTER TABLE daily_stock_snapshot
    ADD COLUMN IF NOT EXISTS topic_id BIGINT;

CREATE INDEX IF NOT EXISTS idx_daily_stock_snapshot_date_topic_id
    ON daily_stock_snapshot (date, topic_id);
