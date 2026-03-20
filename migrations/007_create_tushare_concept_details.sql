-- Migration 007: Create tushare_concept_details table
-- Concept board component stocks from Tushare

BEGIN;

CREATE TABLE IF NOT EXISTS tushare_concept_details (
    id              BIGSERIAL       PRIMARY KEY,
    ts_code         VARCHAR(16)     NOT NULL,
    concept_name    VARCHAR(128)    NOT NULL,
    concept_code    VARCHAR(16)     NOT NULL,
    source          VARCHAR(16)     NOT NULL DEFAULT 'tushare',
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tushare_concept_details_ts_concept UNIQUE (ts_code, concept_name)
);

CREATE INDEX IF NOT EXISTS idx_tushare_concept_details_ts_code      ON tushare_concept_details (ts_code);
CREATE INDEX IF NOT EXISTS idx_tushare_concept_details_concept_code ON tushare_concept_details (concept_code);

COMMENT ON TABLE  tushare_concept_details                   IS '股票基础概念标签，来源 Tushare concept_detail';
COMMENT ON COLUMN tushare_concept_details.ts_code           IS '股票代码，如 000001.SZ';
COMMENT ON COLUMN tushare_concept_details.concept_name      IS '概念名称，如"人工智能"';
COMMENT ON COLUMN tushare_concept_details.concept_code     IS 'Tushare 概念代码，如 TS0001';

COMMIT;
