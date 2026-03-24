BEGIN;

CREATE TABLE IF NOT EXISTS topic_dictionary
(
    id              BIGSERIAL PRIMARY KEY,
    raw_topic_name  VARCHAR(64) NOT NULL,
    normalized_name VARCHAR(64) NOT NULL,
    category        VARCHAR(64) NOT NULL,
    source          INT NOT NULL ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );


COMMENT ON TABLE topic_dictionary IS 'topic转换';
COMMENT ON COLUMN topic_dictionary.raw_topic_name IS '原始话题';
COMMENT ON COLUMN topic_dictionary.normalized_name IS '转换后话题';
COMMENT ON COLUMN topic_dictionary.category IS '大类';

COMMIT;