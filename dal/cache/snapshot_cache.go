package cache

import (
	"context"
)

type SnapshotCacheInterface interface {
	SaveSnapshot(ctx context.Context, data *PoolSnapshotData) error
	GetSnapshot(ctx context.Context) (*PoolSnapshotData, error)
	DeleteSnapshot(ctx context.Context) error
}

var _ SnapshotCacheInterface = (*SnapshotCacheImpl)(nil)

type SnapshotCacheImpl struct{}

func NewSnapshotCache() *SnapshotCacheImpl {
	return &SnapshotCacheImpl{}
}

const snapshotKey = "snapshot:current"

type PoolSnapshotData struct {
	Date         string      `json:"date"`
	SnapshotTime string      `json:"snapshot_time"`
	LimitUp      interface{} `json:"limit_up"`
	Above5Pct    interface{} `json:"above_5pct"`
}

func (c SnapshotCacheImpl) SaveSnapshot(ctx context.Context, data *PoolSnapshotData) error {
	Lock()
	defer Unlock()

	// Store a copy
	copied := *data
	Cache.Set(snapshotKey, &copied, TTLUntilEndOfDay())
	return nil
}

func (c SnapshotCacheImpl) GetSnapshot(ctx context.Context) (*PoolSnapshotData, error) {
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(snapshotKey); found {
		data := v.(*PoolSnapshotData)
		// Return a copy
		copied := *data
		return &copied, nil
	}
	return nil, nil
}

func (c SnapshotCacheImpl) DeleteSnapshot(ctx context.Context) error {
	Lock()
	defer Unlock()

	Cache.Delete(snapshotKey)
	return nil
}
