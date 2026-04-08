package cache

import (
	"context"
	"fmt"
	"stock/model/dal_model"
)

type PoolDataCacheInterface interface {
	SetByDate(ctx context.Context, date string, poolType int, pools []dal_model.DailyStockPool) error
	GetByDate(ctx context.Context, date string, poolType int) ([]dal_model.DailyStockPool, bool)
	ClearByDate(ctx context.Context, date string, poolType int) error
}

type PoolDataCacheImpl struct{}

func NewPoolDataCache() *PoolDataCacheImpl {
	return &PoolDataCacheImpl{}
}

func poolDataKey(date string, poolType int) string {
	return fmt.Sprintf("pool:data:%s:%d", date, poolType)
}

func (c PoolDataCacheImpl) SetByDate(ctx context.Context, date string, poolType int, pools []dal_model.DailyStockPool) error {
	copied := make([]dal_model.DailyStockPool, len(pools))
	copy(copied, pools)
	Cache.Set(poolDataKey(date, poolType), copied, TTLUntilEndOfDay())
	return nil
}

func (c PoolDataCacheImpl) GetByDate(ctx context.Context, date string, poolType int) ([]dal_model.DailyStockPool, bool) {
	if v, found := Cache.Get(poolDataKey(date, poolType)); found {
		if pools, ok := v.([]dal_model.DailyStockPool); ok {
			copied := make([]dal_model.DailyStockPool, len(pools))
			copy(copied, pools)
			return copied, true
		}
	}
	return nil, false
}

func (c PoolDataCacheImpl) ClearByDate(ctx context.Context, date string, poolType int) error {
	Cache.Delete(poolDataKey(date, poolType))
	return nil
}
