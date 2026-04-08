package dal_model

// TopicHeatInfo 表示单日 topic 热度快照。
// StockCount 表示 strongCount：涨幅严格大于 ClassifyThreshold(当前为 5%) 的股票数。
// RisingCount 表示涨幅严格大于 HeatRisingThreshold(当前为 3%) 的股票数。
// AvgGain 仅用于 topic_stock_count 全量缺失时的退化计算，不参与主热度公式。
type TopicHeatInfo struct {
	TopicID            int64   `json:"topic_id"`
	TotalRelatedStocks int     `json:"total_related_stocks"`
	StockCount         int     `json:"stock_count"`
	LimitUpCount       int     `json:"limit_up_count"`
	RisingCount        int     `json:"rising_count"`
	AvgGain            float64 `json:"avg_gain"`
	ParticipationRate  float64 `json:"participation_rate"`
	OverallStrength    float64 `json:"overall_strength"`
	HeatScore          float64 `json:"heat_score"`
}
