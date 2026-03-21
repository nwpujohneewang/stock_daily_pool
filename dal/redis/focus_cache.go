package redis

import (
	"context"
	"fmt"
)

type FocusCacheInterface interface {
	SetFocusTopics(ctx context.Context, date string, topicIDs []int64) error
	GetFocusTopics(ctx context.Context, date string) ([]int64, error)
	IsFocused(ctx context.Context, date string, topicID int64) (bool, error)
	RemoveFocusTopic(ctx context.Context, date string, topicID int64) error
}

var _ FocusCacheInterface = (*FocusCacheImpl)(nil)

type FocusCacheImpl struct{}

func NewFocusCache() *FocusCacheImpl {
	return &FocusCacheImpl{}
}

func (c FocusCacheImpl) SetFocusTopics(ctx context.Context, date string, topicIDs []int64) error {
	key := "focus:topics:" + date
	var members []interface{}
	for _, id := range topicIDs {
		members = append(members, id)
	}
	if len(members) == 0 {
		return nil
	}
	return RedisClient(ctx).SAdd(ctx, key, members...).Err()
}

func (c FocusCacheImpl) GetFocusTopics(ctx context.Context, date string) ([]int64, error) {
	key := "focus:topics:" + date
	members, err := RedisClient(ctx).SMembers(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var ids []int64
	for _, m := range members {
		var id int64
		fmt.Sscanf(m, "%d", &id)
		ids = append(ids, id)
	}
	return ids, nil
}

func (c FocusCacheImpl) IsFocused(ctx context.Context, date string, topicID int64) (bool, error) {
	key := "focus:topics:" + date
	return RedisClient(ctx).SIsMember(ctx, key, topicID).Result()
}

func (c FocusCacheImpl) RemoveFocusTopic(ctx context.Context, date string, topicID int64) error {
	key := "focus:topics:" + date
	return RedisClient(ctx).SRem(ctx, key, topicID).Err()
}
