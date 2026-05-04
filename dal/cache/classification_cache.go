package cache

import (
	"context"
	"fmt"
	"stock/model/dal_model"
)

type ClassificationCacheInterface interface {
	GetClassificationResult(ctx context.Context, date string) (map[string][]dal_model.TopicRelation, bool, error)
	MergeClassificationResult(ctx context.Context, date string, results map[string][]dal_model.TopicRelation) error
}

var _ ClassificationCacheInterface = (*ClassificationCacheImpl)(nil)

type ClassificationCacheImpl struct{}

func NewClassificationCache() *ClassificationCacheImpl {
	return &ClassificationCacheImpl{}
}

func (c *ClassificationCacheImpl) GetClassificationResult(ctx context.Context, date string) (map[string][]dal_model.TopicRelation, bool, error) {
	key := fmt.Sprintf("cache:classification_result:%s", date)
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(key); found {
		hashMap := v.(map[string][]dal_model.TopicRelation)
		// Return a copy
		result := make(map[string][]dal_model.TopicRelation, len(hashMap))
		for tsCode, topics := range hashMap {
			copied := make([]dal_model.TopicRelation, len(topics))
			copy(copied, topics)
			result[tsCode] = copied
		}
		return result, true, nil
	}
	return nil, false, nil
}

func (c *ClassificationCacheImpl) MergeClassificationResult(ctx context.Context, date string, results map[string][]dal_model.TopicRelation) error {
	if len(results) == 0 {
		return nil
	}
	key := fmt.Sprintf("cache:classification_result:%s", date)
	Lock()
	defer Unlock()

	var hashMap map[string][]dal_model.TopicRelation
	if v, found := Cache.Get(key); found {
		hashMap = v.(map[string][]dal_model.TopicRelation)
	} else {
		hashMap = make(map[string][]dal_model.TopicRelation)
	}

	for tsCode, topics := range results {
		copied := make([]dal_model.TopicRelation, len(topics))
		copy(copied, topics)
		hashMap[tsCode] = copied
	}

	Cache.Set(key, hashMap, TTLUntilEndOfDay())
	return nil
}
