package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type AlertDedupCache struct {
	client *redis.Client
}

func NewAlertDedupCache(client *redis.Client) *AlertDedupCache {
	return &AlertDedupCache{client: client}
}

func (c *AlertDedupCache) IsAlerted(ctx context.Context, date, tsCode string, topicID int64) (bool, error) {
	key := "alerted:" + date
	alertKey := fmt.Sprintf("%s:%d", tsCode, topicID)
	return c.client.SIsMember(ctx, key, alertKey).Result()
}

func (c *AlertDedupCache) SetAlerted(ctx context.Context, date, tsCode string, topicID int64) error {
	key := "alerted:" + date
	alertKey := fmt.Sprintf("%s:%d", tsCode, topicID)
	return c.client.SAdd(ctx, key, alertKey).Err()
}

func (c *AlertDedupCache) GetAlertedTopics(ctx context.Context, date, tsCode string) ([]int64, error) {
	pattern := fmt.Sprintf("%s:*", tsCode)
	keys, err := c.client.Keys(ctx, pattern).Result()
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
