// internal/service/pool_types.go
package pool

import (
	"errors"
	"stock/model/dal_model"
)

// ErrDataNotReady indicates realtime data is not yet available
var ErrDataNotReady = errors.New("realtime data not ready")
var ErrHistoricalSnapshotNotGenerated = errors.New("该日期未生成分类快照")

// PoolQueryParams 池查询参数
type PoolQueryParams struct {
	Date     string
	PoolType int // 1=涨停, 2=涨幅超5%
}

// PoolItemResult 池数据结果
type PoolItemResult struct {
	TsCode       string
	Name         string
	Price        float64
	PreClose     float64
	ChangePct    float64
	PoolType     int16
	LimitUpPrice float64
	Date         string
}

// ReclassifyItemResult 重分类结果项
type ReclassifyItemResult struct {
	TsCode                string
	Name                  string
	Price                 float64
	PreClose              float64
	ChangePct             float64
	PoolType              int
	LimitUpPrice          float64
	IsLimitUp             bool
	IsAbove5Pct           bool
	BoardCode             string
	Topics                []dal_model.TopicRelation
	ClassifyLayer         string
	Confidence            float64
	Skipped               bool
	SkipReason            string
	YesterdayChangePct    float64
	IsYesterdayStrong     bool
	ConsecutiveStrongDays int16
	LimitTimes            int16
	TotalMv               *float64
	Vol                   *float64
	Amount                *float64
}

// ReclassifyResult 重分类结果
type ReclassifyResult struct {
	Items                []ReclassifyItemResult
	YesterdayStrongItems []ReclassifyItemResult
	IsTrading            bool
}
