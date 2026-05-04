package response

// LLMClassifyItem 单股 LLM 分类结果
type LLMClassifyItem struct {
	TsCode     string  `json:"ts_code"`
	Name       string  `json:"name"`
	TopicName  string  `json:"topic_name"`
	TopicID    int64   `json:"topic_id"`
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
}

// LLMRunHistoryResp POST /llm-classify/run-history 响应
type LLMRunHistoryResp struct {
	Status    string            `json:"status"`
	Date      string            `json:"date"`
	ElapsedMs int64             `json:"elapsed_ms"`
	Total     int               `json:"total"`
	Results   []LLMClassifyItem `json:"results"`
}
