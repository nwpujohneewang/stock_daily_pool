package cache

import (
	"sync"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

var (
	// Cache is the global go-cache instance
	Cache *gocache.Cache
	// shanghaiLoc is the Asia/Shanghai timezone
	shanghaiLoc *time.Location
	// mu protects concurrent modifications to map-type values
	mu sync.RWMutex
)

// Init initializes the global cache instance
func Init() {
	var err error
	shanghaiLoc, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		panic("failed to load Asia/Shanghai timezone: " + err.Error())
	}
	// Default expiration 24h, cleanup interval 10 minutes
	Cache = gocache.New(24*time.Hour, 10*time.Minute)
}

// TTLUntilEndOfDay calculates the duration until 23:59 Shanghai time
func TTLUntilEndOfDay() time.Duration {
	now := time.Now().In(shanghaiLoc)
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 0, 0, shanghaiLoc)
	ttl := endOfDay.Sub(now)
	if ttl < 0 {
		ttl = time.Minute // If past 23:59, expire in 1 minute
	}
	return ttl
}

// Lock acquires write lock for map operations
func Lock() {
	mu.Lock()
}

// Unlock releases write lock
func Unlock() {
	mu.Unlock()
}

// RLock acquires read lock for map operations
func RLock() {
	mu.RLock()
}

// RUnlock releases read lock
func RUnlock() {
	mu.RUnlock()
}
