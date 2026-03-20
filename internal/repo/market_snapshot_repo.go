package repo

import (
	"context"
	"time"

	"gorm.io/gorm"
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
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO market_snapshots (date, status, raw_data, topic_count, stock_count, updated_at)
		VALUES (?, ?, ?, ?, ?, NOW())
		ON CONFLICT (date) DO UPDATE SET
			status = EXCLUDED.status,
			raw_data = EXCLUDED.raw_data,
			topic_count = EXCLUDED.topic_count,
			stock_count = EXCLUDED.stock_count,
			retry_count = market_snapshots.retry_count + 1,
			updated_at = NOW()
	`, snapshot.Date, snapshot.Status, snapshot.RawData, snapshot.TopicCount, snapshot.StockCount).Error
}

func (r *MarketSnapshotRepo) GetByDate(ctx context.Context, date string) (*MarketSnapshot, error) {
	var m MarketSnapshot
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, date, status, raw_data, topic_count, stock_count, retry_count, error_msg, created_at, updated_at
		FROM market_snapshots WHERE date = ?
	`, date).Scan(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MarketSnapshotRepo) GetFailedDates(ctx context.Context) ([]string, error) {
	var dates []string
	err := r.db.WithContext(ctx).Raw(`SELECT date FROM market_snapshots WHERE status = 2 ORDER BY date DESC`).Scan(&dates).Error
	if err != nil {
		return nil, err
	}
	return dates, nil
}

func (r *MarketSnapshotRepo) UpdateStatus(ctx context.Context, date string, status int, errorMsg string) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE market_snapshots SET status = ?, error_msg = ?, updated_at = NOW() WHERE date = ?
	`, status, errorMsg, date).Error
}
