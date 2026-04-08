package cache

import (
	"context"
	"fmt"
	"stock/model/dal_model"
)

type DailySnapshotCacheInterface interface {
	SetByDate(ctx context.Context, date string, snapshots []dal_model.DailyStockSnapshot) error
	GetByDate(ctx context.Context, date string) ([]dal_model.DailyStockSnapshot, bool)
	ClearByDate(ctx context.Context, date string) error
	SetLimitUpByDate(ctx context.Context, date string, snapshots []dal_model.DailyStockSnapshot) error
	GetLimitUpByDate(ctx context.Context, date string) ([]dal_model.DailyStockSnapshot, bool)
	ClearLimitUpByDate(ctx context.Context, date string) error
}

type DailySnapshotCacheImpl struct{}

func NewDailySnapshotCache() *DailySnapshotCacheImpl {
	return &DailySnapshotCacheImpl{}
}

func dailySnapshotKey(date string) string {
	return fmt.Sprintf("snapshot:data:%s", date)
}

func dailySnapshotLimitUpKey(date string) string {
	return fmt.Sprintf("snapshot:limitup:%s", date)
}

func (c DailySnapshotCacheImpl) SetByDate(ctx context.Context, date string, snapshots []dal_model.DailyStockSnapshot) error {
	copied := make([]dal_model.DailyStockSnapshot, len(snapshots))
	copy(copied, snapshots)
	Cache.Set(dailySnapshotKey(date), copied, TTLUntilEndOfDay())
	return nil
}

func (c DailySnapshotCacheImpl) GetByDate(ctx context.Context, date string) ([]dal_model.DailyStockSnapshot, bool) {
	if v, found := Cache.Get(dailySnapshotKey(date)); found {
		if snapshots, ok := v.([]dal_model.DailyStockSnapshot); ok {
			copied := make([]dal_model.DailyStockSnapshot, len(snapshots))
			copy(copied, snapshots)
			return copied, true
		}
	}
	return nil, false
}

func (c DailySnapshotCacheImpl) ClearByDate(ctx context.Context, date string) error {
	Cache.Delete(dailySnapshotKey(date))
	return nil
}

func (c DailySnapshotCacheImpl) SetLimitUpByDate(ctx context.Context, date string, snapshots []dal_model.DailyStockSnapshot) error {
	copied := make([]dal_model.DailyStockSnapshot, len(snapshots))
	copy(copied, snapshots)
	Cache.Set(dailySnapshotLimitUpKey(date), copied, TTLUntilEndOfDay())
	return nil
}

func (c DailySnapshotCacheImpl) GetLimitUpByDate(ctx context.Context, date string) ([]dal_model.DailyStockSnapshot, bool) {
	if v, found := Cache.Get(dailySnapshotLimitUpKey(date)); found {
		if snapshots, ok := v.([]dal_model.DailyStockSnapshot); ok {
			copied := make([]dal_model.DailyStockSnapshot, len(snapshots))
			copy(copied, snapshots)
			return copied, true
		}
	}
	return nil, false
}

func (c DailySnapshotCacheImpl) ClearLimitUpByDate(ctx context.Context, date string) error {
	Cache.Delete(dailySnapshotLimitUpKey(date))
	return nil
}
