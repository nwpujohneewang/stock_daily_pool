// model/api/response/pool.go
package response

import "stock/model/dal_model"

// PoolItem 池数据项
type PoolItem struct {
	TsCode            string  `json:"ts_code"`
	Name              string  `json:"name"`
	Price             float64 `json:"price"`
	PreClose          float64 `json:"pre_close"`
	ChangePct         float64 `json:"change_pct"`
	PoolType          int     `json:"pool_type"`
	LimitUpPrice      float64 `json:"limit_up_price,omitempty"`
	Date              string  `json:"date"`
	FirstTime         string  `json:"first_time,omitempty"`
	LastTime          string  `json:"last_time,omitempty"`
	LimitTimes        int     `json:"limit_times,omitempty"`
	LimitTimesDisplay string  `json:"limit_times_display,omitempty"`
}

// ReclassifyTopicRelation 话题关联（用于重分类结果）
type ReclassifyTopicRelation struct {
	TopicID    int64   `json:"topic_id"`
	TopicName  string  `json:"topic_name"`
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
	Layer      string  `json:"layer"`
}

// ReclassifyItem 重分类项
type ReclassifyItem struct {
	TsCode                string                    `json:"ts_code"`
	Name                  string                    `json:"name"`
	Price                 float64                   `json:"price"`
	PreClose              float64                   `json:"pre_close"`
	ChangePct             float64                   `json:"change_pct"`
	PoolType              int                       `json:"pool_type"`
	LimitUpPrice          float64                   `json:"limit_up_price,omitempty"`
	IsLimitUp             bool                      `json:"is_limit_up"`
	IsAbove5Pct           bool                      `json:"is_above_5pct"`
	BoardCode             string                    `json:"board_code"`
	Topics                []dal_model.TopicRelation `json:"topics,omitempty"`
	AllTopics             []dal_model.TopicRelation `json:"all_topics,omitempty"`
	ClassifyLayer         string                    `json:"classify_layer,omitempty"`
	Confidence            float64                   `json:"confidence,omitempty"`
	Skipped               bool                      `json:"skipped"`
	SkipReason            string                    `json:"skip_reason,omitempty"`
	FirstTime             string                    `json:"first_time,omitempty"`
	LastTime              string                    `json:"last_time,omitempty"`
	LimitTimes            int                       `json:"limit_times,omitempty"`
	LimitTimesDisplay     string                    `json:"limit_times_display,omitempty"`
	YesterdayChangePct    float64                   `json:"yesterday_change_pct,omitempty"`
	IsYesterdayStrong     bool                      `json:"is_yesterday_strong,omitempty"`
	ConsecutiveStrongDays int                       `json:"consecutive_strong_days,omitempty"`
	TotalMv               *float64                  `json:"total_mv,omitempty"`
	Vol                   *float64                  `json:"vol,omitempty"`
	Amount                *float64                  `json:"amount,omitempty"`
}

// ReclassifyResp 重分类响应
type ReclassifyResp struct {
	Items                []ReclassifyItem `json:"items"`
	YesterdayStrongItems []ReclassifyItem `json:"yesterday_strong_items"`
	IsTrading            bool             `json:"is_trading"`
}
