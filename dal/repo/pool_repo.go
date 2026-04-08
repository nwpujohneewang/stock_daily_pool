package repo

import (
	"context"
	"stock/dal/cache"
	"stock/dal/dao"
	"stock/model/dal_model"
)

type PoolRepository interface {
	GetByDate(ctx context.Context, date string, poolType int) ([]dal_model.DailyStockPool, error)
	Upsert(ctx context.Context, pool *dal_model.DailyStockPool) error
	UpsertSnapshot(ctx context.Context, pool *dal_model.DailyStockPool) error
	UpsertSnapshotBatch(ctx context.Context, pools []dal_model.DailyStockPool) error
	GetByTsCodeAndDate(ctx context.Context, date, tsCode string) (*dal_model.DailyStockPool, error)
}

type poolRepoImpl struct {
	dao dao.PoolDAO
}

func NewPoolRepository() PoolRepository {
	return &poolRepoImpl{dao: dao.NewPoolDAO()}
}

func (r *poolRepoImpl) GetByDate(ctx context.Context, date string, poolType int) ([]dal_model.DailyStockPool, error) {
	poolCache := cache.NewPoolDataCache()
	if pools, found := poolCache.GetByDate(ctx, date, poolType); found {
		return pools, nil
	}
	pools, err := r.dao.GetByDate(ctx, date, poolType)
	if err != nil {
		return nil, err
	}
	_ = poolCache.SetByDate(ctx, date, poolType, pools)
	return pools, nil
}

func (r *poolRepoImpl) Upsert(ctx context.Context, pool *dal_model.DailyStockPool) error {
	if err := r.dao.Upsert(ctx, pool); err != nil {
		return err
	}
	return r.invalidateDate(ctx, pool.Date.Format("2006-01-02"), int(pool.PoolType))
}

func (r *poolRepoImpl) UpsertSnapshot(ctx context.Context, pool *dal_model.DailyStockPool) error {
	if err := r.dao.UpsertSnapshot(ctx, pool); err != nil {
		return err
	}
	return r.invalidateDate(ctx, pool.Date.Format("2006-01-02"), int(pool.PoolType))
}

func (r *poolRepoImpl) UpsertSnapshotBatch(ctx context.Context, pools []dal_model.DailyStockPool) error {
	if err := r.dao.UpsertSnapshotBatch(ctx, pools); err != nil {
		return err
	}
	for _, pool := range pools {
		_ = r.invalidateDate(ctx, pool.Date.Format("2006-01-02"), int(pool.PoolType))
	}
	return nil
}

func (r *poolRepoImpl) GetByTsCodeAndDate(ctx context.Context, date, tsCode string) (*dal_model.DailyStockPool, error) {
	return r.dao.GetByTsCodeAndDate(ctx, date, tsCode)
}

func (r *poolRepoImpl) invalidateDate(ctx context.Context, date string, poolType int) error {
	poolCache := cache.NewPoolDataCache()
	return poolCache.ClearByDate(ctx, date, poolType)
}
