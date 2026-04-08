package classify

import (
	"math"
	"time"
)

func CalcHistoryScore(lastSeenDate string, hitCount int, today string) float64 {
	if lastSeenDate == "" || today == "" {
		return historyScoreByHitCount(hitCount, 0.2)
	}

	lastSeen, err := time.Parse("2006-01-02", lastSeenDate)
	if err != nil {
		return historyScoreByHitCount(hitCount, 0.2)
	}
	todayTime, err := time.Parse("2006-01-02", today)
	if err != nil {
		return historyScoreByHitCount(hitCount, 0.2)
	}

	days := int(todayTime.Sub(lastSeen).Hours() / 24)
	if days < 0 {
		days = 0
	}
	if days <= 7 {
		return 1.0 - float64(days)/7.0
	}
	if days <= 14 {
		return historyScoreByHitCount(hitCount, 0.5)
	}
	return historyScoreByHitCount(hitCount, 0.2)
}

func IsRecentTopic(lastSeenDate string, today string) bool {
	if lastSeenDate == "" || today == "" {
		return false
	}

	lastSeen, err := time.Parse("2006-01-02", lastSeenDate)
	if err != nil {
		return false
	}
	todayTime, err := time.Parse("2006-01-02", today)
	if err != nil {
		return false
	}

	days := int(todayTime.Sub(lastSeen).Hours() / 24)
	if days < 0 {
		days = 0
	}
	return days <= 7
}

func CalcCombinedScore(heatScore, historyScore float64) float64 {
	return heatScore + historyScore
}

func historyScoreByHitCount(hitCount int, weight float64) float64 {
	if hitCount <= 0 {
		return 0
	}
	return math.Min(float64(hitCount)/10.0, 1.0) * weight
}
