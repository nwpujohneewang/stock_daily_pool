package utils

import "strings"

var nonThemeKeywords = []string{
	// 财务业绩类
	"预增", "预减", "扭亏", "年报", "季报",
	// 短线行为类
	"昨日", "首板", "连板", "涨停", "炸板", "触板", "高换手", "高振幅", "多板", "近期",
	// 风格/市值/价格特征类
	"微盘", "小盘", "中盘", "大盘",
	"百元股", "低价股", "新高", "破净", "破发",
	"红利", "价值", "成长", "权重", "周期",
	// 资金/持仓类
	"重仓", "融资融券", "沪股通", "深股通", "证金", "养老金", "转债标的", "QFII", "机构", "参股", "茅指数",
	// 特殊状态类
	"ST", "超跌", "举牌",
	// 资本运作/事件类
	"并购重组", "股权激励", "股权转让", "IPO受益", "创投", "独角兽",
	// 平台标签/组合类
	"热股", "组合", "指数",
	// 指数/样本池类
	"MSCI", "富时罗素", "标准普尔", "创业板", "科创板", "做市",
	"AH股", "AB股", "B股", "GDR", "中字头",
	// 常见数字指数标签
	"50", "180", "300", "380", "500",
}

// 是否属于“非题材概念”
func isNonThemeConcept(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for _, kw := range nonThemeKeywords {
		if strings.Contains(name, kw) {
			return true
		}
	}
	return false
}

// 过滤出非题材概念
func FilterNonThemeConcepts(input map[string]float64) map[string]float64 {
	result := make(map[string]float64, len(input))
	for name, pct := range input {
		if isNonThemeConcept(name) {
			result[name] = pct
		}
	}
	return result
}

func FilterOverZeroAndNonThemeConcepts(input map[string]float64) map[string]float64 {
	result := make(map[string]float64, len(input))
	for name, pct := range input {
		if pct < 0 {
			continue
		}
		if isNonThemeConcept(name) {
			continue
		}
		result[name] = pct
	}
	return result
}
