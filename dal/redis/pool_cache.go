package redis

import (
	"context"
	"fmt"
)

type PoolCacheInterface interface {
	AddLimitUp(ctx context.Context, date, tsCode string) error
	RemoveLimitUp(ctx context.Context, date, tsCode string) error
	GetLimitUpMembers(ctx context.Context, date string) ([]string, error)
	IsLimitUp(ctx context.Context, date, tsCode string) (bool, error)
	AddAbove5(ctx context.Context, date, tsCode string) error
	RemoveAbove5(ctx context.Context, date, tsCode string) error
	GetAbove5Members(ctx context.Context, date string) ([]string, error)
	SetFirstLimitTime(ctx context.Context, date, tsCode, limitTime string) error
	GetFirstLimitTime(ctx context.Context, date, tsCode string) (string, error)
	AddLimitUpBatch(ctx context.Context, date string, tsCodes []string) error
	AddAbove5Batch(ctx context.Context, date string, tsCodes []string) error
}

var _ PoolCacheInterface = (*PoolCacheImpl)(nil)

type PoolCacheImpl struct{}

func NewPoolCache() *PoolCacheImpl {
	return &PoolCacheImpl{}
}

func (c PoolCacheImpl) AddLimitUp(ctx context.Context, date, tsCode string) error {
	key := fmt.Sprintf("pool:limit_up:%s", date)
	return RedisClient(ctx).SAdd(ctx, key, tsCode).Err()
}

func (c PoolCacheImpl) AddLimitUpBatch(ctx context.Context, date string, tsCodes []string) error {
	if len(tsCodes) == 0 {
		return nil
	}
	key := fmt.Sprintf("pool:limit_up:%s", date)
	members := make([]interface{}, len(tsCodes))
	for i, code := range tsCodes {
		members[i] = code
	}
	return RedisClient(ctx).SAdd(ctx, key, members...).Err()
}

func (c PoolCacheImpl) RemoveLimitUp(ctx context.Context, date, tsCode string) error {
	key := fmt.Sprintf("pool:limit_up:%s", date)
	return RedisClient(ctx).SRem(ctx, key, tsCode).Err()
}

func (c PoolCacheImpl) GetLimitUpMembers(ctx context.Context, date string) ([]string, error) {
	key := fmt.Sprintf("pool:limit_up:%s", date)
	return RedisClient(ctx).SMembers(ctx, key).Result()
}

func (c PoolCacheImpl) IsLimitUp(ctx context.Context, date, tsCode string) (bool, error) {
	key := fmt.Sprintf("pool:limit_up:%s", date)
	return RedisClient(ctx).SIsMember(ctx, key, tsCode).Result()
}

func (c PoolCacheImpl) AddAbove5(ctx context.Context, date, tsCode string) error {
	key := fmt.Sprintf("pool:above5:%s", date)
	return RedisClient(ctx).SAdd(ctx, key, tsCode).Err()
}

func (c PoolCacheImpl) AddAbove5Batch(ctx context.Context, date string, tsCodes []string) error {
	if len(tsCodes) == 0 {
		return nil
	}
	key := fmt.Sprintf("pool:above5:%s", date)
	members := make([]interface{}, len(tsCodes))
	for i, code := range tsCodes {
		members[i] = code
	}
	return RedisClient(ctx).SAdd(ctx, key, members...).Err()
}

func (c PoolCacheImpl) RemoveAbove5(ctx context.Context, date, tsCode string) error {
	key := fmt.Sprintf("pool:above5:%s", date)
	return RedisClient(ctx).SRem(ctx, key, tsCode).Err()
}

func (c PoolCacheImpl) GetAbove5Members(ctx context.Context, date string) ([]string, error) {
	key := fmt.Sprintf("pool:above5:%s", date)
	return RedisClient(ctx).SMembers(ctx, key).Result()
}

func (c PoolCacheImpl) SetFirstLimitTime(ctx context.Context, date, tsCode, limitTime string) error {
	key := fmt.Sprintf("first_limit:%s", date)
	return RedisClient(ctx).HSetNX(ctx, key, tsCode, limitTime).Err()
}

func (c PoolCacheImpl) GetFirstLimitTime(ctx context.Context, date, tsCode string) (string, error) {
	key := fmt.Sprintf("first_limit:%s", date)
	return RedisClient(ctx).HGet(ctx, key, tsCode).Result()
}
