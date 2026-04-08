package cache

import (
	"context"
	"stock/model/dal_model"
)

const (
	topicDictionaryAllKey = "topic_dictionary:all"
	topicDictionaryMapKey = "topic_dictionary:map"
)

type TopicDictionaryCacheInterface interface {
	SetAll(ctx context.Context, dicts []dal_model.TopicDictionary) error
	GetAll(ctx context.Context) ([]dal_model.TopicDictionary, bool)
	ClearAll(ctx context.Context) error
	SetAllMap(ctx context.Context, dictMap map[string]dal_model.TopicDictionary) error
	GetAllMap(ctx context.Context) (map[string]dal_model.TopicDictionary, bool)
	ClearAllMap(ctx context.Context) error
}

type TopicDictionaryCacheImpl struct{}

func NewTopicDictionaryCache() *TopicDictionaryCacheImpl {
	return &TopicDictionaryCacheImpl{}
}

func (c TopicDictionaryCacheImpl) SetAll(ctx context.Context, dicts []dal_model.TopicDictionary) error {
	dictMap := make(map[string]dal_model.TopicDictionary, len(dicts))
	for _, dict := range dicts {
		dictMap[dict.RawTopicName] = dict
	}
	return c.SetAllMap(ctx, dictMap)
}

func (c TopicDictionaryCacheImpl) GetAll(ctx context.Context) ([]dal_model.TopicDictionary, bool) {
	if v, found := Cache.Get(topicDictionaryAllKey); found {
		if dictMap, ok := v.(map[string]dal_model.TopicDictionary); ok {
			result := make([]dal_model.TopicDictionary, 0, len(dictMap))
			for _, dict := range dictMap {
				result = append(result, dict)
			}
			return result, true
		}
	}
	return nil, false
}

func (c TopicDictionaryCacheImpl) ClearAll(ctx context.Context) error {
	Cache.Delete(topicDictionaryAllKey)
	return nil
}

func (c TopicDictionaryCacheImpl) SetAllMap(ctx context.Context, dictMap map[string]dal_model.TopicDictionary) error {
	copied := make(map[string]dal_model.TopicDictionary, len(dictMap))
	for k, v := range dictMap {
		copied[k] = v
	}
	Cache.Set(topicDictionaryMapKey, copied, TTLUntilEndOfDay())
	Cache.Set(topicDictionaryAllKey, copied, TTLUntilEndOfDay())
	return nil
}

func (c TopicDictionaryCacheImpl) GetAllMap(ctx context.Context) (map[string]dal_model.TopicDictionary, bool) {
	if v, found := Cache.Get(topicDictionaryMapKey); found {
		if dictMap, ok := v.(map[string]dal_model.TopicDictionary); ok {
			copied := make(map[string]dal_model.TopicDictionary, len(dictMap))
			for k, val := range dictMap {
				copied[k] = val
			}
			return copied, true
		}
	}
	return nil, false
}

func (c TopicDictionaryCacheImpl) ClearAllMap(ctx context.Context) error {
	Cache.Delete(topicDictionaryMapKey)
	return nil
}
