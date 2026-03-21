package dal_model

import (
	"encoding/json"
	"time"
)

type FocusTopic struct {
	ID        int64     `gorm:"column:id"         json:"id"`
	Date      time.Time `gorm:"column:date"       json:"date"`
	TopicID   int64     `gorm:"column:topic_id"   json:"topic_id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

func (FocusTopic) TableName() string { return "focus_topics" }

type DailyStockPool struct {
	ID             int64           `gorm:"column:id"               json:"id"`
	Date           time.Time       `gorm:"column:date"             json:"date"`
	TsCode         string          `gorm:"column:ts_code"          json:"ts_code"`
	StockName      string          `gorm:"column:stock_name"       json:"stock_name"`
	PoolType       int16           `gorm:"column:pool_type"        json:"pool_type"`
	ChangePct      *float64        `gorm:"column:change_pct"       json:"change_pct,omitempty"`
	CurrentPrice   *float64        `gorm:"column:current_price"    json:"current_price,omitempty"`
	PreClose       *float64        `gorm:"column:pre_close"        json:"pre_close,omitempty"`
	LimitUpPrice   *float64        `gorm:"column:limit_up_price"   json:"limit_up_price,omitempty"`
	FirstLimitTime *string         `gorm:"column:first_limit_time" json:"first_limit_time,omitempty"`
	BoardCode      *string         `gorm:"column:board_code"       json:"board_code,omitempty"`
	TopicIDs       json.RawMessage `gorm:"column:topic_ids"         json:"topic_ids,omitempty"`
	Vol            *float64        `gorm:"column:vol"               json:"vol,omitempty"`
	Amount         *float64        `gorm:"column:amount"           json:"amount,omitempty"`
	SnapshotTime   *time.Time      `gorm:"column:snapshot_time"    json:"snapshot_time,omitempty"`
	CreatedAt      time.Time       `gorm:"column:created_at"       json:"created_at"`
}

func (DailyStockPool) TableName() string { return "daily_stock_pool" }

const (
	PoolTypeLimitUp = 1
	PoolTypeAbove5  = 2
)
