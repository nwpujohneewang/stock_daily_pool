package cache

import (
	"context"
	"fmt"
)

type YesterdayStrongEntry struct {
	TsCode             string
	YesterdayChangePct float64
}

type PoolCacheInterface interface {
	AddLimitUp(ctx context.Context, date, tsCode string) error
	RemoveLimitUp(ctx context.Context, date, tsCode string) error
	GetLimitUpMembers(ctx context.Context, date string) ([]string, error)
	IsLimitUp(ctx context.Context, date, tsCode string) (bool, error)
	AddAbove5(ctx context.Context, date, tsCode string) error
	RemoveAbove5(ctx context.Context, date, tsCode string) error
	GetAbove5Members(ctx context.Context, date string) ([]string, error)
	SetFirstLimitTime(ctx context.Context, date, tsCode, limitTime string) error
	GetFirstLimitTime(ctx context.Context, date string, tsCodes []string) (map[string]string, error)
	AddLimitUpBatch(ctx context.Context, date string, tsCodes []string) error
	RemoveLimitUpBatch(ctx context.Context, date string, tsCodes []string) error
	AddAbove5Batch(ctx context.Context, date string, tsCodes []string) error
	RemoveAbove5Batch(ctx context.Context, date string, tsCodes []string) error
	SetYesterdayStrongMembers(ctx context.Context, date string, entries []YesterdayStrongEntry) error
	GetYesterdayStrongMembers(ctx context.Context, date string) ([]YesterdayStrongEntry, error)
	IsYesterdayStrong(ctx context.Context, date, tsCode string) bool
	GetAllFirstLimitTime(ctx context.Context, date string) (map[string]string, error)
}

var _ PoolCacheInterface = (*PoolCacheImpl)(nil)

type PoolCacheImpl struct{}

func NewPoolCache() *PoolCacheImpl {
	return &PoolCacheImpl{}
}

func (c PoolCacheImpl) AddLimitUp(ctx context.Context, date, tsCode string) error {
	key := fmt.Sprintf("pool:limit_up:%s", date)
	Lock()
	defer Unlock()

	var set map[string]struct{}
	if v, found := Cache.Get(key); found {
		set = v.(map[string]struct{})
	} else {
		set = make(map[string]struct{})
	}
	set[tsCode] = struct{}{}
	Cache.Set(key, set, TTLUntilEndOfDay())
	return nil
}

func (c PoolCacheImpl) AddLimitUpBatch(ctx context.Context, date string, tsCodes []string) error {
	if len(tsCodes) == 0 {
		return nil
	}
	key := fmt.Sprintf("pool:limit_up:%s", date)
	Lock()
	defer Unlock()

	var set map[string]struct{}
	if v, found := Cache.Get(key); found {
		set = v.(map[string]struct{})
	} else {
		set = make(map[string]struct{})
	}
	for _, code := range tsCodes {
		set[code] = struct{}{}
	}
	Cache.Set(key, set, TTLUntilEndOfDay())
	return nil
}

func (c PoolCacheImpl) RemoveLimitUp(ctx context.Context, date, tsCode string) error {
	key := fmt.Sprintf("pool:limit_up:%s", date)
	Lock()
	defer Unlock()

	if v, found := Cache.Get(key); found {
		set := v.(map[string]struct{})
		delete(set, tsCode)
		Cache.Set(key, set, TTLUntilEndOfDay())
	}
	return nil
}

func (c PoolCacheImpl) RemoveLimitUpBatch(ctx context.Context, date string, tsCodes []string) error {
	if len(tsCodes) == 0 {
		return nil
	}
	key := fmt.Sprintf("pool:limit_up:%s", date)
	Lock()
	defer Unlock()

	if v, found := Cache.Get(key); found {
		set := v.(map[string]struct{})
		for _, code := range tsCodes {
			delete(set, code)
		}
		Cache.Set(key, set, TTLUntilEndOfDay())
	}
	return nil
}

func (c PoolCacheImpl) GetLimitUpMembers(ctx context.Context, date string) ([]string, error) {
	key := fmt.Sprintf("pool:limit_up:%s", date)
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(key); found {
		set := v.(map[string]struct{})
		members := make([]string, 0, len(set))
		for k := range set {
			members = append(members, k)
		}
		return members, nil
	}
	return []string{}, nil
}

func (c PoolCacheImpl) IsLimitUp(ctx context.Context, date, tsCode string) (bool, error) {
	key := fmt.Sprintf("pool:limit_up:%s", date)
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(key); found {
		set := v.(map[string]struct{})
		_, exists := set[tsCode]
		return exists, nil
	}
	return false, nil
}

func (c PoolCacheImpl) AddAbove5(ctx context.Context, date, tsCode string) error {
	key := fmt.Sprintf("pool:above5:%s", date)
	Lock()
	defer Unlock()

	var set map[string]struct{}
	if v, found := Cache.Get(key); found {
		set = v.(map[string]struct{})
	} else {
		set = make(map[string]struct{})
	}
	set[tsCode] = struct{}{}
	Cache.Set(key, set, TTLUntilEndOfDay())
	return nil
}

