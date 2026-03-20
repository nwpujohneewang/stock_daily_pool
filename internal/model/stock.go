package model

import (
	"time"
)

type BoardCode string

const (
	BoardMain BoardCode = "MAIN"
	BoardGEM  BoardCode = "GEM"
	BoardSTAR BoardCode = "STAR"
	BoardBSE  BoardCode = "BSE"
)

type StockBasicInfo struct {
	ID        int64      `gorm:"column:id"         json:"id"`
	TsCode    string     `gorm:"column:ts_code"    json:"ts_code"`
	Symbol    string     `gorm:"column:symbol"     json:"symbol"`
	Name      string     `gorm:"column:name"       json:"name"`
	Exchange  string     `gorm:"column:exchange"   json:"exchange"`
	BoardCode string     `gorm:"column:board_code" json:"board_code"`
	Industry  *string    `gorm:"column:industry"   json:"industry,omitempty"`
	IsST      bool       `gorm:"column:is_st"      json:"is_st"`
	ListDate  *time.Time `gorm:"column:list_date"  json:"list_date,omitempty"`
	Status    int16      `gorm:"column:status"     json:"status"`
	CreatedAt time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (StockBasicInfo) TableName() string { return "stock_basic_info" }

type StockQuote struct {
	TsCode       string    `json:"ts_code"`
	PreClose     float64   `json:"pre_close"`
	Price        float64   `json:"price"`
	PctChg       float64   `json:"pct_chg"`
	Vol          int64     `json:"vol"`
	Amount       float64   `json:"amount"`
	TurnoverRate float64   `json:"turnover_rate"`
	UpdateTime   time.Time `json:"update_time"`
}

type BoardRule struct {
	BoardCode      BoardCode `json:"board_code"`
	BoardName      string    `json:"board_name"`
	LimitUpRatio   float64   `json:"limit_up_ratio"`
	LimitDownRatio float64   `json:"limit_down_ratio"`
	CodePatterns   []string  `json:"code_patterns"`
}

type DetectInput struct {
	TsCode       string    `json:"ts_code"`
	StockName    string    `json:"stock_name"`
	CurrentPrice float64   `json:"current_price"`
	PreClose     float64   `json:"pre_close"`
	ChangePct    float64   `json:"change_pct"`
	Volume       float64   `json:"volume"`
	BoardCode    BoardCode `json:"board_code"`
	IsST         bool      `json:"is_st"`
	QuoteTime    time.Time `json:"quote_time"`
	PrevState    int       `json:"prev_state"`
}

type DetectOutput struct {
	IsLimitUp      bool       `json:"is_limit_up"`
	IsAbove5Pct    bool       `json:"is_above_5pct"`
	IsFirstLimitUp bool       `json:"is_first_limit_up"`
	LimitUpPrice   float64    `json:"limit_up_price"`
	FirstLimitTime *time.Time `json:"first_limit_time"`
	CurrentState   int        `json:"current_state"`
	BoardSeq       int        `json:"board_seq"`
	Skipped        bool       `json:"skipped"`
	SkipReason     string     `json:"skip_reason"`
}

const (
	LimitStateNone     = 0
	LimitStateLimitUp  = 1
	LimitStateOpened   = 2
	LimitStateReSealed = 3
)

const (
	BoardSeqFirst  = 1
	BoardSeqSecond = 2
	BoardSeqThird  = 3
)
