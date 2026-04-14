package classify

import (
	"time"
)

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