func (c PoolCacheImpl) AddAbove5Batch(ctx context.Context, date string, tsCodes []string) error {
	if len(tsCodes) == 0 {
		return nil
	}
	key := fmt.Sprintf("pool:above5:%s", date)
	Lock()
	defer Unlock()

	var set map[string]struct{}
	if v, found := Cache.Get(key); found {
		set = v.(map[string]struct{})
	} else {
		set = make(map[string]struct{})
	}
	for _, code := range tsCodes {
		set[code] = struct{}{}
	}
	Cache.Set(key, set, TTLUntilEndOfDay())
	return nil
}

func (c PoolCacheImpl) RemoveAbove5(ctx context.Context, date, tsCode string) error {
	key := fmt.Sprintf("pool:above5:%s", date)
	Lock()
	defer Unlock()

	if v, found := Cache.Get(key); found {
		set := v.(map[string]struct{})
		delete(set, tsCode)
		Cache.Set(key, set, TTLUntilEndOfDay())
	}
	return nil
}

func (c PoolCacheImpl) RemoveAbove5Batch(ctx context.Context, date string, tsCodes []string) error {
	if len(tsCodes) == 0 {
		return nil
	}
	key := fmt.Sprintf("pool:above5:%s", date)
	Lock()
	defer Unlock()

	if v, found := Cache.Get(key); found {
		set := v.(map[string]struct{})
		for _, code := range tsCodes {
			delete(set, code)
		}
		Cache.Set(key, set, TTLUntilEndOfDay())
	}
	return nil
}

func (c PoolCacheImpl) GetAbove5Members(ctx context.Context, date string) ([]string, error) {
	key := fmt.Sprintf("pool:above5:%s", date)
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(key); found {
		set := v.(map[string]struct{})
		members := make([]string, 0, len(set))
		for k := range set {
			members = append(members, k)
		}
		return members, nil
	}
	return []string{}, nil
}

func (c PoolCacheImpl) SetFirstLimitTime(ctx context.Context, date, tsCode, limitTime string) error {
	key := fmt.Sprintf("first_limit:%s", date)
	Lock()
	defer Unlock()

	var hashMap map[string]string
	if v, found := Cache.Get(key); found {
		hashMap = v.(map[string]string)
		// HSETNX: only set if not exists
		if _, exists := hashMap[tsCode]; exists {
			return nil
		}
	} else {
		hashMap = make(map[string]string)
	}
	hashMap[tsCode] = limitTime
	Cache.Set(key, hashMap, TTLUntilEndOfDay())
	return nil
}

func (c PoolCacheImpl) GetAllFirstLimitTime(ctx context.Context, date string) (map[string]string, error) {
	key := fmt.Sprintf("first_limit:%s", date)
	RLock()
	defer RUnlock()
	if v, found := Cache.Get(key); found {
		hashMap := v.(map[string]string)
		result := make(map[string]string, len(hashMap))
		for k, v := range hashMap {
			result[k] = v
		}
		return result, nil
	}
	return map[string]string{}, nil
}

func (c PoolCacheImpl) GetFirstLimitTime(ctx context.Context, date string, tsCodes []string) (map[string]string, error) {
	result := make(map[string]string, len(tsCodes))
	if len(tsCodes) == 0 {
		return result, nil
	}
	key := fmt.Sprintf("first_limit:%s", date)
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(key); found {
		hashMap := v.(map[string]string)
		for _, code := range tsCodes {
			if val, exists := hashMap[code]; exists {
				result[code] = val
			}
		}
	}
	return result, nil
}

func (c PoolCacheImpl) SetYesterdayStrongMembers(ctx context.Context, date string, entries []YesterdayStrongEntry) error {
	key := fmt.Sprintf("pool:yesterday_strong:%s", date)
	Lock()
	defer Unlock()

	entryMap := make(map[string]YesterdayStrongEntry)
	for _, e := range entries {
		entryMap[e.TsCode] = e
	}
	Cache.Set(key, entryMap, TTLUntilEndOfDay())
	return nil
}

func (c PoolCacheImpl) GetYesterdayStrongMembers(ctx context.Context, date string) ([]YesterdayStrongEntry, error) {
	key := fmt.Sprintf("pool:yesterday_strong:%s", date)
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(key); found {
		entryMap := v.(map[string]YesterdayStrongEntry)
		entries := make([]YesterdayStrongEntry, 0, len(entryMap))
		for _, e := range entryMap {
			entries = append(entries, e)
		}
		return entries, nil
	}
	return []YesterdayStrongEntry{}, nil
}

func (c PoolCacheImpl) IsYesterdayStrong(ctx context.Context, date, tsCode string) bool {
	key := fmt.Sprintf("pool:yesterday_strong:%s", date)
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(key); found {
		entryMap := v.(map[string]YesterdayStrongEntry)
		_, exists := entryMap[tsCode]
		return exists
	}
	return false
}
