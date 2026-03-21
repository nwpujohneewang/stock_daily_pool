package db

import (
	"context"
	"fmt"

	"gorm.io/gorm/clause"
	"stock/internal/model"
)

var _ PoolRepository = (*PoolRepoImpl)(nil)

type PoolRepository interface {
	GetByDate(ctx context.Context, date string, poolType int) ([]model.DailyStockPool, error)
	Upsert(ctx context.Context, pool *model.DailyStockPool) error
	UpsertSnapshot(ctx context.Context, pool *model.DailyStockPool) error
	GetByTsCodeAndDate(ctx context.Context, date, tsCode string) (*model.DailyStockPool, error)
}

type PoolRepoImpl struct{}

func NewPoolRepository() *PoolRepoImpl {
	return &PoolRepoImpl{}
}

func (r PoolRepoImpl) GetByDate(ctx context.Context, date string, poolType int) ([]model.DailyStockPool, error) {
	var pools []model.DailyStockPool
	tableName := fmt.Sprintf("daily_stock_pool_%s", date[0:7])
	err := PostgresStockDB(ctx).Table(tableName).Where("date = ? AND pool_type = ?", date, poolType).Find(&pools).Error
	if err != nil {
		return nil, err
	}
	return pools, nil
}

func (r PoolRepoImpl) Upsert(ctx context.Context, pool *model.DailyStockPool) error {
	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "date"}, {Name: "ts_code"}, {Name: "pool_type"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"stock_name", "change_pct", "current_price", "pre_close",
			"limit_up_price", "first_limit_time", "board_code", "topic_ids", "snapshot_time",
		}),
	}).Create(pool).Error
}

func (r PoolRepoImpl) UpsertSnapshot(ctx context.Context, pool *model.DailyStockPool) error {
	return r.Upsert(ctx, pool)
}

func (r PoolRepoImpl) GetByTsCodeAndDate(ctx context.Context, date, tsCode string) (*model.DailyStockPool, error) {
	var pools []model.DailyStockPool
	tableName := fmt.Sprintf("daily_stock_pool_%s", date[0:7])
	err := PostgresStockDB(ctx).Table(tableName).Where("date = ? AND ts_code = ?", date, tsCode).Find(&pools).Error
	if err != nil {
		return nil, err
	}
	if len(pools) == 0 {
		return nil, nil
	}
	return &pools[0], nil
}

func (r PoolRepoImpl) GetByTsCode(ctx context.Context, tsCode string) (*model.DailyStockPool, error) {
	var pool model.DailyStockPool
	err := PostgresStockDB(ctx).Where("ts_code = ?", tsCode).First(&pool).Error
	if err != nil {
		return nil, err
	}
	return &pool, nil
}
