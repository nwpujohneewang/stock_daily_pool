package utils

import "time"

// Hardcoded holiday calendar (A-share). This is intentionally minimal and can be extended.
// Dates are in Asia/Shanghai and formatted as YYYY-MM-DD.
var holidayDates = map[string]struct{}{
	"2026-01-01": {},
	"2026-01-02": {},
	"2026-01-03": {},
	// 春节：2月15日(腊月二十八)至23日(正月初七)
	"2026-02-15": {},
	"2026-02-16": {},
	"2026-02-17": {},
	"2026-02-18": {},
	"2026-02-19": {},
	"2026-02-20": {},
	"2026-02-21": {},
	"2026-02-22": {},
	"2026-02-23": {},
	// 清明节：4月4日至6日
	"2026-04-04": {},
	"2026-04-05": {},
	"2026-04-06": {},
	// 劳动节：5月1日至5日
	"2026-05-01": {},
	"2026-05-02": {},
	"2026-05-03": {},
	"2026-05-04": {},
	"2026-05-05": {},
	// 端午节：6月19日至21日
	"2026-06-19": {},
	"2026-06-20": {},
	"2026-06-21": {},
	// 中秋节：9月25日至27日
	"2026-09-25": {},
	"2026-09-26": {},
	"2026-09-27": {},
	// 国庆节：10月1日至7日
	"2026-10-01": {},
	"2026-10-02": {},
	"2026-10-03": {},
	"2026-10-04": {},
	"2026-10-05": {},
	"2026-10-06": {},
	"2026-10-07": {},
}

func IsHoliday(t time.Time) bool {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	date := t.In(loc).Format("2006-01-02")
	_, ok := holidayDates[date]
	return ok
}

func IsTradingDay(t time.Time) bool {
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return false
	}
	if IsHoliday(t) {
		return false
	}
	return true
}

func PreviousTradingDay(t time.Time) time.Time {
	prev := t.AddDate(0, 0, -1)
	for !IsTradingDay(prev) {
		prev = prev.AddDate(0, 0, -1)
	}
	return prev
}
