package dal_model

import (
	"time"
)

type TopicSynonym struct {
	ID        int64     `gorm:"column:id"         json:"id"`
	TopicID   int64     `gorm:"column:topic_id"   json:"topic_id"`
	Synonym   string    `gorm:"column:synonym"    json:"synonym"`
	Source    string    `gorm:"column:source"     json:"source"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

func (TopicSynonym) TableName() string { return "topic_synonyms" }
