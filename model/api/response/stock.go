// model/api/response/stock.go
package response

import "time"

// StockItem 股票项
type StockItem struct {
	TsCode    string     `json:"ts_code"`
	Symbol    string     `json:"symbol"`
	Name      string     `json:"name"`
	Exchange  string     `json:"exchange"`
	BoardCode string     `json:"board_code"`
	Industry  *string    `json:"industry,omitempty"`
	IsST      bool       `json:"is_st"`
	ListDate  *time.Time `json:"list_date,omitempty"`
}

// TopicRelationItem 话题关联项
type TopicRelationItem struct {
	TopicID       int64      `json:"topic_id"`
	TopicName     string     `json:"topic_name"`
	Category      string     `json:"category"`
	Source        string     `json:"source"`
	HitCount      int        `json:"hit_count"`
	FirstSeenDate *time.Time `json:"first_seen_date,omitempty"`
	LastSeenDate  *time.Time `json:"last_seen_date,omitempty"`
}

// StockDetail 股票详情
type StockDetail struct {
	Info   StockItem           `json:"info"`
	Topics []TopicRelationItem `json:"topics"`
}

// StockSearchResp 股票搜索响应
type StockSearchResp struct {
	Items    []StockItem `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}
