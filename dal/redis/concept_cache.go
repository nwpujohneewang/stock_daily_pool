package redis

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

type ConceptCacheInterface interface {
	GetStockConcepts(ctx context.Context, tsCode string) ([]string, error)
	SetStockConcepts(ctx context.Context, tsCode string, concepts []string) error
	GetConceptToTopics(ctx context.Context, conceptName string) ([]int64, error)
	SetConceptToTopics(ctx context.Context, conceptName string, topicIDs []int64) error
}

var _ ConceptCacheInterface = (*ConceptCacheImpl)(nil)

type ConceptCacheImpl struct{}

func NewConceptCache() *ConceptCacheImpl {
	return &ConceptCacheImpl{}
}

func (c ConceptCacheImpl) GetStockConcepts(ctx context.Context, tsCode string) ([]string, error) {
	key := "cache:stock_concepts"
	data, err := RedisClient(ctx).HGet(ctx, key, tsCode).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var concepts []string
	if err := json.Unmarshal([]byte(data), &concepts); err != nil {
		return nil, err
	}
	return concepts, nil
}

func (c ConceptCacheImpl) SetStockConcepts(ctx context.Context, tsCode string, concepts []string) error {
	key := "cache:stock_concepts"
	data, err := json.Marshal(concepts)
	if err != nil {
		return err
	}
	return RedisClient(ctx).HSet(ctx, key, tsCode, data).Err()
}

func (c ConceptCacheImpl) GetConceptToTopics(ctx context.Context, conceptName string) ([]int64, error) {
	key := "cache:concept_to_topics"
	data, err := RedisClient(ctx).HGet(ctx, key, conceptName).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var topicIDs []int64
	if err := json.Unmarshal([]byte(data), &topicIDs); err != nil {
		return nil, err
	}
	return topicIDs, nil
}

func (c ConceptCacheImpl) SetConceptToTopics(ctx context.Context, conceptName string, topicIDs []int64) error {
	key := "cache:concept_to_topics"
	data, err := json.Marshal(topicIDs)
	if err != nil {
		return err
	}
	return RedisClient(ctx).HSet(ctx, key, conceptName, data).Err()
}
