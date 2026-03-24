package dal_model

import (
	"time"
)

type TopicDictionary struct {
	ID             int64     `gorm:"column:id"              json:"id"`
	RawTopicName   string    `gorm:"column:raw_topic_name"  json:"raw_topic_name"`
	NormalizedName string    `gorm:"column:normalized_name" json:"normalized_name"`
	Category       string    `gorm:"column:category"        json:"category"`
	Source         int       `gorm:"column:source"          json:"source"`
	CreatedAt      time.Time `gorm:"column:created_at"      json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"      json:"updated_at"`
}

func (TopicDictionary) TableName() string { return "topic_dictionary" }
