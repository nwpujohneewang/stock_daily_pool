-- Migration 012: Create market_snapshots table
-- Market snapshots / crawl logs

BEGIN;

CREATE TABLE IF NOT EXISTS market_snapshots (
    id              BIGSERIAL       PRIMARY KEY,
    date            DATE            NOT NULL,
    status          SMALLINT        NOT NULL DEFAULT 0,
    raw_data        JSONB,
    topic_count     INT             NOT NULL DEFAULT 0,
    stock_count     INT             NOT NULL DEFAULT 0,
    retry_count     INT             NOT NULL DEFAULT 0,
    error_msg       TEXT,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_market_snapshots_date UNIQUE (date)
);

CREATE INDEX IF NOT EXISTS idx_market_snapshots_status ON market_snapshots (status);
CREATE INDEX IF NOT EXISTS idx_market_snapshots_date   ON market_snapshots (date DESC);

COMMENT ON TABLE  market_snapshots                  IS '市场快照 / 韭研爬取记录，每日一条';
COMMENT ON COLUMN market_snapshots.status           IS '0=待处理 1=成功 2=失败';
COMMENT ON COLUMN market_snapshots.raw_data         IS '韭研公社接口原始返回 JSON（可选，用于溯源）';
COMMENT ON COLUMN market_snapshots.topic_count      IS '该日热点数量';
COMMENT ON COLUMN market_snapshots.stock_count     IS '该日股票数量';
COMMENT ON COLUMN market_snapshots.retry_count     IS '爬取重试次数';

COMMIT;
