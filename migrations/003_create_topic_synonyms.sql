-- Migration 003: Create topic_synonyms table
-- Hot topic synonym normalization

BEGIN;

CREATE TABLE IF NOT EXISTS topic_synonyms (
    id          BIGSERIAL       PRIMARY KEY,
    topic_id    BIGINT          NOT NULL,
    synonym     VARCHAR(128)    NOT NULL,
    source      VARCHAR(16)     NOT NULL DEFAULT 'jiuyan',
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_topic_synonyms_synonym UNIQUE (synonym),
    CONSTRAINT fk_topic_synonyms_topic   FOREIGN KEY (topic_id)
        REFERENCES topics (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_topic_synonyms_topic_id ON topic_synonyms (topic_id);

COMMENT ON TABLE  topic_synonyms                IS '热点同义词，用于名称归一化';
COMMENT ON COLUMN topic_synonyms.topic_id       IS '归并到的目标热点 ID';
COMMENT ON COLUMN topic_synonyms.synonym        IS '同义词文本，全局唯一';
COMMENT ON COLUMN topic_synonyms.source         IS '来源：jiuyan=爬虫识别 manual=手动添加 auto=自动检测';

COMMIT;
