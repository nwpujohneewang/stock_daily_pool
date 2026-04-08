package cache

import (
	"context"
	"fmt"
	"stock/model/dal_model"
)

type StockTopicsRelationCacheInterface interface {
	GetStockTopics(ctx context.Context, tsCode string) ([]dal_model.TopicRelation, error)
	GetStockTopicsBatch(ctx context.Context, tsCodes []string) (map[string][]dal_model.TopicRelation, error)
	SetStockTopics(ctx context.Context, tsCode string, mappings []dal_model.TopicRelation) error
	SetStockTopicsBatch(ctx context.Context, data map[string][]dal_model.TopicRelation) error
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
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(key); found {
		hashMap := v.(map[string][]dal_model.TopicRelation)
		if mappings, exists := hashMap[tsCode]; exists {
			// Return a copy
			result := make([]dal_model.TopicRelation, len(mappings))
			copy(result, mappings)
			return result, nil
		}
	}
	return nil, nil
}

func (c StockTopicsRelationCacheInterfaceImpl) GetStockTopicsBatch(ctx context.Context, tsCodes []string) (map[string][]dal_model.TopicRelation, error) {
	if len(tsCodes) == 0 {
		return nil, nil
	}
	key := "cache:stock_topics"
	RLock()
	defer RUnlock()

	result := make(map[string][]dal_model.TopicRelation, len(tsCodes))
	if v, found := Cache.Get(key); found {
		hashMap := v.(map[string][]dal_model.TopicRelation)
		for _, tc := range tsCodes {
			if mappings, exists := hashMap[tc]; exists {
				// Copy each mapping slice
				copied := make([]dal_model.TopicRelation, len(mappings))
				copy(copied, mappings)
				result[tc] = copied
			}
		}
	}
	return result, nil
}

func (c StockTopicsRelationCacheInterfaceImpl) SetStockTopics(ctx context.Context, tsCode string, mappings []dal_model.TopicRelation) error {
	key := "cache:stock_topics"
	Lock()
	defer Unlock()

	var hashMap map[string][]dal_model.TopicRelation
	if v, found := Cache.Get(key); found {
		hashMap = v.(map[string][]dal_model.TopicRelation)
	} else {
		hashMap = make(map[string][]dal_model.TopicRelation)
	}
	// Store a copy
	copied := make([]dal_model.TopicRelation, len(mappings))
	copy(copied, mappings)
	hashMap[tsCode] = copied
	Cache.Set(key, hashMap, TTLUntilEndOfDay())
	return nil
}

func (c StockTopicsRelationCacheInterfaceImpl) SetStockTopicsBatch(ctx context.Context, data map[string][]dal_model.TopicRelation) error {
	if len(data) == 0 {
		return nil
	}
	key := "cache:stock_topics"
	Lock()
	defer Unlock()

	var hashMap map[string][]dal_model.TopicRelation
	if v, found := Cache.Get(key); found {
		hashMap = v.(map[string][]dal_model.TopicRelation)
	} else {
		hashMap = make(map[string][]dal_model.TopicRelation)
	}
	for tsCode, mappings := range data {
		copied := make([]dal_model.TopicRelation, len(mappings))
		copy(copied, mappings)
		hashMap[tsCode] = copied
	}
	Cache.Set(key, hashMap, TTLUntilEndOfDay())
	return nil
}

func (c StockTopicsRelationCacheInterfaceImpl) GetBindStrength(ctx context.Context, tsCode string, topicID int64) (int, error) {
	key := "cache:bind_strength"
	field := fmt.Sprintf("%s:%d", tsCode, topicID)
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(key); found {
		hashMap := v.(map[string]int)
		if val, exists := hashMap[field]; exists {
			return val, nil
		}
	}
	return 0, nil
}

func (c StockTopicsRelationCacheInterfaceImpl) SetBindStrength(ctx context.Context, tsCode string, topicID int64, count int) error {
	key := "cache:bind_strength"
	field := fmt.Sprintf("%s:%d", tsCode, topicID)
	Lock()
	defer Unlock()

	var hashMap map[string]int
	if v, found := Cache.Get(key); found {
		hashMap = v.(map[string]int)
	} else {
		hashMap = make(map[string]int)
	}
	hashMap[field] = count
	Cache.Set(key, hashMap, TTLUntilEndOfDay())
	return nil
}

func (c StockTopicsRelationCacheInterfaceImpl) GetAllBindStrength(ctx context.Context, tsCode string) (map[int64]int, error) {
	key := "cache:bind_strength"
	prefix := tsCode + ":"
	RLock()
	defer RUnlock()

	result := make(map[int64]int)
	if v, found := Cache.Get(key); found {
		hashMap := v.(map[string]int)
		for field, val := range hashMap {
			// Check if field starts with tsCode:
			if len(field) > len(prefix) && field[:len(prefix)] == prefix {
				var topicID int64
				fmt.Sscanf(field[len(prefix):], "%d", &topicID)
				result[topicID] = val
			}
		}
	}
	return result, nil
}
