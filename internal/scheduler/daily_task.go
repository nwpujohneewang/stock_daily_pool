package scheduler

import (
	"context"
	"log"
	"time"

	"stock/internal/cache"
	"stock/internal/repo"
	"stock/internal/service"
)

type DailyTask struct {
	stockService    *service.StockService
	crawlerService  *service.CrawlerService
	conceptService  *service.ConceptSyncService
	snapshotService *service.SnapshotService
	boardRepo       *repo.BoardRepo
	poolCache       *cache.PoolCache
	conceptCache    *cache.ConceptCache
	mappingCache    *cache.MappingCache
	focusCache      *cache.FocusCache
	logger          *log.Logger
}

func NewDailyTask(
	stockService *service.StockService,
	crawlerService *service.CrawlerService,
	conceptService *service.ConceptSyncService,
	snapshotService *service.SnapshotService,
	boardRepo *repo.BoardRepo,
	poolCache *cache.PoolCache,
	conceptCache *cache.ConceptCache,
	mappingCache *cache.MappingCache,
	focusCache *cache.FocusCache,
) *DailyTask {
	return &DailyTask{
		stockService:    stockService,
		crawlerService:  crawlerService,
		conceptService:  conceptService,
		snapshotService: snapshotService,
		boardRepo:       boardRepo,
		poolCache:       poolCache,
		conceptCache:    conceptCache,
		mappingCache:    mappingCache,
		focusCache:      focusCache,
		logger:          log.Default(),
	}
}

func (d *DailyTask) PreMarketInit(ctx context.Context) error {
	d.logger.Println("running pre-market init")
	return nil
}

func (d *DailyTask) CacheWarmup(ctx context.Context) error {
	d.logger.Println("running cache warmup")
	return nil
}

func (d *DailyTask) JiuyanSync(ctx context.Context) error {
	d.logger.Println("running jiuyan sync")
	return nil
}

func (d *DailyTask) ConceptSync(ctx context.Context) error {
	d.logger.Println("running concept sync")
	return nil
}

func (d *DailyTask) ClosingSnapshot(ctx context.Context) error {
	d.logger.Println("running closing snapshot")
	date := time.Now().Format("2006-01-02")
	return d.snapshotService.TakeSnapshot(ctx, date)
}

func (d *DailyTask) IsTradingDay() bool {
	now := time.Now()
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return false
	}
	return true
}
