package redis

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
	return RedisClient(ctx).SIsMember(ctx, key, alertKey).Result()
}

func (c AlertDedupCacheImpl) SetAlerted(ctx context.Context, date, tsCode string, topicID int64) error {
	key := "alerted:" + date
	alertKey := fmt.Sprintf("%s:%d", tsCode, topicID)
	return RedisClient(ctx).SAdd(ctx, key, alertKey).Err()
}

func (c AlertDedupCacheImpl) GetAlertedTopics(ctx context.Context, date, tsCode string) ([]int64, error) {
	pattern := fmt.Sprintf("%s:*", tsCode)
	keys, err := RedisClient(ctx).Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	var topicIDs []int64
	for _, k := range keys {
		var topicID int64
		fmt.Sscanf(k, "alerted:%s:%d", &topicID)
		topicIDs = append(topicIDs, topicID)
	}
	return topicIDs, nil
}
