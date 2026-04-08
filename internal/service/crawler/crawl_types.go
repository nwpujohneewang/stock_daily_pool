package crawler

import "errors"

// CrawlParams 爬取参数（从 curl 解析）
type CrawlParams struct {
	Date      string
	Token     string
	Cookie    string
	Timestamp string
}

// CrawlResult 爬取结果
type CrawlResult struct {
	Date        string
	TopicsCount int
	StocksCount int
}

// MissingTopicsError 缺失 topic 的错误
type MissingTopicsError struct {
	Topics []string
}

func (e *MissingTopicsError) Error() string {
	return "some topics missing from dictionary"
}

// ErrMissingTopics 缺失 topic 错误标识
var ErrMissingTopics = errors.New("some topics missing from dictionary")

// ErrInvalidDate 日期格式错误
var ErrInvalidDate = errors.New("invalid date format")

// ErrEmptyDate 日期为空
var ErrEmptyDate = errors.New("date is empty")

// ErrInvalidDateRange 日期范围错误
var ErrInvalidDateRange = errors.New("start_date must not be after end_date")

// RebuildResult 重建结果
type RebuildResult struct {
	StartDate      string             `json:"start_date"`
	EndDate        string             `json:"end_date"`
	TopicsCount    int                `json:"topics_count"`
	RelationsCount int                `json:"relations_count"`
	Topics         []RebuildTopicInfo `json:"topics"`
}

// RebuildTopicInfo 重建的 topic 信息
type RebuildTopicInfo struct {
	Name           string `json:"name"`            // 原始 topic 名称
	NormalizedName string `json:"normalized_name"` // 归一化后名称
	StocksCount    int    `json:"stocks_count"`    // 关联股票数
}
