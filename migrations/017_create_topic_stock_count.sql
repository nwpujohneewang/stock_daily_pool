CREATE TABLE IF NOT EXISTS topic_stock_count (
    topic_id BIGINT NOT NULL,
    total_stock_count INT NOT NULL DEFAULT 0,
    computed_date DATE NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (topic_id)
);

COMMENT ON TABLE topic_stock_count IS '每个 topic 的历史关联股票总数（每日预计算）';
COMMENT ON COLUMN topic_stock_count.total_stock_count IS 'COUNT(DISTINCT ts_code) from stock_topic_relations';
COMMENT ON COLUMN topic_stock_count.computed_date IS '计算日期';
