package dal_model

import "time"

type DailyStockSnapshot struct {
	ID                    int64     `gorm:"column:id"                      json:"id"`
	Date                  time.Time `gorm:"column:date"                    json:"date"`
	TsCode                string    `gorm:"column:ts_code"                 json:"ts_code"`
	StockName             string    `gorm:"column:stock_name"              json:"stock_name"`
	TopicID               *int64    `gorm:"column:topic_id"                json:"topic_id,omitempty"`
	ChangePct             *float64  `gorm:"column:change_pct"              json:"change_pct,omitempty"`
	IsLimitUp             bool      `gorm:"column:is_limit_up"             json:"is_limit_up"`
	LimitTimes            int16     `gorm:"column:limit_times"             json:"limit_times"`
	ConsecutiveStrongDays int16     `gorm:"column:consecutive_strong_days" json:"consecutive_strong_days"`
	TotalMv               *float64  `gorm:"column:total_mv"                json:"total_mv,omitempty"`
	Vol                   *float64  `gorm:"column:vol"                     json:"vol,omitempty"`
	Amount                *float64  `gorm:"column:amount"                  json:"amount,omitempty"`
	CreatedAt             time.Time `gorm:"column:created_at"              json:"created_at"`
}

func (DailyStockSnapshot) TableName() string { return "daily_stock_snapshot" }
