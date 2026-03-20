-- Migration 004: Create tushare_concepts table
-- Tushare concept board list

BEGIN;

CREATE TABLE IF NOT EXISTS tushare_concepts (
    id              BIGSERIAL       PRIMARY KEY,
    concept_code    VARCHAR(16)     NOT NULL,
    concept_name    VARCHAR(128)    NOT NULL,
    source          VARCHAR(16)     NOT NULL DEFAULT 'tushare',
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tushare_concepts_code UNIQUE (concept_code),
    CONSTRAINT uq_tushare_concepts_name UNIQUE (concept_name)
);

COMMENT ON TABLE  tushare_concepts                   IS 'Tushare 概念板块列表';
COMMENT ON COLUMN tushare_concepts.concept_code      IS 'Tushare 概念代码，如 TS0001';
COMMENT ON COLUMN tushare_concepts.concept_name      IS '概念名称，如"人工智能"、"芯片概念"';

COMMIT;
