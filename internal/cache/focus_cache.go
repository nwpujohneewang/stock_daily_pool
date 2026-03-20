package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type FocusCache struct {
	client *redis.Client
}

func NewFocusCache(client *redis.Client) *FocusCache {
	return &FocusCache{client: client}
}

func (c *FocusCache) SetFocusTopics(ctx context.Context, date string, topicIDs []int64) error {
	key := "focus:topics:" + date
	var members []interface{}
	for _, id := range topicIDs {
		members = append(members, id)
	}
	if len(members) == 0 {
		return nil
	}
	return c.client.SAdd(ctx, key, members...).Err()
}

func (c *FocusCache) GetFocusTopics(ctx context.Context, date string) ([]int64, error) {
	key := "focus:topics:" + date
	members, err := c.client.SMembers(ctx, key).Result()
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

func (c *FocusCache) IsFocused(ctx context.Context, date string, topicID int64) (bool, error) {
	key := "focus:topics:" + date
	return c.client.SIsMember(ctx, key, topicID).Result()
}

func (c *FocusCache) RemoveFocusTopic(ctx context.Context, date string, topicID int64) error {
	key := "focus:topics:" + date
	return c.client.SRem(ctx, key, topicID).Err()
}
