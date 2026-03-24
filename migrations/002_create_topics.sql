-- Migration 002: Create topics table
-- Core entity: Hot topics/theme

BEGIN;

CREATE TABLE IF NOT EXISTS topics (
    id                BIGSERIAL       PRIMARY KEY,
    name              VARCHAR(128)    NOT NULL,
    source            VARCHAR(16)     NOT NULL DEFAULT 'jiuyan',
    jiuyan_field_id   VARCHAR(64),
    first_seen_date   DATE,
    last_seen_date    DATE,
    occurrence_count  INT             NOT NULL DEFAULT 0,
    priority          INT             NOT NULL DEFAULT 0,
    is_active         BOOLEAN         NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    category          VARCHAR(32),
    CONSTRAINT uq_topics_name UNIQUE (name)
);

CREATE INDEX IF NOT EXISTS idx_topics_source   ON topics (source);
CREATE INDEX IF NOT EXISTS idx_topics_active   ON topics (is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_topics_priority ON topics (priority DESC) WHERE is_active = TRUE;

COMMENT ON TABLE  topics                        IS '热点主题库，核心实体';
COMMENT ON COLUMN topics.name                   IS '热点名称，全局唯一';
COMMENT ON COLUMN topics.source                 IS '数据来源：jiuyan=韭研公社 manual=人工创建';
COMMENT ON COLUMN topics.jiuyan_field_id        IS '韭研公社的 action_field_id，用于数据溯源';
COMMENT ON COLUMN topics.first_seen_date        IS '该热点首次出现的日期';
COMMENT ON COLUMN topics.last_seen_date         IS '该热点最近一次出现的日期';
COMMENT ON COLUMN topics.occurrence_count       IS '历史出现天数，用于热点活跃度评估';
COMMENT ON COLUMN topics.priority               IS '展示和处理优先级，数值越大越优先';
COMMENT ON COLUMN topics.is_active              IS 'FALSE=软删除，系统不再使用但保留历史数据';

COMMIT;
