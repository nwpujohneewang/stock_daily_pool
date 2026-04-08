package cache

import (
	"context"
	"fmt"

	"stock/model/dal_model"
)

type TopicHeatCacheInterface interface {
	Get(ctx context.Context, date string, topicID int64) (*dal_model.TopicHeatInfo, error)
	GetAll(ctx context.Context, date string) (map[int64]*dal_model.TopicHeatInfo, error)
	SetAll(ctx context.Context, date string, data map[int64]*dal_model.TopicHeatInfo) error
}

type TopicHeatCacheImpl struct{}

func NewTopicHeatCache() *TopicHeatCacheImpl {
	return &TopicHeatCacheImpl{}
}

func (c TopicHeatCacheImpl) Get(ctx context.Context, date string, topicID int64) (*dal_model.TopicHeatInfo, error) {
	key := fmt.Sprintf("cache:topic_heat:%s", date)
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(key); found {
		cached := v.(map[int64]*dal_model.TopicHeatInfo)
		if info, ok := cached[topicID]; ok && info != nil {
			copied := *info
			return &copied, nil
		}
	}
	return nil, nil
}

func (c TopicHeatCacheImpl) GetAll(ctx context.Context, date string) (map[int64]*dal_model.TopicHeatInfo, error) {
	key := fmt.Sprintf("cache:topic_heat:%s", date)
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(key); found {
		cached := v.(map[int64]*dal_model.TopicHeatInfo)
		result := make(map[int64]*dal_model.TopicHeatInfo, len(cached))
		for topicID, info := range cached {
			if info == nil {
				continue
			}
			copied := *info
			result[topicID] = &copied
		}
		return result, nil
	}
	return nil, nil
}

func (c TopicHeatCacheImpl) SetAll(ctx context.Context, date string, data map[int64]*dal_model.TopicHeatInfo) error {
	key := fmt.Sprintf("cache:topic_heat:%s", date)
	Lock()
	defer Unlock()

	// Keep a deep copy so the v2 strong/rising semantics are preserved after cache round-trips.
	copied := make(map[int64]*dal_model.TopicHeatInfo, len(data))
	for topicID, info := range data {
		if info == nil {
			continue
		}
		item := *info
		copied[topicID] = &item
	}
	Cache.Set(key, copied, TTLUntilEndOfDay())
	return nil
}
