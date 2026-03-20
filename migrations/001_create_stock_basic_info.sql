-- Migration 001: Create stock_basic_info table
-- Stores basic stock information from Tushare stock_basic

BEGIN;

CREATE TABLE IF NOT EXISTS stock_basic_info (
    id              BIGSERIAL       PRIMARY KEY,
    ts_code         VARCHAR(16)     NOT NULL,
    symbol          VARCHAR(10)     NOT NULL,
    name            VARCHAR(64)     NOT NULL,
    exchange        VARCHAR(8)      NOT NULL,
    board_code      VARCHAR(16)     NOT NULL DEFAULT 'MAIN',
    area            VARCHAR(16),
    industry        VARCHAR(64),
    is_st           BOOLEAN         NOT NULL DEFAULT FALSE,
    list_date       DATE,
    status          SMALLINT        NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_stock_basic_info_ts_code UNIQUE (ts_code)
);

CREATE INDEX IF NOT EXISTS idx_stock_basic_info_board_code ON stock_basic_info (board_code);
CREATE INDEX IF NOT EXISTS idx_stock_basic_info_exchange   ON stock_basic_info (exchange);
CREATE INDEX IF NOT EXISTS idx_stock_basic_info_status     ON stock_basic_info (status) WHERE status = 1;

COMMENT ON TABLE  stock_basic_info                   IS '股票基础信息，来源 Tushare stock_basic';
COMMENT ON COLUMN stock_basic_info.ts_code           IS 'Tushare 股票代码，如 000001.SZ';
COMMENT ON COLUMN stock_basic_info.symbol            IS '纯数字代码，如 000001';
COMMENT ON COLUMN stock_basic_info.exchange          IS '交易所编码：SSE=上交所 SZSE=深交所 BSE=北交所';
COMMENT ON COLUMN stock_basic_info.board_code        IS '板块编码，用于涨跌停规则: MAIN/GEM/STAR/BSE';
COMMENT ON COLUMN stock_basic_info.is_st             IS '是否ST股，ST股不参与涨停监控';
COMMENT ON COLUMN stock_basic_info.status            IS '1=正常上市 0=停牌或退市';
COMMENT ON COLUMN stock_basic_info.area              IS '地域'

COMMIT;
