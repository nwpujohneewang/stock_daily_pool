package cache

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"stock/internal/model"
)

type MappingCache struct {
	client *redis.Client
}

func NewMappingCache(client *redis.Client) *MappingCache {
	return &MappingCache{client: client}
}

func (c *MappingCache) GetStockTopics(ctx context.Context, tsCode string) ([]model.TopicMapping, error) {
	key := "cache:stock_topics"
	data, err := c.client.HGet(ctx, key, tsCode).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var mappings []model.TopicMapping
	if err := json.Unmarshal([]byte(data), &mappings); err != nil {
		return nil, err
	}
	return mappings, nil
}

func (c *MappingCache) SetStockTopics(ctx context.Context, tsCode string, mappings []model.TopicMapping) error {
	key := "cache:stock_topics"
	data, err := json.Marshal(mappings)
	if err != nil {
		return err
	}
	return c.client.HSet(ctx, key, tsCode, data).Err()
}

func (c *MappingCache) GetBindStrength(ctx context.Context, tsCode string, topicID int64) (int, error) {
	key := "cache:bind_strength"
	field := fmt.Sprintf("%s:%d", tsCode, topicID)
	data, err := c.client.HGet(ctx, key, field).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return data, err
}

func (c *MappingCache) SetBindStrength(ctx context.Context, tsCode string, topicID int64, count int) error {
	key := "cache:bind_strength"
	field := fmt.Sprintf("%s:%d", tsCode, topicID)
	return c.client.HSet(ctx, key, field, count).Err()
}

func (c *MappingCache) GetAllBindStrength(ctx context.Context, tsCode string) (map[int64]int, error) {
	key := "cache:bind_strength"
	pattern := fmt.Sprintf("%s:*", tsCode)
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[int64]int)
	for _, k := range keys {
		var topicID int64
		fmt.Sscanf(k, "cache:bind_strength:%s:%d", &topicID)
		val, _ := c.client.HGet(ctx, key, k).Int()
		result[topicID] = val
	}
	return result, nil
}
