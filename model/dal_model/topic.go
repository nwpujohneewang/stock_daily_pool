package dal_model

import (
	"time"
)

type Topic struct {
	ID              int64      `gorm:"column:id"               json:"id"`
	Name            string     `gorm:"column:name"             json:"name"`
	Source          string     `gorm:"column:source"            json:"source"`
	JiuyanFieldID   *string    `gorm:"column:jiuyan_field_id"  json:"jiuyan_field_id,omitempty"`
	FirstSeenDate   *time.Time `gorm:"column:first_seen_date"  json:"first_seen_date,omitempty"`
	LastSeenDate    *time.Time `gorm:"column:last_seen_date"   json:"last_seen_date,omitempty"`
	OccurrenceCount int        `gorm:"column:occurrence_count"  json:"occurrence_count"`
	Priority        int        `gorm:"column:priority"          json:"priority"`
	IsActive        bool       `gorm:"column:is_active"         json:"is_active"`
	CreatedAt       time.Time  `gorm:"column:created_at"        json:"created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"        json:"updated_at"`
}

func (Topic) TableName() string { return "topics" }
