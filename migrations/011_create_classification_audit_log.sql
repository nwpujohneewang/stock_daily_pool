-- Migration 011: Create classification_audit_log table
-- Classification evidence audit log

BEGIN;

CREATE TABLE IF NOT EXISTS classification_audit_log (
    id                  BIGSERIAL       PRIMARY KEY,
    date                DATE            NOT NULL,
    ts_code             VARCHAR(16)     NOT NULL,
    topic_id            BIGINT,
    classify_layer      VARCHAR(32)     NOT NULL,
    strategy            VARCHAR(32)     NOT NULL,
    candidate_scores    JSONB,
    evidence_text       TEXT,
    confidence          FLOAT,
    corrected_topic_id  BIGINT,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_classification_audit_log_stock_date ON classification_audit_log (ts_code, date);
CREATE INDEX IF NOT EXISTS idx_classification_audit_log_layer      ON classification_audit_log (classify_layer);
CREATE INDEX IF NOT EXISTS idx_classification_audit_log_date       ON classification_audit_log (date);

COMMENT ON TABLE  classification_audit_log                          IS '分类证据审计日志，核心数据资产';
COMMENT ON COLUMN classification_audit_log.classify_layer           IS '命中层级：L1_REDIS / L2_PG_JIUYAN / L3_PG_CONCEPT / L4_LLM';
COMMENT ON COLUMN classification_audit_log.strategy                 IS '分类策略：JIUYAN_ATTR / CONCEPT_ATTR / LLM_V1 / MANUAL';
COMMENT ON COLUMN classification_audit_log.candidate_scores         IS 'JSONB: 候选热点及频次/时效/概念匹配/热度四维评分';
COMMENT ON COLUMN classification_audit_log.confidence               IS '最终置信度 0~1';
COMMENT ON COLUMN classification_audit_log.corrected_topic_id       IS '人工修正后的热点ID，用于后期训练数据积累';

COMMIT;
