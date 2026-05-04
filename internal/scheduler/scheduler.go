package scheduler

import (
	"context"
	"stock/config"
	"stock/dal/cache"
	"stock/dal/repo"
	"stock/external/tushare"
	"stock/internal/pkg/logger"
	"stock/internal/pkg/utils"
	"stock/internal/service"
	"stock/internal/service/snapshot"
	"stock/internal/service/stock"
	"time"

	"github.com/robfig/cron/v3"

	"go.uber.org/zap"
)

type Scheduler struct {
	cron           *cron.Cron
	cfg            *config.SchedulerConfig
	monitorService service.MonitorServiceInterface
}

func NewScheduler(
	cfg *config.SchedulerConfig,
	monitorService service.MonitorServiceInterface,
) *Scheduler {
	return &Scheduler{
		cron:           cron.New(cron.WithSeconds()),
		cfg:            cfg,
		monitorService: monitorService,
	}
}

func (s *Scheduler) Setup() {
	if _, err := s.cron.AddFunc("0 0 9 * * 1-5", s.preMarketInit); err != nil {
		logger.Warn("register preMarketInit failed", zap.Error(err))
	}
	if _, err := s.cron.AddFunc("0 5 15 * * 1-5", s.closingSnapshot); err != nil {
		logger.Warn("register closingSnapshot failed", zap.Error(err))
	}
	if _, err := s.cron.AddFunc("0 0 23 * * 1-5", s.computeTopicStockCount); err != nil {
		logger.Warn("register computeTopicStockCount failed", zap.Error(err))
	}
	if _, err := s.cron.AddFunc("0 0 18 * * 1-5", s.syncStockBasic); err != nil {
		logger.Warn("register syncStockBasic failed", zap.Error(err))
	}
}

func (s *Scheduler) Start() {
	s.cron.Start()
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}

func (s *Scheduler) TriggerPreMarketInit() {
	s.preMarketInit()
}

func (s *Scheduler) TriggerLoadYesterdayStrongPool() {
	s.loadYesterdayStrongPool()
}

func (s *Scheduler) TriggerClosingSnapshot() {
	s.closingSnapshot()
}

func (s *Scheduler) TriggerSyncStockBasic() {
	s.syncStockBasic()
}

func (s *Scheduler) TriggerComputeTopicStockCount() {
	s.computeTopicStockCount()
}

func (s *Scheduler) preMarketInit() {
	ctx := context.Background()
	date := time.Now().Format("2006-01-02")

	if !isTradingDay(time.Now()) {
		return
	}

	logger.Info("running pre-market init", zap.String("date", date))

	// Reset monitor daily state
	if s.monitorService != nil {
		s.monitorService.ResetClassifiedToday()
		logger.Info("reset monitor daily state")
	}

	poolCache := cache.NewPoolCache()
	_ = poolCache.RemoveLimitUp(ctx, getPrevDate(date), "")
	_ = poolCache.RemoveAbove5(ctx, getPrevDate(date), "")

	stockRepo := repo.NewStockRepository()
	stocks, err := stockRepo.GetActiveStocks(ctx)
	if err != nil {
		logger.Warn("get active stocks failed", zap.Error(err))
		return
	}

	logger.Info("loaded active stocks", zap.Int("count", len(stocks)))

	stockCache := cache.NewStockCache()
	if err := stockCache.SetActiveStocks(ctx, stocks); err != nil {
		logger.Warn("set active stocks cache failed", zap.Error(err))
	}

	tsCodes := make([]string, 0, len(stocks))
	for _, stock := range stocks {
		if stock.TsCode != "" {
			tsCodes = append(tsCodes, stock.TsCode)
		}
	}
	if err := repo.NewStockTopicRelationRepository().WarmupByTsCodes(ctx, tsCodes); err != nil {
		logger.Warn("warmup stock topic relations cache failed", zap.Error(err), zap.Int("count", len(tsCodes)))
	}

	if err := WarmupTopicStockCountCache(ctx); err != nil {
		logger.Warn("warmup topic stock count cache failed", zap.Error(err))
	}

	s.loadYesterdayStrongPool()
}

