-- migrations/015_create_daily_stock_snapshot.sql

CREATE TABLE daily_stock_snapshot (
    id                      BIGSERIAL,
    date                    DATE NOT NULL,
    ts_code                 VARCHAR(16) NOT NULL,
    stock_name              VARCHAR(64) NOT NULL,
    change_pct              NUMERIC(8,4),
    is_limit_up             BOOLEAN NOT NULL DEFAULT FALSE,
    limit_times             SMALLINT DEFAULT 0,
    consecutive_strong_days SMALLINT DEFAULT 1,
    total_mv                NUMERIC(16,2),
    vol                     NUMERIC(16,2),
    amount                  NUMERIC(16,2),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, date),
    UNIQUE (date, ts_code)
) PARTITION BY RANGE (date);

-- Create partitions for current and next months
CREATE TABLE daily_stock_snapshot_2026_03 PARTITION OF daily_stock_snapshot
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
CREATE TABLE daily_stock_snapshot_2026_04 PARTITION OF daily_stock_snapshot
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
CREATE TABLE daily_stock_snapshot_2026_05 PARTITION OF daily_stock_snapshot
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');

-- Indexes
CREATE INDEX idx_daily_stock_snapshot_date ON daily_stock_snapshot (date);
CREATE INDEX idx_daily_stock_snapshot_ts_code_date ON daily_stock_snapshot (ts_code, date);
