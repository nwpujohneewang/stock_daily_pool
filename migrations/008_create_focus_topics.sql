-- Migration 008: Create focus_topics table
-- Daily focused topics for alert system

BEGIN;

CREATE TABLE IF NOT EXISTS focus_topics (
    id          BIGSERIAL       PRIMARY KEY,
    date        DATE            NOT NULL,
    topic_id    BIGINT          NOT NULL,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_focus_topics_date_topic UNIQUE (date, topic_id),
    CONSTRAINT fk_focus_topics_topic      FOREIGN KEY (topic_id)
        REFERENCES topics (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_focus_topics_date     ON focus_topics (date);
CREATE INDEX IF NOT EXISTS idx_focus_topics_topic_id ON focus_topics (topic_id);

COMMENT ON TABLE  focus_topics                IS '每日关注热点，策略引擎仅对关注热点告警';
COMMENT ON COLUMN focus_topics.date           IS '关注日期，格式 YYYY-MM-DD';
COMMENT ON COLUMN focus_topics.topic_id       IS '被关注的热点 ID';

COMMIT;
