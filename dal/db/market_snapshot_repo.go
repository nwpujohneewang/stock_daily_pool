package db

import (
	"context"
	"time"

	"gorm.io/gorm/clause"
	"stock/internal/model"
)

var _ MarketSnapshotRepository = (*MarketSnapshotRepoImpl)(nil)

type MarketSnapshotRepository interface {
	Upsert(ctx context.Context, snapshot *model.MarketSnapshot) error
	GetByDate(ctx context.Context, date string) (*model.MarketSnapshot, error)
	GetFailedDates(ctx context.Context) ([]string, error)
	UpdateStatus(ctx context.Context, date string, status int, errorMsg string) error
}

type MarketSnapshotRepoImpl struct{}

func NewMarketSnapshotRepository() *MarketSnapshotRepoImpl {
	return &MarketSnapshotRepoImpl{}
}

func (r MarketSnapshotRepoImpl) Upsert(ctx context.Context, snapshot *model.MarketSnapshot) error {
	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"status", "raw_data", "topic_count", "stock_count",
			"retry_count", "updated_at",
		}),
	}).Create(snapshot).Error
}

func (r MarketSnapshotRepoImpl) GetByDate(ctx context.Context, date string) (*model.MarketSnapshot, error) {
	var m model.MarketSnapshot
	err := PostgresStockDB(ctx).Where("date = ?", date).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r MarketSnapshotRepoImpl) GetFailedDates(ctx context.Context) ([]string, error) {
	var dates []string
	err := PostgresStockDB(ctx).Select("date").Where("status = ?", 2).
		Order("date DESC").Find(&dates).Error
	if err != nil {
		return nil, err
	}
	return dates, nil
}

func (r MarketSnapshotRepoImpl) UpdateStatus(ctx context.Context, date string, status int, errorMsg string) error {
	return PostgresStockDB(ctx).Model(&model.MarketSnapshot{}).
		Where("date = ?", date).
		Updates(map[string]interface{}{
			"status":     status,
			"error_msg":  errorMsg,
			"updated_at": time.Now(),
		}).Error
}
