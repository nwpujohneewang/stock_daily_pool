package utils

import (
	"testing"
)

type TopicScore struct {
	TopicID       int64   `json:"topic_id"`
	TopicName     string  `json:"topic_name"`
	ActivityScore float64 `json:"s_activity"`
	BindStrength  float64 `json:"s_bind_strength"`
	TimeProximity float64 `json:"s_time_proximity"`
	RecencyScore  float64 `json:"s_recency"`
	TotalScore    float64 `json:"total_score"`
	Source        string  `json:"source"`
	UsedHitCount  bool    `json:"used_hit_count"` // true = TotalScore was set from hit_count (last_seen > 30d)
}

func TestToString(t *testing.T) {
	s := &TopicScore{
		TopicID: 123,
	}

	ToString(s)
}
