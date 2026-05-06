BEGIN;

CREATE TABLE IF NOT EXISTS llm_classify_results (
    id          BIGSERIAL   PRIMARY KEY,
    ts_code     VARCHAR(16) NOT NULL,
    date        DATE        NOT NULL,
    topic       VARCHAR(64) NOT NULL,
    reason      TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_llm_classify_results_ts_code_date UNIQUE (ts_code, date)
);

CREATE INDEX IF NOT EXISTS idx_llm_classify_results_date ON llm_classify_results (date);
CREATE INDEX IF NOT EXISTS idx_llm_classify_results_ts_code ON llm_classify_results (ts_code);

COMMENT ON TABLE  llm_classify_results                IS 'LLM分类结果持久化表';
COMMENT ON COLUMN llm_classify_results.ts_code        IS '股票代码';
COMMENT ON COLUMN llm_classify_results.date           IS '分类日期';
COMMENT ON COLUMN llm_classify_results.topic          IS 'LLM分类的热点名称';
COMMENT ON COLUMN llm_classify_results.reason         IS 'LLM分类的理由';

COMMIT;
