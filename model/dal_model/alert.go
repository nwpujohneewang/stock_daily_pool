package dal_model

import (
	"encoding/json"
	"time"
)

type StrategyAlert struct {
	ID           int64           `gorm:"column:id"            json:"id"`
	Date         time.Time       `gorm:"column:date"          json:"date"`
	TsCode       string          `gorm:"column:ts_code"       json:"ts_code"`
	StockName    string          `gorm:"column:stock_name"    json:"stock_name"`
	TopicID      *int64          `gorm:"column:topic_id"      json:"topic_id,omitempty"`
	TopicName    *string         `gorm:"column:topic_name"    json:"topic_name,omitempty"`
	AlertType    int16           `gorm:"column:alert_type"    json:"alert_type"`
	TriggerPrice *float64        `gorm:"column:trigger_price" json:"trigger_price,omitempty"`
	TriggerTime  *time.Time      `gorm:"column:trigger_time"  json:"trigger_time,omitempty"`
	PrevDayPct   *float64        `gorm:"column:prev_day_pct"  json:"prev_day_pct,omitempty"`
	ExtraInfo    json.RawMessage `gorm:"column:extra_info"    json:"extra_info,omitempty"`
	Notified     bool            `gorm:"column:notified"      json:"notified"`
	CreatedAt    time.Time       `gorm:"column:created_at"    json:"created_at"`
}

func (StrategyAlert) TableName() string { return "strategy_alerts" }

type AlertExtraInfo struct {
	Price        float64      `json:"price"`
	PreClose     float64      `json:"pre_close"`
	ChangePct    float64      `json:"change_pct"`
	Volume       int64        `json:"volume"`
	Amount       float64      `json:"amount"`
	Turnover     float64      `json:"turnover"`
	Bid          [][2]float64 `json:"bid"`
	Ask          [][2]float64 `json:"ask"`
	LimitUpPrice float64      `json:"limit_up_price"`
	BoardCode    string       `json:"board_code"`
}

const (
	AlertTypeStrategy2 = 1
)
