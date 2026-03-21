package repo

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MarketSnapshot struct {
	ID         int64     `gorm:"column:id"`
	Date       time.Time `gorm:"column:date"`
	Status     int       `gorm:"column:status"`
	RawData    []byte    `gorm:"column:raw_data"`
	TopicCount int       `gorm:"column:topic_count"`
	StockCount int       `gorm:"column:stock_count"`
	RetryCount int       `gorm:"column:retry_count"`
	ErrorMsg   string    `gorm:"column:error_msg"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

type MarketSnapshotRepo struct {
	db *gorm.DB
}

func NewMarketSnapshotRepo(db *gorm.DB) *MarketSnapshotRepo {
	return &MarketSnapshotRepo{db: db}
}

func (r *MarketSnapshotRepo) Upsert(ctx context.Context, snapshot *MarketSnapshot) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"status", "raw_data", "topic_count", "stock_count",
			"retry_count", "updated_at",
		}),
	}).Create(snapshot).Error
}

func (r *MarketSnapshotRepo) GetByDate(ctx context.Context, date string) (*MarketSnapshot, error) {
	var m MarketSnapshot
	err := r.db.WithContext(ctx).Where("date = ?", date).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MarketSnapshotRepo) GetFailedDates(ctx context.Context) ([]string, error) {
	var dates []string
	err := r.db.WithContext(ctx).Select("date").Where("status = ?", 2).
		Order("date DESC").Find(&dates).Error
	if err != nil {
		return nil, err
	}
	return dates, nil
}

func (r *MarketSnapshotRepo) UpdateStatus(ctx context.Context, date string, status int, errorMsg string) error {
	return r.db.WithContext(ctx).Model(&MarketSnapshot{}).
		Where("date = ?", date).
		Updates(map[string]interface{}{
			"status":     status,
			"error_msg":  errorMsg,
			"updated_at": time.Now(),
		}).Error
}
