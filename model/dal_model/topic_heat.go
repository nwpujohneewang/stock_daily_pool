package dal_model

// TopicHeatInfo 表示单日 topic 热度快照。
// StockCount 表示 strongCount：涨幅严格大于 ClassifyThreshold(当前为 5%) 的股票数。
// RisingCount 表示涨幅严格大于 HeatRisingThreshold(当前为 3%) 的股票数。
type TopicHeatInfo struct {
	TopicID            int64   `json:"topic_id"`
	TotalRelatedStocks int     `json:"total_related_stocks"`
	StockCount         int     `json:"stock_count"`
	LimitUpCount       int     `json:"limit_up_count"`
	RisingCount        int     `json:"rising_count"`
	ParticipationRate  float64 `json:"participation_rate"`
	HeatScore          float64 `json:"heat_score"`
}
