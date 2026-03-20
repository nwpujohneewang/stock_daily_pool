package cache

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

type ConceptCache struct {
	client *redis.Client
}

func NewConceptCache(client *redis.Client) *ConceptCache {
	return &ConceptCache{client: client}
}

func (c *ConceptCache) GetStockConcepts(ctx context.Context, tsCode string) ([]string, error) {
	key := "cache:stock_concepts"
	data, err := c.client.HGet(ctx, key, tsCode).Result()
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

func (c *ConceptCache) SetStockConcepts(ctx context.Context, tsCode string, concepts []string) error {
	key := "cache:stock_concepts"
	data, err := json.Marshal(concepts)
	if err != nil {
		return err
	}
	return c.client.HSet(ctx, key, tsCode, data).Err()
}

func (c *ConceptCache) GetConceptToTopics(ctx context.Context, conceptName string) ([]int64, error) {
	key := "cache:concept_to_topics"
	data, err := c.client.HGet(ctx, key, conceptName).Result()
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

func (c *ConceptCache) SetConceptToTopics(ctx context.Context, conceptName string, topicIDs []int64) error {
	key := "cache:concept_to_topics"
	data, err := json.Marshal(topicIDs)
	if err != nil {
		return err
	}
	return c.client.HSet(ctx, key, conceptName, data).Err()
}
