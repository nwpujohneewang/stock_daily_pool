// Package request provides API request DTOs.
package request

// CrawlJiuyanReq 韭研数据爬取请求
type CrawlJiuyanReq struct {
	Curl string `json:"curl" binding:"required"`
}

// RebuildTopicRelationsReq 重建 topic relations 请求
type RebuildTopicRelationsReq struct {
	StartDate string `json:"start_date" binding:"required"`
	EndDate   string `json:"end_date" binding:"required"`
}
