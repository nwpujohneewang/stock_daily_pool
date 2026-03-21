package repo

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"stock/internal/model"
)

type PoolRepo struct {
	db *gorm.DB
}

func NewPoolRepo(db *gorm.DB) *PoolRepo {
	return &PoolRepo{db: db}
}

func (r *PoolRepo) GetByDate(ctx context.Context, date string, poolType int) ([]model.DailyStockPool, error) {
	var pools []model.DailyStockPool
	tableName := fmt.Sprintf("daily_stock_pool_%s", date[0:7])
	err := r.db.WithContext(ctx).Raw(fmt.Sprintf(`
		SELECT id, date, ts_code, stock_name, pool_type, change_pct, current_price, pre_close, limit_up_price, first_limit_time, board_code, topic_ids, snapshot_time, created_at
		FROM %s WHERE date = ? AND pool_type = ?
	`, tableName), date, poolType).Scan(&pools).Error
	if err != nil {
		return nil, err
	}
	return pools, nil
}

func (r *PoolRepo) Upsert(ctx context.Context, pool *model.DailyStockPool) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "date"}, {Name: "ts_code"}, {Name: "pool_type"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"stock_name", "change_pct", "current_price", "pre_close",
			"limit_up_price", "first_limit_time", "board_code", "topic_ids", "snapshot_time",
		}),
	}).Create(pool).Error
}

func (r *PoolRepo) UpsertSnapshot(ctx context.Context, pool *model.DailyStockPool) error {
	return r.Upsert(ctx, pool)
}

func (r *PoolRepo) GetByTsCodeAndDate(ctx context.Context, date, tsCode string) (*model.DailyStockPool, error) {
	var pools []model.DailyStockPool
	tableName := fmt.Sprintf("daily_stock_pool_%s", date[0:7])
	err := r.db.WithContext(ctx).Raw(fmt.Sprintf(`
		SELECT id, date, ts_code, stock_name, pool_type, change_pct, current_price, pre_close, limit_up_price, first_limit_time, board_code, topic_ids, snapshot_time, created_at
		FROM %s WHERE date = ? AND ts_code = ?
	`, tableName), date, tsCode).Scan(&pools).Error
	if err != nil {
		return nil, err
	}
	if len(pools) == 0 {
		return nil, nil
	}
	return &pools[0], nil
}
