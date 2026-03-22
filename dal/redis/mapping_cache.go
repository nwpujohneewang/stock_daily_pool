package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"stock/model/dal_model"

	"github.com/redis/go-redis/v9"
)

type StockTopicsRelationCacheInterface interface {
	GetStockTopics(ctx context.Context, tsCode string) ([]dal_model.TopicRelation, error)
	GetStockTopicsBatch(ctx context.Context, tsCodes []string) (map[string][]dal_model.TopicRelation, error)
	SetStockTopics(ctx context.Context, tsCode string, mappings []dal_model.TopicRelation) error
	GetBindStrength(ctx context.Context, tsCode string, topicID int64) (int, error)
	SetBindStrength(ctx context.Context, tsCode string, topicID int64, count int) error
	GetAllBindStrength(ctx context.Context, tsCode string) (map[int64]int, error)
}

var _ StockTopicsRelationCacheInterface = (*StockTopicsRelationCacheInterfaceImpl)(nil)

type StockTopicsRelationCacheInterfaceImpl struct{}

func NewStockTopicsRelationCache() *StockTopicsRelationCacheInterfaceImpl {
	return &StockTopicsRelationCacheInterfaceImpl{}
}

func (c StockTopicsRelationCacheInterfaceImpl) GetStockTopics(ctx context.Context, tsCode string) ([]dal_model.TopicRelation, error) {
	key := "cache:stock_topics"
	data, err := RedisClient(ctx).HGet(ctx, key, tsCode).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var mappings []dal_model.TopicRelation
	if err := json.Unmarshal([]byte(data), &mappings); err != nil {
		return nil, err
	}
	return mappings, nil
}

func (c StockTopicsRelationCacheInterfaceImpl) GetStockTopicsBatch(ctx context.Context, tsCodes []string) (map[string][]dal_model.TopicRelation, error) {
	if len(tsCodes) == 0 {
		return nil, nil
	}
	key := "cache:stock_topics"
	pipe := RedisClient(ctx).Pipeline()
	cmds := make([]*redis.StringCmd, len(tsCodes))
	for i, tc := range tsCodes {
		cmds[i] = pipe.HGet(ctx, key, tc)
	}
	pipe.Exec(ctx)

	result := make(map[string][]dal_model.TopicRelation, len(tsCodes))
	for i, tc := range tsCodes {
		data, err := cmds[i].Result()
		if err != nil || data == "" {
			continue
		}
		var mappings []dal_model.TopicRelation
		if err := json.Unmarshal([]byte(data), &mappings); err == nil {
			result[tc] = mappings
		}
	}
	return result, nil
}

func (c StockTopicsRelationCacheInterfaceImpl) SetStockTopics(ctx context.Context, tsCode string, mappings []dal_model.TopicRelation) error {
	key := "cache:stock_topics"
	data, err := json.Marshal(mappings)
	if err != nil {
		return err
	}
	return RedisClient(ctx).HSet(ctx, key, tsCode, data).Err()
}

func (c StockTopicsRelationCacheInterfaceImpl) GetBindStrength(ctx context.Context, tsCode string, topicID int64) (int, error) {
	key := "cache:bind_strength"
	field := fmt.Sprintf("%s:%d", tsCode, topicID)
	data, err := RedisClient(ctx).HGet(ctx, key, field).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return data, err
}

func (c StockTopicsRelationCacheInterfaceImpl) SetBindStrength(ctx context.Context, tsCode string, topicID int64, count int) error {
	key := "cache:bind_strength"
	field := fmt.Sprintf("%s:%d", tsCode, topicID)
	return RedisClient(ctx).HSet(ctx, key, field, count).Err()
}

func (c StockTopicsRelationCacheInterfaceImpl) GetAllBindStrength(ctx context.Context, tsCode string) (map[int64]int, error) {
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
