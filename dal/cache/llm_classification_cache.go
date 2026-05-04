package cache

import (
	"context"
	"fmt"
	"stock/model/dal_model"
)

type LLMClassificationCacheInterface interface {
	GetClassificationResult(ctx context.Context, date string) (map[string][]dal_model.TopicRelation, bool, error)
	SetClassificationResult(ctx context.Context, date string, results map[string][]dal_model.TopicRelation) error
	MergeClassificationResult(ctx context.Context, date string, results map[string][]dal_model.TopicRelation) error
}

var _ LLMClassificationCacheInterface = (*LLMClassificationCacheImpl)(nil)

type LLMClassificationCacheImpl struct{}

func NewLLMClassificationCache() *LLMClassificationCacheImpl {
	return &LLMClassificationCacheImpl{}
}

func (c *LLMClassificationCacheImpl) GetClassificationResult(ctx context.Context, date string) (map[string][]dal_model.TopicRelation, bool, error) {
	key := fmt.Sprintf("cache:llm_classification_result:%s", date)
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(key); found {
		hashMap := v.(map[string][]dal_model.TopicRelation)
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

func (c *LLMClassificationCacheImpl) SetClassificationResult(ctx context.Context, date string, results map[string][]dal_model.TopicRelation) error {
	key := fmt.Sprintf("cache:llm_classification_result:%s", date)
	Lock()
	defer Unlock()

	hashMap := make(map[string][]dal_model.TopicRelation, len(results))
	for tsCode, topics := range results {
		copied := make([]dal_model.TopicRelation, len(topics))
		copy(copied, topics)
		hashMap[tsCode] = copied
	}
	Cache.Set(key, hashMap, TTLUntilEndOfDay())
	return nil
}

func (c *LLMClassificationCacheImpl) MergeClassificationResult(ctx context.Context, date string, results map[string][]dal_model.TopicRelation) error {
	if len(results) == 0 {
		return nil
	}
	key := fmt.Sprintf("cache:llm_classification_result:%s", date)
	Lock()
	defer Unlock()

	var hashMap map[string][]dal_model.TopicRelation
	if v, found := Cache.Get(key); found {
		hashMap = v.(map[string][]dal_model.TopicRelation)
	} else {
		hashMap = make(map[string][]dal_model.TopicRelation)
	}

	dirty := false
	for tsCode, topics := range results {
		copied := make([]dal_model.TopicRelation, len(topics))
		copy(copied, topics)
		hashMap[tsCode] = copied
		dirty = true
	}

	if dirty {
		Cache.Set(key, hashMap, TTLUntilEndOfDay())
	}
	return nil
}
