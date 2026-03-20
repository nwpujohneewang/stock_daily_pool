-- Migration 010: Create strategy_alerts table
-- Strategy alert records

BEGIN;

CREATE TABLE IF NOT EXISTS strategy_alerts (
    id              BIGSERIAL       PRIMARY KEY,
    date            DATE            NOT NULL,
    ts_code         VARCHAR(16)     NOT NULL,
    stock_name      VARCHAR(64)     NOT NULL,
    topic_id        BIGINT,
    topic_name      VARCHAR(128),
    alert_type      SMALLINT        NOT NULL DEFAULT 1,
    trigger_price   FLOAT,
    trigger_time    TIMESTAMPTZ,
    prev_day_pct    FLOAT,
    extra_info      JSONB,
    notified        BOOLEAN         NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_strategy_alerts_date     ON strategy_alerts (date);
CREATE INDEX IF NOT EXISTS idx_strategy_alerts_ts_code  ON strategy_alerts (ts_code, date);
CREATE INDEX IF NOT EXISTS idx_strategy_alerts_topic_id ON strategy_alerts (topic_id);
CREATE INDEX IF NOT EXISTS idx_strategy_alerts_notified ON strategy_alerts (notified) WHERE notified = FALSE;

COMMENT ON TABLE  strategy_alerts                       IS '策略告警记录';
COMMENT ON COLUMN strategy_alerts.alert_type            IS '告警类型：1=策略2(关注热点+昨日5%+今日涨停)';
COMMENT ON COLUMN strategy_alerts.extra_info            IS 'JSONB: {price, volume, amount, turnover, bid1-5, ask1-5, ...}';
COMMENT ON COLUMN strategy_alerts.notified              IS '是否已通过 WebSocket/飞书 推送';

COMMIT;
