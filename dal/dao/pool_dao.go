package dao

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"stock/model/dal_model"
)

// dateFormatRegex validates YYYY-MM-DD format
var dateFormatRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// validateDateFormat ensures the date string matches YYYY-MM-DD format
func validateDateFormat(date string) error {
	if len(date) < 7 {
		return fmt.Errorf("invalid date format: date too short")
	}
	if !dateFormatRegex.MatchString(date) {
		return fmt.Errorf("invalid date format: expected YYYY-MM-DD, got %s", date)
	}
	return nil
}

var _ PoolDAO = (*poolDAOImpl)(nil)

type PoolDAO interface {
	GetByDate(ctx context.Context, date string, poolType int) ([]dal_model.DailyStockPool, error)
	Upsert(ctx context.Context, pool *dal_model.DailyStockPool) error
	UpsertSnapshot(ctx context.Context, pool *dal_model.DailyStockPool) error
	UpsertSnapshotBatch(ctx context.Context, pools []dal_model.DailyStockPool) error
	GetByTsCodeAndDate(ctx context.Context, date, tsCode string) (*dal_model.DailyStockPool, error)
}

type poolDAOImpl struct{}

func NewPoolDAO() PoolDAO {
	return &poolDAOImpl{}
}

func (r poolDAOImpl) GetByDate(ctx context.Context, date string, poolType int) ([]dal_model.DailyStockPool, error) {
	if err := validateDateFormat(date); err != nil {
		return nil, err
	}
	var pools []dal_model.DailyStockPool
	err := PostgresStockDB(ctx).Where("date = ? AND pool_type = ?", date, poolType).Find(&pools).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return pools, nil
}

func (r poolDAOImpl) Upsert(ctx context.Context, pool *dal_model.DailyStockPool) error {
	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "date"}, {Name: "ts_code"}, {Name: "pool_type"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"stock_name", "change_pct", "current_price", "pre_close",
			"limit_up_price", "first_limit_time", "board_code", "topic_ids",
			"vol", "amount", "snapshot_time",
		}),
	}).Create(pool).Error
}

func (r poolDAOImpl) UpsertSnapshot(ctx context.Context, pool *dal_model.DailyStockPool) error {
	return r.Upsert(ctx, pool)
}

func (r poolDAOImpl) UpsertSnapshotBatch(ctx context.Context, pools []dal_model.DailyStockPool) error {
	if len(pools) == 0 {
		return nil
	}
	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "date"}, {Name: "ts_code"}, {Name: "pool_type"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"stock_name", "change_pct", "current_price", "pre_close",
			"limit_up_price", "first_limit_time", "board_code", "topic_ids",
			"vol", "amount", "snapshot_time",
		}),
	}).CreateInBatches(&pools, 500).Error
}

func (r poolDAOImpl) GetByTsCodeAndDate(ctx context.Context, date, tsCode string) (*dal_model.DailyStockPool, error) {
	if err := validateDateFormat(date); err != nil {
		return nil, err
	}
	var pools []dal_model.DailyStockPool
	err := PostgresStockDB(ctx).Where("date = ? AND ts_code = ?", date, tsCode).Find(&pools).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if len(pools) == 0 {
		return nil, nil
	}
	return &pools[0], nil
}

func (r poolDAOImpl) GetByTsCode(ctx context.Context, tsCode string) (*dal_model.DailyStockPool, error) {
	var pool dal_model.DailyStockPool
	err := PostgresStockDB(ctx).Where("ts_code = ?", tsCode).First(&pool).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &pool, nil
}
