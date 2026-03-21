package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"stock/model/dal_model"

	"github.com/redis/go-redis/v9"
)

type MappingCacheInterface interface {
	GetStockTopics(ctx context.Context, tsCode string) ([]dal_model.TopicMapping, error)
	SetStockTopics(ctx context.Context, tsCode string, mappings []dal_model.TopicMapping) error
	GetBindStrength(ctx context.Context, tsCode string, topicID int64) (int, error)
	SetBindStrength(ctx context.Context, tsCode string, topicID int64, count int) error
	GetAllBindStrength(ctx context.Context, tsCode string) (map[int64]int, error)
}

var _ MappingCacheInterface = (*MappingCacheImpl)(nil)

type MappingCacheImpl struct{}

func NewMappingCache() *MappingCacheImpl {
	return &MappingCacheImpl{}
}

func (c MappingCacheImpl) GetStockTopics(ctx context.Context, tsCode string) ([]dal_model.TopicMapping, error) {
	key := "cache:stock_topics"
	data, err := RedisClient(ctx).HGet(ctx, key, tsCode).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var mappings []dal_model.TopicMapping
	if err := json.Unmarshal([]byte(data), &mappings); err != nil {
		return nil, err
	}
	return mappings, nil
}

func (c MappingCacheImpl) SetStockTopics(ctx context.Context, tsCode string, mappings []dal_model.TopicMapping) error {
	key := "cache:stock_topics"
	data, err := json.Marshal(mappings)
	if err != nil {
		return err
	}
	return RedisClient(ctx).HSet(ctx, key, tsCode, data).Err()
}

func (c MappingCacheImpl) GetBindStrength(ctx context.Context, tsCode string, topicID int64) (int, error) {
	key := "cache:bind_strength"
	field := fmt.Sprintf("%s:%d", tsCode, topicID)
	data, err := RedisClient(ctx).HGet(ctx, key, field).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return data, err
}

func (c MappingCacheImpl) SetBindStrength(ctx context.Context, tsCode string, topicID int64, count int) error {
	key := "cache:bind_strength"
	field := fmt.Sprintf("%s:%d", tsCode, topicID)
	return RedisClient(ctx).HSet(ctx, key, field, count).Err()
}

func (c MappingCacheImpl) GetAllBindStrength(ctx context.Context, tsCode string) (map[int64]int, error) {
	key := "cache:bind_strength"
	pattern := fmt.Sprintf("%s:*", tsCode)
	keys, err := RedisClient(ctx).Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[int64]int)
	for _, k := range keys {
		var topicID int64
		fmt.Sscanf(k, "cache:bind_strength:%s:%d", &topicID)
		val, _ := RedisClient(ctx).HGet(ctx, key, k).Int()
		result[topicID] = val
	}
	return result, nil
}
