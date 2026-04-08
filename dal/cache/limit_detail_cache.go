package cache

import (
	"context"
	"fmt"
)

// LimitDetail holds limit-up detail info from tushare limit_list_d API
type LimitDetail struct {
	FirstTime  string // 首次封板时间 "09:31:05"
	LastTime   string // 最后封板时间 "14:55:00"
	LimitTimes int    // 连板数
}

// LimitDetailCacheInterface defines the cache operations for limit details
type LimitDetailCacheInterface interface {
	Set(ctx context.Context, date string, details map[string]*LimitDetail) error
	Get(ctx context.Context, date, tsCode string) (*LimitDetail, error)
	GetAll(ctx context.Context, date string) (map[string]*LimitDetail, error)
}

var _ LimitDetailCacheInterface = (*LimitDetailCacheImpl)(nil)

// LimitDetailCacheImpl implements LimitDetailCacheInterface using go-cache
type LimitDetailCacheImpl struct{}

// NewLimitDetailCache creates a new LimitDetailCacheImpl
func NewLimitDetailCache() *LimitDetailCacheImpl {
	return &LimitDetailCacheImpl{}
}

// Set stores all limit details for a date
func (c *LimitDetailCacheImpl) Set(ctx context.Context, date string, details map[string]*LimitDetail) error {
	key := fmt.Sprintf("limit_detail:%s", date)
	Lock()
	defer Unlock()

	// Store a copy
	copied := make(map[string]*LimitDetail, len(details))
	for k, v := range details {
		detail := *v
		copied[k] = &detail
	}
	Cache.Set(key, copied, TTLUntilEndOfDay())
	return nil
}

// Get retrieves limit detail for a single stock
func (c *LimitDetailCacheImpl) Get(ctx context.Context, date, tsCode string) (*LimitDetail, error) {
	key := fmt.Sprintf("limit_detail:%s", date)
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(key); found {
		details := v.(map[string]*LimitDetail)
		if detail, exists := details[tsCode]; exists {
			// Return a copy
			copied := *detail
			return &copied, nil
		}
	}
	return nil, nil
}

// GetAll retrieves all limit details for a date
func (c *LimitDetailCacheImpl) GetAll(ctx context.Context, date string) (map[string]*LimitDetail, error) {
	key := fmt.Sprintf("limit_detail:%s", date)
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(key); found {
		details := v.(map[string]*LimitDetail)
		// Return a copy
		copied := make(map[string]*LimitDetail, len(details))
		for k, v := range details {
			detail := *v
			copied[k] = &detail
		}
		return copied, nil
	}
	return make(map[string]*LimitDetail), nil
}
