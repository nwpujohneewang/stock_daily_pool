package cache

import (
	"context"
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
	if len(topicIDs) == 0 {
		return nil
	}
	key := "focus:topics:" + date
	Lock()
	defer Unlock()

	var set map[int64]struct{}
	if v, found := Cache.Get(key); found {
		set = v.(map[int64]struct{})
	} else {
		set = make(map[int64]struct{})
	}
	for _, id := range topicIDs {
		set[id] = struct{}{}
	}
	Cache.Set(key, set, TTLUntilEndOfDay())
	return nil
}

func (c FocusCacheImpl) GetFocusTopics(ctx context.Context, date string) ([]int64, error) {
	key := "focus:topics:" + date
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(key); found {
		set := v.(map[int64]struct{})
		ids := make([]int64, 0, len(set))
		for id := range set {
			ids = append(ids, id)
		}
		return ids, nil
	}
	return []int64{}, nil
}

func (c FocusCacheImpl) IsFocused(ctx context.Context, date string, topicID int64) (bool, error) {
	key := "focus:topics:" + date
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(key); found {
		set := v.(map[int64]struct{})
		_, exists := set[topicID]
		return exists, nil
	}
	return false, nil
}

func (c FocusCacheImpl) RemoveFocusTopic(ctx context.Context, date string, topicID int64) error {
	key := "focus:topics:" + date
	Lock()
	defer Unlock()

	if v, found := Cache.Get(key); found {
		set := v.(map[int64]struct{})
		delete(set, topicID)
		Cache.Set(key, set, TTLUntilEndOfDay())
	}
	return nil
}
