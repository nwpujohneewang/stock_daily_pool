package converter

import (
	"testing"
)

func TestJiuyanToTushare(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"深市", "sz000001", "000001.SZ"},
		{"沪市", "sh600000", "600000.SH"},
		{"科创板", "sh688001", "688001.SH"},
		{"北交所", "bj830799", "830799.BJ"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JiuyanToTushare(tt.input)
			if got != tt.want {
				t.Errorf("JiuyanToTushare(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTushareToJiuyan(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"深市", "000001.SZ", "sz000001"},
		{"沪市", "600000.SH", "sh600000"},
		{"科创板", "688001.SH", "sh688001"},
		{"北交所", "830799.BJ", "bj830799"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TushareToJiuyan(tt.input)
			if got != tt.want {
				t.Errorf("TushareToJiuyan(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDetectMarket(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
		wantOK bool
	}{
		{"沪市主板", "600000", "SH", true},
		{"深市主板", "000001", "SZ", true},
		{"创业板", "300001", "SZ", true},
		{"科创板", "688001", "SH", true},
		{"北交所", "830001", "BJ", true},
		{"无效", "xyz", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DetectMarket(tt.input)
			if (err == nil) != tt.wantOK {
				t.Errorf("DetectMarket(%q) error = %v, wantOK %v", tt.input, err, tt.wantOK)
				return
			}
			if got != tt.want {
				t.Errorf("DetectMarket(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSymbolToTsCode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"深市", "000001", "000001.SZ"},
		{"沪市", "600000", "600000.SH"},
		{"创业板", "300001", "300001.SZ"},
		{"科创板", "688001", "688001.SH"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SymbolToTsCode(tt.input)
			if err != nil {
				t.Errorf("SymbolToTsCode(%q) error = %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("SymbolToTsCode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
