package cache

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type SnapshotCache struct {
	client *redis.Client
}

func NewSnapshotCache(client *redis.Client) *SnapshotCache {
	return &SnapshotCache{client: client}
}

const snapshotKey = "snapshot:current"

type PoolSnapshotData struct {
	Date         string      `json:"date"`
	SnapshotTime string      `json:"snapshot_time"`
	LimitUp      interface{} `json:"limit_up"`
	Above5Pct    interface{} `json:"above_5pct"`
}

func (c *SnapshotCache) SaveSnapshot(ctx context.Context, data *PoolSnapshotData) error {
	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	return c.client.Set(ctx, snapshotKey, body, 0).Err()
}

func (c *SnapshotCache) GetSnapshot(ctx context.Context) (*PoolSnapshotData, error) {
	body, err := c.client.Get(ctx, snapshotKey).Bytes()
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

func (c *SnapshotCache) DeleteSnapshot(ctx context.Context) error {
	return c.client.Del(ctx, snapshotKey).Err()
}
