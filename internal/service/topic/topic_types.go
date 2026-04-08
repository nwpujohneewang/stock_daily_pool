// internal/service/topic_types.go
package topic

import (
	"errors"
	"time"
)

// TopicQueryParams 话题查询参数
type TopicQueryParams struct {
	Keyword  string
	Page     int
	PageSize int
}

// TopicQueryResult 话题查询结果
type TopicQueryResult struct {
	ID              int64
	Name            string
	Category        string
	Source          string
	OccurrenceCount int
	FirstSeenDate   *time.Time
	LastSeenDate    *time.Time
}

// TopicListResult 话题列表结果
type TopicListResult struct {
	Items []TopicQueryResult
	Total int64
}

// ErrTopicNotFound 话题不存在
var ErrTopicNotFound = errors.New("topic not found")

// ErrStockTopicRelationTopicNotFound 关联的话题不存在
var ErrStockTopicRelationTopicNotFound = errors.New("stock topic relation topic not found")