func (s *Scheduler) isWithinTradingWindow(t time.Time) bool {
	h, m := t.Hour(), t.Minute()
	total := h*60 + m
	start := 9*60 + 25
	end := 15*60 + 1
	return total >= start && total <= end
}

func (s *Scheduler) closingSnapshot() {
	ctx := context.Background()
	loc, _ := time.LoadLocation("Asia/Shanghai")
	date := time.Now().In(loc).Format("2006-01-02")

	if !isTradingDay(time.Now()) {
		return
	}

	logger.Info("running closing snapshot", zap.String("date", date))

	snapshotSvc := snapshot.NewSnapshotService()
	if err := snapshotSvc.TakeSnapshot(ctx, date); err != nil {
		logger.Error("closing snapshot failed", zap.Error(err))
		return
	}

	logger.Info("closing snapshot completed", zap.String("date", date))
}

func (s *Scheduler) computeTopicStockCount() {
	ctx := context.Background()
	loc, _ := time.LoadLocation("Asia/Shanghai")
	date := time.Now().In(loc).Format("2006-01-02")

	if !isTradingDay(time.Now()) {
		return
	}

	logger.Info("running topic stock count compute", zap.String("date", date))
	if err := ComputeTopicStockCount(ctx); err != nil {
		logger.Error("topic stock count compute failed", zap.Error(err))
		return
	}

	logger.Info("topic stock count compute completed", zap.String("date", date))
}

func (s *Scheduler) cacheWarmup() {
	logger.Info("cache warmup task")
}

func (s *Scheduler) syncStockBasic() {
	ctx := context.Background()
	loc, _ := time.LoadLocation("Asia/Shanghai")
	date := time.Now().In(loc).Format("2006-01-02")

	if !isTradingDay(time.Now()) {
		return
	}

	logger.Info("running stock basic sync", zap.String("date", date))

	cfg := config.GlobalConfig
	tushareClient := tushare.NewClient(&cfg.Tushare, cfg.Retry)
	svc := stock.NewStockService(tushareClient)

	if err := svc.SyncStockBasic(ctx, date); err != nil {
		logger.Error("stock basic sync failed", zap.Error(err))
		return
	}

	logger.Info("stock basic sync completed", zap.String("date", date))
}

func isTradingDay(t time.Time) bool {
	return utils.IsTradingDay(t)
}

func getPrevDate(date string) string {
	t, _ := time.Parse("2006-01-02", date)
	return utils.PreviousTradingDay(t).Format("2006-01-02")
}

func getPreviousTradingDay(t time.Time) time.Time {
	return utils.PreviousTradingDay(t)
}

func (s *Scheduler) loadYesterdayStrongPool() {
	ctx := context.Background()
	loc, _ := time.LoadLocation("Asia/Shanghai")
	today := time.Now().In(loc)
	yesterday := getPreviousTradingDay(today)

	logger.Info("loading yesterday strong pool",
		zap.String("today", today.Format("2006-01-02")),
		zap.String("yesterday", yesterday.Format("2006-01-02")))

	snapshotRepo := repo.NewSnapshotRepository()
	records, err := snapshotRepo.GetByDate(ctx, yesterday.Format("2006-01-02"))
	if err != nil {
		logger.Warn("get yesterday snapshot failed", zap.Error(err))
		return
	}

	// Include both yesterday's limit-up and above-5% stocks in yesterday strong pool.
	entries := make([]cache.YesterdayStrongEntry, 0, len(records))
	for _, r := range records {
		if r.ChangePct == nil {
			continue
		}
		entries = append(entries, cache.YesterdayStrongEntry{
			TsCode:             r.TsCode,
			YesterdayChangePct: *r.ChangePct,
		})
	}

	poolCache := cache.NewPoolCache()
	if err := poolCache.SetYesterdayStrongMembers(ctx, today.Format("2006-01-02"), entries); err != nil {
		logger.Error("set yesterday strong members failed", zap.Error(err))
		return
	}

	logger.Info("loaded yesterday strong pool", zap.Int("count", len(entries)))
}

func (s *Scheduler) GetCron() *cron.Cron {
	return s.cron
}
