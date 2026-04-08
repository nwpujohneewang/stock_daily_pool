// internal/service/stock_types.go
package stock

import (
	"errors"
	"time"
)

// StockSearchParams 股票搜索参数
type StockSearchParams struct {
	Query    string
	Topic    string
	Category string
	Page     int
	PageSize int
}

// StockResult 股票结果
type StockResult struct {
	TsCode    string
	Symbol    string
	Name      string
	Exchange  string
	BoardCode string
	Industry  *string
	IsST      bool
	ListDate  *time.Time
}

// TopicRelationResult 话题关联结果
type TopicRelationResult struct {
	TopicID       int64
	TopicName     string
	Category      string
	Source        string
	HitCount      int
	FirstSeenDate *time.Time
	LastSeenDate  *time.Time
}

// StockDetailResult 股票详情结果
type StockDetailResult struct {
	Info   StockResult
	Topics []TopicRelationResult
}

// StockSearchResult 股票搜索结果
type StockSearchResult struct {
	Items []StockResult
	Total int64
}

// ErrStockNotFound 股票不存在
var ErrStockNotFound = errors.New("stock not found")
