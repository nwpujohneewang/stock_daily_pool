package cache

import "context"

const topicStockCountKey = "cache:topic_stock_count"

type TopicStockCountCacheInterface interface {
	GetAll(ctx context.Context) (map[int64]int, error)
	Get(ctx context.Context, topicID int64) (int, error)
	SetAll(ctx context.Context, data map[int64]int) error
}

type TopicStockCountCacheImpl struct{}

func NewTopicStockCountCache() *TopicStockCountCacheImpl {
	return &TopicStockCountCacheImpl{}
}

func (c TopicStockCountCacheImpl) GetAll(ctx context.Context) (map[int64]int, error) {
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(topicStockCountKey); found {
		cached := v.(map[int64]int)
		result := make(map[int64]int, len(cached))
		for topicID, count := range cached {
			result[topicID] = count
		}
		return result, nil
	}
	return nil, nil
}

func (c TopicStockCountCacheImpl) Get(ctx context.Context, topicID int64) (int, error) {
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(topicStockCountKey); found {
		cached := v.(map[int64]int)
		return cached[topicID], nil
	}
	return 0, nil
}

func (c TopicStockCountCacheImpl) SetAll(ctx context.Context, data map[int64]int) error {
	Lock()
	defer Unlock()

	copied := make(map[int64]int, len(data))
	for topicID, count := range data {
		copied[topicID] = count
	}
	Cache.Set(topicStockCountKey, copied, TTLUntilEndOfDay())
	return nil
}
