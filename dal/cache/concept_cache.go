package cache

import (
	"context"
)

type ConceptCacheInterface interface {
	GetStockConcepts(ctx context.Context, tsCode string) ([]string, error)
	GetStockConceptsBatch(ctx context.Context, tsCodes []string) (map[string][]string, error)
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
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(key); found {
		hashMap := v.(map[string][]string)
		if concepts, exists := hashMap[tsCode]; exists {
			// Return a copy
			result := make([]string, len(concepts))
			copy(result, concepts)
			return result, nil
		}
	}
	return nil, nil
}

func (c ConceptCacheImpl) GetStockConceptsBatch(ctx context.Context, tsCodes []string) (map[string][]string, error) {
	if len(tsCodes) == 0 {
		return nil, nil
	}
	key := "cache:stock_concepts"
	RLock()
	defer RUnlock()

	result := make(map[string][]string, len(tsCodes))
	if v, found := Cache.Get(key); found {
		hashMap := v.(map[string][]string)
		for _, tc := range tsCodes {
			if concepts, exists := hashMap[tc]; exists {
				// Copy each concept slice
				copied := make([]string, len(concepts))
				copy(copied, concepts)
				result[tc] = copied
			}
		}
	}
	return result, nil
}

func (c ConceptCacheImpl) SetStockConcepts(ctx context.Context, tsCode string, concepts []string) error {
	key := "cache:stock_concepts"
	Lock()
	defer Unlock()

	var hashMap map[string][]string
	if v, found := Cache.Get(key); found {
		hashMap = v.(map[string][]string)
	} else {
		hashMap = make(map[string][]string)
	}
	// Store a copy
	copied := make([]string, len(concepts))
	copy(copied, concepts)
	hashMap[tsCode] = copied
	Cache.Set(key, hashMap, TTLUntilEndOfDay())
	return nil
}

func (c ConceptCacheImpl) GetConceptToTopics(ctx context.Context, conceptName string) ([]int64, error) {
	key := "cache:concept_to_topics"
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(key); found {
		hashMap := v.(map[string][]int64)
		if topicIDs, exists := hashMap[conceptName]; exists {
			// Return a copy
			result := make([]int64, len(topicIDs))
			copy(result, topicIDs)
			return result, nil
		}
	}
	return nil, nil
}

func (c ConceptCacheImpl) SetConceptToTopics(ctx context.Context, conceptName string, topicIDs []int64) error {
	key := "cache:concept_to_topics"
	Lock()
	defer Unlock()

	var hashMap map[string][]int64
	if v, found := Cache.Get(key); found {
		hashMap = v.(map[string][]int64)
	} else {
		hashMap = make(map[string][]int64)
	}
	// Store a copy
	copied := make([]int64, len(topicIDs))
	copy(copied, topicIDs)
	hashMap[conceptName] = copied
	Cache.Set(key, hashMap, TTLUntilEndOfDay())
	return nil
}
