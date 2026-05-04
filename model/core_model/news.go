package core_model

import "time"

// RawNews 原始新闻（所有源统一格式）
type RawNews struct {
	NewsID          string    `json:"news_id"` // 全局唯一: {source}_{原始ID}
	Source          string    `json:"source"`  // eastmoney / cls
	Title           string    `json:"title"`
	Content         string    `json:"content"`
	PublishTime     time.Time `json:"publish_time"`
	URL             string    `json:"url"`
	Importance      string    `json:"importance"` // high / medium / low
	MentionedStocks []string  `json:"mentioned_stocks,omitempty"`
}

// HotTopic 提取出的热点主题
type HotTopic struct {
	TopicName     string   `json:"topic_name"` // 如"AI算力"
	TopicL1       string   `json:"topic_l1"`   // L1板块
	TopicL2       string   `json:"topic_l2"`   // L2概念
	Summary       string   `json:"summary"`    // 50字摘要
	Catalyst      string   `json:"catalyst"`   // 驱动因素
	Keywords      []string `json:"keywords"`   // 匹配用关键词
	HotLevel      string   `json:"hot_level"`  // high / medium / low
	SourceNewsIDs []string `json:"source_news_ids"`
}

// NewsExtractResult 单条新闻LLM结构化提取结果
type NewsExtractResult struct {
	NewsID     string   `json:"news_id"`
	Summary    string   `json:"summary"`
	Keywords   []string `json:"keywords"`
	Sector     string   `json:"sector"`
	Impact     string   `json:"impact"` // 利好/利空/中性
	Importance string   `json:"importance"`
	EventType  string   `json:"event_type"` // 政策/业绩/行业动态
}

// TopicStockRelation 热点关联的股票
type TopicStockRelation struct {
	TopicName string `json:"topic_name"`
	TsCode    string `json:"ts_code"`
	StockName string `json:"stock_name"`
	Relevance string `json:"relevance"` // core / related
	Reason    string `json:"reason"`
}
