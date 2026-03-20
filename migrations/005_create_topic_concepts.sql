-- Migration 005: Create topic_concepts table
-- Mapping between Tushare concepts and system topics

BEGIN;

CREATE TABLE IF NOT EXISTS topic_concepts (
    id              BIGSERIAL       PRIMARY KEY,
    concept_name    VARCHAR(128)    NOT NULL,
    concept_code    VARCHAR(16)     NOT NULL,
    topic_id        BIGINT          NOT NULL,
    match_type      VARCHAR(16)     NOT NULL DEFAULT 'exact',
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_topic_concepts_name_topic UNIQUE (concept_name, topic_id),
    CONSTRAINT fk_topic_concepts_topic      FOREIGN KEY (topic_id)
        REFERENCES topics (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_topic_concepts_topic_id     ON topic_concepts (topic_id);
CREATE INDEX IF NOT EXISTS idx_topic_concepts_concept_code ON topic_concepts (concept_code);

COMMENT ON TABLE  topic_concepts                    IS '概念板块 → 热点映射，多对多';
COMMENT ON COLUMN topic_concepts.concept_name       IS 'Tushare 概念名称';
COMMENT ON COLUMN topic_concepts.concept_code       IS 'Tushare 概念代码';
COMMENT ON COLUMN topic_concepts.topic_id           IS '映射到的系统热点 ID';
COMMENT ON COLUMN topic_concepts.match_type         IS '匹配方式：exact=精确匹配 synonym=同义词 llm=LLM匹配 manual=人工标注';

COMMIT;
