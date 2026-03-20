-- Migration 006: Create stock_topic_relations table
-- Core mapping table: stock to topic relationships

BEGIN;

CREATE TABLE IF NOT EXISTS stock_topic_relations (
    id              BIGSERIAL       PRIMARY KEY,
    ts_code         VARCHAR(16)     NOT NULL,
    topic_id        BIGINT          NOT NULL,
    source          VARCHAR(16)     NOT NULL DEFAULT 'jiuyan',
    confidence      FLOAT,
    hit_count       INT             NOT NULL DEFAULT 1,
    last_seen_date  DATE,
    first_seen_date DATE,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_stock_topic_relations_ts_topic UNIQUE (ts_code, topic_id),
    CONSTRAINT fk_stock_topic_relations_topic    FOREIGN KEY (topic_id)
        REFERENCES topics (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_stock_topic_relations_ts_code  ON stock_topic_relations (ts_code);
CREATE INDEX IF NOT EXISTS idx_stock_topic_relations_topic_id ON stock_topic_relations (topic_id);
CREATE INDEX IF NOT EXISTS idx_stock_topic_relations_source   ON stock_topic_relations (source);

COMMENT ON TABLE  stock_topic_relations                     IS '股票-热点关联映射，核心表';
COMMENT ON COLUMN stock_topic_relations.ts_code             IS '股票代码，如 000001.SZ';
COMMENT ON COLUMN stock_topic_relations.topic_id            IS '关联的热点 ID';
COMMENT ON COLUMN stock_topic_relations.source              IS '分类来源：jiuyan=韭研爬虫 llm=LLM分类 manual=人工标注(最高优先级)';
COMMENT ON COLUMN stock_topic_relations.confidence          IS '分类置信度 0~1，LLM 分类时有意义';
COMMENT ON COLUMN stock_topic_relations.hit_count           IS '韭研历史标注频次，用于归因算法的频次维度权重';
COMMENT ON COLUMN stock_topic_relations.last_seen_date      IS '最近一次在韭研数据中出现的日期';
COMMENT ON COLUMN stock_topic_relations.first_seen_date     IS '首次在韭研数据中出现的日期';

COMMIT;
