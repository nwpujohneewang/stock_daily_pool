// model/api/response/alert.go
package response

import "time"

// AlertItem 告警项
type AlertItem struct {
	ID           int64      `json:"id"`
	Date         time.Time  `json:"date"`
	TsCode       string     `json:"ts_code"`
	StockName    string     `json:"stock_name"`
	TopicID      *int64     `json:"topic_id,omitempty"`
	TopicName    *string    `json:"topic_name,omitempty"`
	AlertType    int16      `json:"alert_type"`
	TriggerPrice *float64   `json:"trigger_price,omitempty"`
	TriggerTime  *time.Time `json:"trigger_time,omitempty"`
	Notified     bool       `json:"notified"`
	CreatedAt    time.Time  `json:"created_at"`
}
