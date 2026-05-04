package limitdetail

import (
	"context"
	"stock/external/tushare"
	"strings"
	"sync"
	"time"

	"stock/dal/cache"
	"stock/internal/pkg/logger"

	"go.uber.org/zap"
)

var (
	instance *LimitDetailService
	once     sync.Once
)

// LimitDetailService handles fetching and caching limit-up details
type LimitDetailService struct {
	tushareClient *tushare.Client
	cache         cache.LimitDetailCacheInterface
	lastRefresh   time.Time
	mu            sync.Mutex
}

// Init initializes the singleton service
func Init(tushareClient *tushare.Client) {
	once.Do(func() {
		instance = &LimitDetailService{
			tushareClient: tushareClient,
			cache:         cache.NewLimitDetailCache(),
		}
	})
}

// GetInstance returns the singleton instance
func GetInstance() *LimitDetailService {
	return instance
}

// RefreshLimitDetails fetches limit_list_d data and updates cache
func (s *LimitDetailService) RefreshLimitDetails(ctx context.Context, date string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Convert date format: "2026-03-29" -> "20260329"
	tradeDate := strings.ReplaceAll(date, "-", "")

	items, err := s.tushareClient.LimitListD(ctx, tradeDate)
	if err != nil {
		logger.Warn("fetch limit_list_d failed", zap.String("date", date), zap.Error(err))
		return err
	}

	details := make(map[string]*cache.LimitDetail, len(items))
	for _, item := range items {
		details[item.TsCode] = &cache.LimitDetail{
			FirstTime:  item.FirstTime,
			LastTime:   item.LastTime,
			LimitTimes: item.LimitTimes,
		}
	}

	if err := s.cache.Set(ctx, date, details); err != nil {
		logger.Warn("set limit detail cache failed", zap.String("date", date), zap.Error(err))
		return err
	}

	s.lastRefresh = time.Now()
	logger.Info("refreshed limit details", zap.String("date", date), zap.Int("count", len(details)))
	return nil
}

// ShouldRefresh checks if enough time has passed since last refresh (5 minutes)
func (s *LimitDetailService) ShouldRefresh() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return time.Since(s.lastRefresh) >= 5*time.Minute
}

// QueryLimitDetails fetches limit-up details for a date via tushare LimitListD and returns a map keyed by ts_code.
// It also populates the in-memory cache for the given date for faster subsequent access.
func (s *LimitDetailService) QueryLimitDetails(ctx context.Context, date string) (map[string]*cache.LimitDetail, error) {
	// Convert date format: "YYYY-MM-DD" -> "YYYYMMDD"
	tradeDate := strings.ReplaceAll(date, "-", "")

	items, err := s.tushareClient.LimitListD(ctx, tradeDate)
	if err != nil {
		logger.Warn("query limit_list_d failed", zap.String("date", date), zap.Error(err))
		return nil, err
	}

	details := make(map[string]*cache.LimitDetail, len(items))
	for _, item := range items {
		details[item.TsCode] = &cache.LimitDetail{
			FirstTime:  item.FirstTime,
			LastTime:   item.LastTime,
			LimitTimes: item.LimitTimes,
		}
	}

	return details, nil
}

// GetTodayFirstLimitTimes returns a map of ts_code -> first limit-up time for the given date.
// It reads from the pool cache which is populated during intraday monitoring via SetFirstLimitTime.
func (s *LimitDetailService) GetTodayFirstLimitTimes(ctx context.Context, date string) (map[string]string, error) {
	poolCache := cache.NewPoolCache()
	return poolCache.GetAllFirstLimitTime(ctx, date)
}
