package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type ActivityCache struct {
	client *redis.Client
}

func NewActivityCache(client *redis.Client) *ActivityCache {
	return &ActivityCache{client: client}
}

func (c *ActivityCache) IncrLimitCount(ctx context.Context, date string, topicID int64) error {
	key := fmt.Sprintf("topic:activity:limit:%s", date)
	return c.client.ZIncrBy(ctx, key, 1, fmt.Sprintf("%d", topicID)).Err()
}

func (c *ActivityCache) DecrLimitCount(ctx context.Context, date string, topicID int64) error {
	key := fmt.Sprintf("topic:activity:limit:%s", date)
	return c.client.ZIncrBy(ctx, key, -1, fmt.Sprintf("%d", topicID)).Err()
}

func (c *ActivityCache) GetTopicLimitCount(ctx context.Context, date string, topicID int64) (int, error) {
	key := fmt.Sprintf("topic:activity:limit:%s", date)
	score, err := c.client.ZScore(ctx, key, fmt.Sprintf("%d", topicID)).Result()
	if err == redis.Nil {
		return 0, nil
	}
	return int(score), err
}

func (c *ActivityCache) GetAllTopicLimitCounts(ctx context.Context, date string) (map[int64]int, error) {
	key := fmt.Sprintf("topic:activity:limit:%s", date)
	results, err := c.client.ZRangeWithScores(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	m := make(map[int64]int)
	for _, z := range results {
		var topicID int64
		fmt.Sscanf(z.Member.(string), "%d", &topicID)
		m[topicID] = int(z.Score)
	}
	return m, nil
}

func (c *ActivityCache) AddLimitTime(ctx context.Context, date string, topicID int64, tsCode, limitTime string) error {
	key := fmt.Sprintf("topic:limit_times:%s:%d", date, topicID)
	data := fmt.Sprintf(`{"ts_code":"%s","time":"%s"}`, tsCode, limitTime)
	return c.client.RPush(ctx, key, data).Err()
}

func (c *ActivityCache) GetLimitTimes(ctx context.Context, date string, topicID int64) ([]string, error) {
	key := fmt.Sprintf("topic:limit_times:%s:%d", date, topicID)
	return c.client.LRange(ctx, key, 0, -1).Result()
}
