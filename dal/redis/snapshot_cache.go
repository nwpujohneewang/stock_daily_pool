package redis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
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
	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	return RedisClient(ctx).Set(ctx, snapshotKey, body, 0).Err()
}

func (c SnapshotCacheImpl) GetSnapshot(ctx context.Context) (*PoolSnapshotData, error) {
	body, err := RedisClient(ctx).Get(ctx, snapshotKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	var data PoolSnapshotData
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("unmarshal snapshot: %w", err)
	}
	return &data, nil
}

func (c SnapshotCacheImpl) DeleteSnapshot(ctx context.Context) error {
	return RedisClient(ctx).Del(ctx, snapshotKey).Err()
}
