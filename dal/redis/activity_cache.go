package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
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
	return RedisClient(ctx).ZIncrBy(ctx, key, 1, fmt.Sprintf("%d", topicID)).Err()
}

func (c ActivityCacheImpl) DecrLimitCount(ctx context.Context, date string, topicID int64) error {
	key := fmt.Sprintf("topic:activity:limit:%s", date)
	return RedisClient(ctx).ZIncrBy(ctx, key, -1, fmt.Sprintf("%d", topicID)).Err()
}

func (c ActivityCacheImpl) GetTopicLimitCount(ctx context.Context, date string, topicID int64) (int, error) {
	key := fmt.Sprintf("topic:activity:limit:%s", date)
	score, err := RedisClient(ctx).ZScore(ctx, key, fmt.Sprintf("%d", topicID)).Result()
	if err == redis.Nil {
		return 0, nil
	}
	return int(score), err
}

func (c ActivityCacheImpl) GetAllTopicLimitCounts(ctx context.Context, date string) (map[int64]int, error) {
	key := fmt.Sprintf("topic:activity:limit:%s", date)
	results, err := RedisClient(ctx).ZRangeWithScores(ctx, key, 0, -1).Result()
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

func (c ActivityCacheImpl) AddLimitTime(ctx context.Context, date string, topicID int64, tsCode, limitTime string) error {
	key := fmt.Sprintf("topic:limit_times:%s:%d", date, topicID)
	data := fmt.Sprintf(`{"ts_code":"%s","time":"%s"}`, tsCode, limitTime)
	return RedisClient(ctx).RPush(ctx, key, data).Err()
}

func (c ActivityCacheImpl) GetLimitTimes(ctx context.Context, date string, topicID int64) ([]string, error) {
	key := fmt.Sprintf("topic:limit_times:%s:%d", date, topicID)
	return RedisClient(ctx).LRange(ctx, key, 0, -1).Result()
}
