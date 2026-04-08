// model/api/response/topic_dict.go
package response

import "time"

// TopicDictItem 话题词典项
type TopicDictItem struct {
	ID             int64     `json:"id"`
	RawTopicName   string    `json:"raw_topic_name"`
	NormalizedName string    `json:"normalized_name"`
	Category       string    `json:"category"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
