-- Migration 009: Create daily_stock_pool table (with partitioning)
-- Daily stock pool snapshots, partitioned by month

BEGIN;

CREATE TABLE IF NOT EXISTS daily_stock_pool (
    id              BIGSERIAL,
    date            DATE            NOT NULL,
    ts_code         VARCHAR(16)     NOT NULL,
    stock_name      VARCHAR(64)     NOT NULL,
    pool_type       SMALLINT        NOT NULL,
    change_pct      FLOAT,
    current_price   FLOAT,
    pre_close       FLOAT,
    limit_up_price  FLOAT,
    first_limit_time TIME,
    board_code      VARCHAR(16),
    topic_ids       BIGINT[],
    snapshot_time   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_daily_stock_pool_date_ts_pool UNIQUE (date, ts_code, pool_type),
    PRIMARY KEY (id, date)
) PARTITION BY RANGE (date);

CREATE INDEX IF NOT EXISTS idx_daily_stock_pool_date      ON daily_stock_pool (date);
CREATE INDEX IF NOT EXISTS idx_daily_stock_pool_ts_code   ON daily_stock_pool (ts_code, date);
CREATE INDEX IF NOT EXISTS idx_daily_stock_pool_pool_type ON daily_stock_pool (date, pool_type);
CREATE INDEX IF NOT EXISTS idx_daily_stock_pool_topic_ids ON daily_stock_pool USING GIN (topic_ids);

COMMENT ON TABLE  daily_stock_pool                      IS '每日股票池快照，按月分区';
COMMENT ON COLUMN daily_stock_pool.pool_type            IS '1=涨停池 2=涨幅大于5%池';
COMMENT ON COLUMN daily_stock_pool.change_pct           IS '涨跌幅百分比，如 9.98 表示 +9.98%';
COMMENT ON COLUMN daily_stock_pool.first_limit_time     IS '首次触及涨停时间，仅 pool_type=1 有值';
COMMENT ON COLUMN daily_stock_pool.topic_ids            IS '当日归因热点ID数组，支持 GIN 索引查询';

COMMIT;
