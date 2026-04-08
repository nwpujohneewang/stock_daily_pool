// model/api/response/topic.go
package response

import "time"

// TopicItem 话题项
type TopicItem struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Category        string     `json:"category"`
	Source          string     `json:"source"`
	OccurrenceCount int        `json:"occurrence_count"`
	FirstSeenDate   *time.Time `json:"first_seen_date,omitempty"`
	LastSeenDate    *time.Time `json:"last_seen_date,omitempty"`
}

// TopicListResp 话题列表响应
type TopicListResp struct {
	Items []TopicItem `json:"items"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Pages int64       `json:"pages"`
}
