package cache

import (
	"context"
	"fmt"
)

type ActivityCacheInterface interface {
	IncrLimitCount(ctx context.Context, date string, topicID int64) error
	DecrLimitCount(ctx context.Context, date string, topicID int64) error
	GetTopicLimitCount(ctx context.Context, date string, topicID int64) (int, error)
	GetAllTopicLimitCounts(ctx context.Context, date string) (map[int64]int, error)
	AddLimitTime(ctx context.Context, date string, topicID int64, tsCode, limitTime string) error
	GetLimitTimes(ctx context.Context, date string, topicID int64) ([]string, error)
}

var _ ActivityCacheInterface = (*ActivityCacheImpl)(nil)

type ActivityCacheImpl struct{}

func NewActivityCache() *ActivityCacheImpl {
	return &ActivityCacheImpl{}
}

func (c ActivityCacheImpl) IncrLimitCount(ctx context.Context, date string, topicID int64) error {
	key := fmt.Sprintf("topic:activity:limit:%s", date)
	Lock()
	defer Unlock()

	var counts map[int64]int
	if v, found := Cache.Get(key); found {
		counts = v.(map[int64]int)
	} else {
		counts = make(map[int64]int)
	}
	counts[topicID]++
	Cache.Set(key, counts, TTLUntilEndOfDay())
	return nil
}

func (c ActivityCacheImpl) DecrLimitCount(ctx context.Context, date string, topicID int64) error {
	key := fmt.Sprintf("topic:activity:limit:%s", date)
	Lock()
	defer Unlock()

	var counts map[int64]int
	if v, found := Cache.Get(key); found {
		counts = v.(map[int64]int)
	} else {
		counts = make(map[int64]int)
	}
	counts[topicID]--
	Cache.Set(key, counts, TTLUntilEndOfDay())
	return nil
}

func (c ActivityCacheImpl) GetTopicLimitCount(ctx context.Context, date string, topicID int64) (int, error) {
	key := fmt.Sprintf("topic:activity:limit:%s", date)
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(key); found {
		counts := v.(map[int64]int)
		return counts[topicID], nil
	}
	return 0, nil
}

func (c ActivityCacheImpl) GetAllTopicLimitCounts(ctx context.Context, date string) (map[int64]int, error) {
	key := fmt.Sprintf("topic:activity:limit:%s", date)
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(key); found {
		counts := v.(map[int64]int)
		// Return a copy to avoid concurrent modification
		result := make(map[int64]int, len(counts))
		for k, v := range counts {
			result[k] = v
		}
		return result, nil
	}
	return make(map[int64]int), nil
}

func (c ActivityCacheImpl) AddLimitTime(ctx context.Context, date string, topicID int64, tsCode, limitTime string) error {
	key := fmt.Sprintf("topic:limit_times:%s:%d", date, topicID)
	data := fmt.Sprintf(`{"ts_code":"%s","time":"%s"}`, tsCode, limitTime)

	Lock()
	defer Unlock()

	var list []string
	if v, found := Cache.Get(key); found {
		list = v.([]string)
	}
	list = append(list, data)
	Cache.Set(key, list, TTLUntilEndOfDay())
	return nil
}

func (c ActivityCacheImpl) GetLimitTimes(ctx context.Context, date string, topicID int64) ([]string, error) {
	key := fmt.Sprintf("topic:limit_times:%s:%d", date, topicID)
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(key); found {
		list := v.([]string)
		// Return a copy
		result := make([]string, len(list))
		copy(result, list)
		return result, nil
	}
	return []string{}, nil
}
