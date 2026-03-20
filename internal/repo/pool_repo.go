package repo

import (
	"context"
	"fmt"

	"gorm.io/gorm"
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
	err := r.db.WithContext(ctx).Raw(fmt.Sprintf(`
		SELECT id, date, ts_code, stock_name, pool_type, change_pct, current_price, pre_close, limit_up_price, first_limit_time, board_code, topic_ids, snapshot_time, created_at
		FROM daily_stock_pool_%s WHERE date = ? AND pool_type = ?
	`, date[0:7]), date, poolType).Scan(&pools).Error
	if err != nil {
		return nil, err
	}
	return pools, nil
}

func (r *PoolRepo) Upsert(ctx context.Context, pool *model.DailyStockPool) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO daily_stock_pool (date, ts_code, stock_name, pool_type, change_pct, current_price, pre_close, limit_up_price, first_limit_time, board_code, topic_ids, snapshot_time)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (date, ts_code, pool_type) DO UPDATE SET
			stock_name = EXCLUDED.stock_name,
			change_pct = EXCLUDED.change_pct,
			current_price = EXCLUDED.current_price,
			pre_close = EXCLUDED.pre_close,
			limit_up_price = EXCLUDED.limit_up_price,
			first_limit_time = EXCLUDED.first_limit_time,
			board_code = EXCLUDED.board_code,
			topic_ids = EXCLUDED.topic_ids,
			snapshot_time = EXCLUDED.snapshot_time
	`, pool.Date, pool.TsCode, pool.StockName, pool.PoolType, pool.ChangePct, pool.CurrentPrice,
		pool.PreClose, pool.LimitUpPrice, pool.FirstLimitTime, pool.BoardCode, pool.TopicIDs, pool.SnapshotTime).Error
}

func (r *PoolRepo) UpsertSnapshot(ctx context.Context, pool *model.DailyStockPool) error {
	return r.Upsert(ctx, pool)
}

func (r *PoolRepo) GetByTsCodeAndDate(ctx context.Context, date, tsCode string) (*model.DailyStockPool, error) {
	var pools []model.DailyStockPool
	err := r.db.WithContext(ctx).Raw(fmt.Sprintf(`
		SELECT id, date, ts_code, stock_name, pool_type, change_pct, current_price, pre_close, limit_up_price, first_limit_time, board_code, topic_ids, snapshot_time, created_at
		FROM daily_stock_pool_%s WHERE date = ? AND ts_code = ?
	`, date[0:7]), date, tsCode).Scan(&pools).Error
	if err != nil {
		return nil, err
	}
	if len(pools) == 0 {
		return nil, nil
	}
	return &pools[0], nil
}
