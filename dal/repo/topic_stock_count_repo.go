package repo

import (
	"context"

	"stock/dal/cache"
	"stock/dal/dao"
	"stock/model/dal_model"
)

type TopicStockCountRepository interface {
	UpsertBatch(ctx context.Context, counts []dal_model.TopicStockCount) error
	GetAll(ctx context.Context) ([]dal_model.TopicStockCount, error)
	AggregateFromRelations(ctx context.Context) ([]dal_model.TopicStockCount, error)
	GetByTopicIDs(ctx context.Context, topicIDs []int64) ([]dal_model.TopicStockCount, error)
	GetAllCountMap(ctx context.Context) (map[int64]int, error)
	GetCountMapByTopicIDs(ctx context.Context, topicIDs []int64) (map[int64]int, error)
	WarmupCache(ctx context.Context) error
}

type topicStockCountRepoImpl struct {
	dao   dao.TopicStockCountDAO
	cache cache.TopicStockCountCacheInterface
}

func NewTopicStockCountRepository() TopicStockCountRepository {
	return &topicStockCountRepoImpl{
		dao:   dao.NewTopicStockCountDAO(),
		cache: cache.NewTopicStockCountCache(),
	}
}

func (r *topicStockCountRepoImpl) UpsertBatch(ctx context.Context, counts []dal_model.TopicStockCount) error {
	if err := r.dao.UpsertBatch(ctx, counts); err != nil {
		return err
	}
	return r.WarmupCache(ctx)
}

func (r *topicStockCountRepoImpl) GetAll(ctx context.Context) ([]dal_model.TopicStockCount, error) {
	return r.dao.GetAll(ctx)
}

func (r *topicStockCountRepoImpl) AggregateFromRelations(ctx context.Context) ([]dal_model.TopicStockCount, error) {
	return r.dao.AggregateFromRelations(ctx)
}

func (r *topicStockCountRepoImpl) GetByTopicIDs(ctx context.Context, topicIDs []int64) ([]dal_model.TopicStockCount, error) {
	return r.dao.GetByTopicIDs(ctx, topicIDs)
}

func (r *topicStockCountRepoImpl) GetAllCountMap(ctx context.Context) (map[int64]int, error) {
	if counts, err := r.cache.GetAll(ctx); err != nil {
		return nil, err
	} else if len(counts) > 0 {
		return counts, nil
	}

	rows, err := r.dao.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	counts := rowsToCountMap(rows)
	if len(counts) == 0 {
		return counts, nil
	}
	if err := r.cache.SetAll(ctx, counts); err != nil {
		return nil, err
	}
	return counts, nil
}

func (r *topicStockCountRepoImpl) GetCountMapByTopicIDs(ctx context.Context, topicIDs []int64) (map[int64]int, error) {
	if len(topicIDs) == 0 {
		return nil, nil
	}

	cachedCounts, err := r.cache.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	if cachedCounts == nil {
		cachedCounts = make(map[int64]int)
	}

	result := make(map[int64]int, len(topicIDs))
	missing := make([]int64, 0)
	for _, topicID := range topicIDs {
		if count, ok := cachedCounts[topicID]; ok {
			result[topicID] = count
			continue
		}
		missing = append(missing, topicID)
	}
	if len(missing) == 0 {
		return result, nil
	}

	rows, err := r.dao.GetByTopicIDs(ctx, missing)
	if err != nil {
		return nil, err
	}
	loaded := rowsToCountMap(rows)
	for topicID, count := range loaded {
		cachedCounts[topicID] = count
		result[topicID] = count
	}
	if len(loaded) > 0 {
		if err := r.cache.SetAll(ctx, cachedCounts); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (r *topicStockCountRepoImpl) WarmupCache(ctx context.Context) error {
	_, err := r.GetAllCountMap(ctx)
	return err
}

func rowsToCountMap(rows []dal_model.TopicStockCount) map[int64]int {
	counts := make(map[int64]int, len(rows))
	for _, row := range rows {
		counts[row.TopicID] = row.TotalStockCount
	}
	return counts
}
