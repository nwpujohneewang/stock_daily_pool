package cache

import (
	"context"
	"fmt"
)

type AlertDedupCacheInterface interface {
	IsAlerted(ctx context.Context, date, tsCode string, topicID int64) (bool, error)
	SetAlerted(ctx context.Context, date, tsCode string, topicID int64) error
	GetAlertedTopics(ctx context.Context, date, tsCode string) ([]int64, error)
}

var _ AlertDedupCacheInterface = (*AlertDedupCacheImpl)(nil)

type AlertDedupCacheImpl struct{}

func NewAlertDedupCache() *AlertDedupCacheImpl {
	return &AlertDedupCacheImpl{}
}

func (c AlertDedupCacheImpl) IsAlerted(ctx context.Context, date, tsCode string, topicID int64) (bool, error) {
	key := "alerted:" + date
	alertKey := fmt.Sprintf("%s:%d", tsCode, topicID)
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(key); found {
		set := v.(map[string]struct{})
		_, exists := set[alertKey]
		return exists, nil
	}
	return false, nil
}

func (c AlertDedupCacheImpl) SetAlerted(ctx context.Context, date, tsCode string, topicID int64) error {
	key := "alerted:" + date
	alertKey := fmt.Sprintf("%s:%d", tsCode, topicID)
	Lock()
	defer Unlock()

	var set map[string]struct{}
	if v, found := Cache.Get(key); found {
		set = v.(map[string]struct{})
	} else {
		set = make(map[string]struct{})
	}
	set[alertKey] = struct{}{}
	Cache.Set(key, set, TTLUntilEndOfDay())
	return nil
}

func (c AlertDedupCacheImpl) GetAlertedTopics(ctx context.Context, date, tsCode string) ([]int64, error) {
	key := "alerted:" + date
	prefix := tsCode + ":"
	RLock()
	defer RUnlock()

	var topicIDs []int64
	if v, found := Cache.Get(key); found {
		set := v.(map[string]struct{})
		for alertKey := range set {
			// Check if alertKey starts with tsCode:
			if len(alertKey) > len(prefix) && alertKey[:len(prefix)] == prefix {
				var topicID int64
				fmt.Sscanf(alertKey[len(prefix):], "%d", &topicID)
				topicIDs = append(topicIDs, topicID)
			}
		}
	}
	return topicIDs, nil
}
