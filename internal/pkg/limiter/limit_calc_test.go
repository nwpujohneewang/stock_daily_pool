package limiter

import (
	"stock/model/dal_model"
	"testing"
)

func TestCalcLimitUpPrice(t *testing.T) {
	tests := []struct {
		name     string
		preClose float64
		ratio    float64
		want     float64
	}{
		{"主板10%", 10.00, 0.10, 11.00},
		{"主板低于1元", 0.50, 0.10, 0.55},
		{"科创板20%", 50.00, 0.20, 60.00},
		{"北交所30%", 5.00, 0.30, 6.50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcLimitUpPrice(tt.preClose, tt.ratio)
			if got != tt.want {
				t.Errorf("CalcLimitUpPrice(%v, %v) = %v, want %v", tt.preClose, tt.ratio, got, tt.want)
			}
		})
	}
}

func TestCalcLimitUpPrice_InvalidInput(t *testing.T) {
	tests := []struct {
		name     string
		preClose float64
		ratio    float64
	}{
		{"零价格", 0, 0.10},
		{"负价格", -10, 0.10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcLimitUpPrice(tt.preClose, tt.ratio)
			if got != 0 {
				t.Errorf("CalcLimitUpPrice(%v, %v) = %v, want 0", tt.preClose, tt.ratio, got)
			}
		})
	}
}

func TestDetectLimitUp(t *testing.T) {
	rule := &dal_model.BoardRule{
		BoardCode:      dal_model.BoardMain,
		LimitUpRatio:   0.10,
		LimitDownRatio: -0.10,
	}

	tests := []struct {
		name          string
		input         dal_model.DetectInput
		wantIsLimitUp bool
		wantIsAbove5  bool
	}{
		{
			name: "涨停",
			input: dal_model.DetectInput{
				TsCode:       "000001.SZ",
				CurrentPrice: 11.00,
				PreClose:     10.00,
				ChangePct:    10.0,
				IsST:         false,
				PrevState:    dal_model.LimitStateNone,
			},
			wantIsLimitUp: true,
			wantIsAbove5:  false,
		},
		{
			name: "涨幅5%以上但未涨停",
			input: dal_model.DetectInput{
				TsCode:       "000001.SZ",
				CurrentPrice: 10.50,
				PreClose:     10.00,
				ChangePct:    5.0,
				IsST:         false,
				PrevState:    dal_model.LimitStateNone,
			},
			wantIsLimitUp: false,
			wantIsAbove5:  true,
		},
		{
			name: "ST股票跳过",
			input: dal_model.DetectInput{
				TsCode:       "000001.SZ",
				CurrentPrice: 11.00,
				PreClose:     10.00,
				ChangePct:    10.0,
				IsST:         true,
				PrevState:    dal_model.LimitStateNone,
			},
			wantIsLimitUp: false,
			wantIsAbove5:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectLimitUp(tt.input, rule)
			if got.IsLimitUp != tt.wantIsLimitUp {
				t.Errorf("IsLimitUp = %v, want %v", got.IsLimitUp, tt.wantIsLimitUp)
			}
			if got.IsAbove5Pct != tt.wantIsAbove5 {
				t.Errorf("IsAbove5Pct = %v, want %v", got.IsAbove5Pct, tt.wantIsAbove5)
			}
		})
	}
}
