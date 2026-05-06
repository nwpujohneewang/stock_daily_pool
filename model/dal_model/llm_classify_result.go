package dal_model

import "time"

type LLMClassifyResult struct {
	ID        int64     `gorm:"column:id"         json:"id"`
	TsCode    string    `gorm:"column:ts_code"    json:"ts_code"`
	Date      time.Time `gorm:"column:date"       json:"date"`
	Topic     string    `gorm:"column:topic"      json:"topic"`
	Reason    string    `gorm:"column:reason"     json:"reason"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

func (LLMClassifyResult) TableName() string { return "llm_classify_results" }
