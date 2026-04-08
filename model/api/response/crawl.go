package response

// CrawlJiuyanResp 韭研数据爬取响应
type CrawlJiuyanResp struct {
	Date        string `json:"date"`
	TopicsCount int    `json:"topics_count"`
	StocksCount int    `json:"stocks_count"`
}

// RebuildTopicRelationsResp 重建 topic relations 响应
type RebuildTopicRelationsResp struct {
	StartDate      string                 `json:"start_date"`
	EndDate        string                 `json:"end_date"`
	TopicsCount    int                    `json:"topics_count"`
	RelationsCount int                    `json:"relations_count"`
	Topics         []RebuildTopicInfoResp `json:"topics"`
}

// RebuildTopicInfoResp 重建的 topic 信息
type RebuildTopicInfoResp struct {
	Name           string `json:"name"`
	NormalizedName string `json:"normalized_name"`
	StocksCount    int    `json:"stocks_count"`
}
