package dal_model

import (
	"time"
)

type StockTopicRelation struct {
	ID            int64      `gorm:"column:id"              json:"id"`
	TsCode        string     `gorm:"column:ts_code"         json:"ts_code"`
	TopicID       int64      `gorm:"column:topic_id"        json:"topic_id"`
	TopicName     string     `gorm:"column:topic_name"      json:"topic_name"`
	Category      string     `gorm:"column:category"        json:"category"`
	Source        string     `gorm:"column:source"          json:"source"`
	Confidence    *float64   `gorm:"column:confidence"      json:"confidence,omitempty"`
	HitCount      int        `gorm:"column:hit_count"        json:"hit_count"`
	LastSeenDate  *time.Time `gorm:"column:last_seen_date"   json:"last_seen_date,omitempty"`
	FirstSeenDate *time.Time `gorm:"column:first_seen_date"  json:"first_seen_date,omitempty"`
	CreatedAt     time.Time  `gorm:"column:created_at"       json:"created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"       json:"updated_at"`
}

func (StockTopicRelation) TableName() string { return "stock_topic_relations" }

type TopicRelation struct {
	TopicID      int64   `json:"topic_id"`
	TopicName    string  `json:"topic_name"`
	Category     string  `json:"category"`
	Source       string  `json:"source"`
	HitCount     int     `json:"hit_count"`
	LastSeenDate string  `json:"last_seen_date"`
	Confidence   float64 `json:"confidence"`
}
