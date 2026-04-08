package dal_model

import "time"

type TopicStockCount struct {
	TopicID         int64     `gorm:"column:topic_id" json:"topic_id"`
	TotalStockCount int       `gorm:"column:total_stock_count" json:"total_stock_count"`
	ComputedDate    time.Time `gorm:"column:computed_date" json:"computed_date"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (TopicStockCount) TableName() string { return "topic_stock_count" }
