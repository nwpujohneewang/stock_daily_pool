package repo

import (
	"context"
	"fmt"
	"stock/dal/cache"
	"stock/dal/dao"
	"stock/model/dal_model"
	"strings"
	"time"

	"gorm.io/gorm"
)

type snapshotPartitionManager struct {
	db    *gorm.DB
	table string
}

type snapshotPartitionEnsurer interface {
	EnsureMonthPartition(ctx context.Context, target time.Time) error
}

func (m *snapshotPartitionManager) EnsureMonthPartition(ctx context.Context, target time.Time) error {
	start := time.Date(target.Year(), target.Month(), 1, 0, 0, 0, 0, target.Location())
	end := start.AddDate(0, 1, 0)
	partitionName := fmt.Sprintf("%s_%04d_%02d", m.table, start.Year(), int(start.Month()))
	err := m.db.WithContext(ctx).Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s
		PARTITION OF %s
		FOR VALUES FROM ('%s') TO ('%s')
	`, partitionName, m.table, start.Format("2006-01-02"), end.Format("2006-01-02"))).Error
	if err != nil {
		return fmt.Errorf("create partition %s: %w", partitionName, err)
	}
	return nil
}

type SnapshotRepository interface {
	DeleteByDate(ctx context.Context, date string) error
	DeleteBeforeOrEqualDate(ctx context.Context, cutoffDate string) error
	InsertBatch(ctx context.Context, snapshots []dal_model.DailyStockSnapshot) error
	GetByDate(ctx context.Context, date string) ([]dal_model.DailyStockSnapshot, error)
	GetLimitUpByDate(ctx context.Context, date string) ([]dal_model.DailyStockSnapshot, error)
	GetByTsCode(ctx context.Context, tsCode string) ([]dal_model.DailyStockSnapshot, error)
	UpdateTopicIDByTsCodeAndDate(ctx context.Context, date, tsCode string, topicID *int64) error
}

type snapshotRepoImpl struct {
	dao              dao.SnapshotDAO
	partitionEnsurer snapshotPartitionEnsurer
}

func NewSnapshotRepository() SnapshotRepository {
	return &snapshotRepoImpl{
		dao:              dao.NewSnapshotDAO(),
		partitionEnsurer: &snapshotPartitionManager{db: dao.DB, table: dal_model.DailyStockSnapshot{}.TableName()},
	}
}

func (r *snapshotRepoImpl) DeleteByDate(ctx context.Context, date string) error {
	if err := r.ensureDatePartition(ctx, date); err != nil {
		return err
	}
	if err := r.dao.DeleteByDate(ctx, date); err != nil {
		return err
	}
	return r.invalidateDate(ctx, date)
}

func (r *snapshotRepoImpl) DeleteBeforeOrEqualDate(ctx context.Context, cutoffDate string) error {
	if err := r.dao.DeleteBeforeOrEqualDate(ctx, cutoffDate); err != nil {
		return err
	}
	return r.invalidateBeforeOrEqual(ctx, cutoffDate)
}

func (r *snapshotRepoImpl) InsertBatch(ctx context.Context, snapshots []dal_model.DailyStockSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}
	if err := r.partitionEnsurer.EnsureMonthPartition(ctx, snapshots[0].Date); err != nil {
		return fmt.Errorf("ensure snapshot partition: %w", err)
	}
	if err := r.dao.InsertBatch(ctx, snapshots); err != nil {
		return err
	}
	for _, snapshot := range snapshots {
		_ = r.invalidateDate(ctx, snapshot.Date.Format("2006-01-02"))
	}
	return nil
}

func (r *snapshotRepoImpl) GetByDate(ctx context.Context, date string) ([]dal_model.DailyStockSnapshot, error) {
	snapshotCache := cache.NewDailySnapshotCache()
	if snapshots, found := snapshotCache.GetByDate(ctx, date); found {
		return snapshots, nil
	}
	snapshots, err := r.dao.GetByDate(ctx, date)
	if err != nil {
		return nil, err
	}
	_ = snapshotCache.SetByDate(ctx, date, snapshots)
	return snapshots, nil
}

func (r *snapshotRepoImpl) GetLimitUpByDate(ctx context.Context, date string) ([]dal_model.DailyStockSnapshot, error) {
	snapshotCache := cache.NewDailySnapshotCache()
	if snapshots, found := snapshotCache.GetLimitUpByDate(ctx, date); found {
		return snapshots, nil
	}
	snapshots, err := r.dao.GetLimitUpByDate(ctx, date)
	if err != nil {
		return nil, err
	}
	_ = snapshotCache.SetLimitUpByDate(ctx, date, snapshots)
	return snapshots, nil
}

func (r *snapshotRepoImpl) GetByTsCode(ctx context.Context, tsCode string) ([]dal_model.DailyStockSnapshot, error) {
	return r.dao.GetByTsCode(ctx, tsCode)
}

func (r *snapshotRepoImpl) UpdateTopicIDByTsCodeAndDate(ctx context.Context, date, tsCode string, topicID *int64) error {
	if err := r.ensureDatePartition(ctx, date); err != nil {
		return err
	}
	if err := r.dao.UpdateTopicIDByTsCodeAndDate(ctx, date, tsCode, topicID); err != nil {
		return err
	}
	return r.invalidateDate(ctx, date)
}

func (r *snapshotRepoImpl) invalidateDate(ctx context.Context, date string) error {
	snapshotCache := cache.NewDailySnapshotCache()
	_ = snapshotCache.ClearByDate(ctx, date)
	_ = snapshotCache.ClearLimitUpByDate(ctx, date)
	return nil
}

func (r *snapshotRepoImpl) invalidateBeforeOrEqual(ctx context.Context, cutoffDate string) error {
	snapshotCache := cache.NewDailySnapshotCache()
	for k := range cache.Cache.Items() {
		date, ok := dateFromSnapshotCacheKey(k)
		if !ok {
			continue
		}
		if date <= cutoffDate {
			_ = snapshotCache.ClearByDate(ctx, date)
			_ = snapshotCache.ClearLimitUpByDate(ctx, date)
		}
	}
	return nil
}

func dateFromSnapshotCacheKey(key string) (string, bool) {
	// Keys are produced by DailySnapshotCacheImpl:
	// - snapshot:data:YYYY-MM-DD
	// - snapshot:limitup:YYYY-MM-DD
	const dataPrefix = "snapshot:data:"
	const limitPrefix = "snapshot:limitup:"
	if strings.HasPrefix(key, dataPrefix) {
		return strings.TrimPrefix(key, dataPrefix), true
	}
	if strings.HasPrefix(key, limitPrefix) {
		return strings.TrimPrefix(key, limitPrefix), true
	}
	return "", false
}

func (r *snapshotRepoImpl) ensureDatePartition(ctx context.Context, date string) error {
	target, err := time.ParseInLocation("2006-01-02", date, time.Local)
	if err != nil {
		return fmt.Errorf("parse snapshot date %s: %w", date, err)
	}
	if err := r.partitionEnsurer.EnsureMonthPartition(ctx, target); err != nil {
		return fmt.Errorf("ensure snapshot partition: %w", err)
	}
	return nil
}
