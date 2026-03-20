-- Migration 013: Create jiuyan raw data table
-- Raw crawled data from jiuyan API

BEGIN;

CREATE TABLE IF NOT EXISTS jiuyan_raw_data (
    id              BIGSERIAL       PRIMARY KEY,
    date            DATE            NOT NULL,
    topic_name      VARCHAR(128)    NOT NULL,
    action_field_id VARCHAR(64),
    stock_code      VARCHAR(32)     NOT NULL,
    stock_name      VARCHAR(64)      NOT NULL,
    expound         TEXT,
    raw_json        JSONB,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_jiuyan_raw_date_topic_stock UNIQUE (date, topic_name, stock_code)
);

CREATE INDEX IF NOT EXISTS idx_jiuyan_raw_date       ON jiuyan_raw_data (date);
CREATE INDEX IF NOT EXISTS idx_jiuyan_raw_topic_name ON jiuyan_raw_data (topic_name);
CREATE INDEX IF NOT EXISTS idx_jiuyan_raw_stock_code ON jiuyan_raw_data (stock_code);

COMMENT ON TABLE  jiuyan_raw_data                       IS '韭研公社原始爬取数据，按股票维度存储每条记录';
COMMENT ON COLUMN jiuyan_raw_data.date               IS '爬取目标日期';
COMMENT ON COLUMN jiuyan_raw_data.topic_name          IS '热点名称';
COMMENT ON COLUMN jiuyan_raw_data.action_field_id    IS '韭研公社的 action_field_id';
COMMENT ON COLUMN jiuyan_raw_data.stock_code         IS '股票代码（韭研格式，如 sz002445）';
COMMENT ON COLUMN jiuyan_raw_data.stock_name          IS '股票名称';
COMMENT ON COLUMN jiuyan_raw_data.expound             IS '涨停原因描述';
COMMENT ON COLUMN jiuyan_raw_data.raw_json            IS '原始 JSON 响应（用于溯源）';

COMMIT;
