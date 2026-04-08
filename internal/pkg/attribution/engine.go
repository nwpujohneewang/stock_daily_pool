package attribution

import (
	"context"
	"encoding/json"
	"stock/dal/cache"
	"time"
)

// CalcActivityScore 计算 S1：热点活跃度
func CalcActivityScore(topicLimitCount int, maxLimitCountAll int) float64 {
	if maxLimitCountAll <= 0 {
		return 0.0
	}
	return float64(topicLimitCount) / float64(maxLimitCountAll)
}

// CalcBindStrengthScore 计算 S2：关联强度
func CalcBindStrengthScore(hitCount int, totalHitCount int) float64 {
	if totalHitCount <= 0 {
		return 0.0
	}
	return float64(hitCount) / float64(totalHitCount)
}

// CalcTimeProximityScore 计算 S3：时间接近度
func CalcTimeProximityScore(stockLimitTime time.Time, topicLimitTimes []time.Time) float64 {
	var before []time.Time
	for _, t := range topicLimitTimes {
		if t.Before(stockLimitTime) {
			before = append(before, t)
		}
	}
	if len(before) == 0 {
		return 0.5
	}
	var sum float64
	for _, t := range before {
		diff := stockLimitTime.Sub(t).Minutes()
		if diff < 0 {
			diff = 0
		}
		sum += diff
	}
	avg := sum / float64(len(before))
	return 1.0 / (1.0 + avg/15.0)
}

const recencyHalfLifeDays = 30
const staleThresholdDays = 15

func CalcRecencyScore(lastSeenDate time.Time, today time.Time) float64 {
	if lastSeenDate.IsZero() {
		return 0.1
	}
	days := int(today.Sub(lastSeenDate).Hours() / 24)
	if days < 0 {
		days = 0
	}
	if days > staleThresholdDays {
		return 0.0
	}
	return 1.0 / (1.0 + float64(days)/recencyHalfLifeDays)
}

// getTopicActivityMap 从 Redis 读取当天热点活跃度。
func getTopicActivityMap(ctx context.Context, date string) (map[int]int, error) {
	activityCache := cache.NewActivityCache()
	result, err := activityCache.GetAllTopicLimitCounts(ctx, date)
	if err != nil {
		return nil, err
	}
	m := make(map[int]int, len(result))
	for k, v := range result {
		m[int(k)] = v
	}
	return m, nil
}

type limitTimeRecord struct {
	TsCode string `json:"ts_code"`
	Time   string `json:"time"`
}

// getTopicLimitTimes 获取某日某热点的涨停时间序列。
func getTopicLimitTimes(ctx context.Context, date string, topicID int64) ([]time.Time, error) {
	activityCache := cache.NewActivityCache()
	rawTimes, err := activityCache.GetLimitTimes(ctx, date, topicID)
	if err != nil {
		return nil, err
	}
	var times []time.Time
	for _, raw := range rawTimes {
		var rec limitTimeRecord
		if err := decodeJSON(raw, &rec); err != nil {
			continue
		}
		t, err := time.Parse("15:04:05", rec.Time)
		if err != nil {
			continue
		}
		times = append(times, t)
	}
	return times, nil
}

func decodeJSON(raw string, v interface{}) error {
	// using stock/common/jsonhelper if available, fallback to stdlib
	return json.Unmarshal([]byte(raw), v)
}
