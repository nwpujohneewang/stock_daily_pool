package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type PoolCache struct {
	client *redis.Client
}

func NewPoolCache(client *redis.Client) *PoolCache {
	return &PoolCache{client: client}
}

func (c *PoolCache) AddLimitUp(ctx context.Context, date, tsCode string) error {
	key := fmt.Sprintf("pool:limit_up:%s", date)
	return c.client.SAdd(ctx, key, tsCode).Err()
}

func (c *PoolCache) RemoveLimitUp(ctx context.Context, date, tsCode string) error {
	key := fmt.Sprintf("pool:limit_up:%s", date)
	return c.client.SRem(ctx, key, tsCode).Err()
}

func (c *PoolCache) GetLimitUpMembers(ctx context.Context, date string) ([]string, error) {
	key := fmt.Sprintf("pool:limit_up:%s", date)
	return c.client.SMembers(ctx, key).Result()
}

func (c *PoolCache) IsLimitUp(ctx context.Context, date, tsCode string) (bool, error) {
	key := fmt.Sprintf("pool:limit_up:%s", date)
	return c.client.SIsMember(ctx, key, tsCode).Result()
}

func (c *PoolCache) AddAbove5(ctx context.Context, date, tsCode string) error {
	key := fmt.Sprintf("pool:above5:%s", date)
	return c.client.SAdd(ctx, key, tsCode).Err()
}

func (c *PoolCache) RemoveAbove5(ctx context.Context, date, tsCode string) error {
	key := fmt.Sprintf("pool:above5:%s", date)
	return c.client.SRem(ctx, key, tsCode).Err()
}

func (c *PoolCache) GetAbove5Members(ctx context.Context, date string) ([]string, error) {
	key := fmt.Sprintf("pool:above5:%s", date)
	return c.client.SMembers(ctx, key).Result()
}

func (c *PoolCache) SetFirstLimitTime(ctx context.Context, date, tsCode, limitTime string) error {
	key := fmt.Sprintf("first_limit:%s", date)
	return c.client.HSetNX(ctx, key, tsCode, limitTime).Err()
}

func (c *PoolCache) GetFirstLimitTime(ctx context.Context, date, tsCode string) (string, error) {
	key := fmt.Sprintf("first_limit:%s", date)
	return c.client.HGet(ctx, key, tsCode).Result()
}
