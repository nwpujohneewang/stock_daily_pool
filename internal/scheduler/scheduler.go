package scheduler

import (
	"context"
	"stock/dal/redis"
	"time"

	"github.com/robfig/cron/v3"
	"stock/config"
	"stock/dal/db"
	"stock/internal/pkg/logger"
	"stock/internal/service"

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
	s.cron.AddFunc(s.cfg.PreMarketInit, s.preMarketInit)
	s.cron.AddFunc(s.cfg.RealtimeCollect, s.realtimeCollect)
	s.cron.AddFunc(s.cfg.JiuyanSync, s.jiuyanSync)
	s.cron.AddFunc(s.cfg.ConceptSync, s.conceptSync)
	s.cron.AddFunc(s.cfg.ClosingSnapshot, s.closingSnapshot)
	s.cron.AddFunc(s.cfg.CacheWarmup, s.cacheWarmup)
	s.cron.AddFunc(s.cfg.HistoryCleanup, s.historyCleanup)
	s.cron.AddFunc(s.cfg.LLMBatch, s.llmBatch)
}

func (s *Scheduler) Start() {
	s.cron.Start()
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}

func (s *Scheduler) preMarketInit() {
	ctx := context.Background()
	date := time.Now().Format("2006-01-02")

	if !isTradingDay(time.Now()) {
		return
	}

	logger.Info("running pre-market init", zap.String("date", date))

	poolCache := redis.NewPoolCache()
	_ = poolCache.RemoveLimitUp(ctx, getPrevDate(date), "")
	_ = poolCache.RemoveAbove5(ctx, getPrevDate(date), "")

	stockRepo := db.NewStockRepository()
	stocks, err := stockRepo.GetActiveStocks(ctx)
	if err != nil {
		logger.Warn("get active stocks failed", zap.Error(err))
		return
	}

	logger.Info("loaded active stocks", zap.Int("count", len(stocks)))
}

func (s *Scheduler) realtimeCollect() {
	ctx := context.Background()
	now := time.Now()

	if !isTradingDay(now) {
		return
	}

	if !s.isWithinTradingWindow(now) {
		return
	}

	date := now.Format("2006-01-02")
	if err := s.monitorService.ProcessTick(ctx, date); err != nil {
		logger.Warn("realtime collect failed", zap.Error(err))
	}
}

func (s *Scheduler) isWithinTradingWindow(t time.Time) bool {
	h, m := t.Hour(), t.Minute()
	total := h*60 + m
	start := 9*60 + 25
	end := 15*60 + 1
	return total >= start && total <= end
}

func (s *Scheduler) jiuyanSync() {
	logger.Info("jiuyan sync task")
}

func (s *Scheduler) conceptSync() {
	logger.Info("concept sync task")
}

func (s *Scheduler) closingSnapshot() {
	date := time.Now().Format("2006-01-02")

	if !isTradingDay(time.Now()) {
		return
	}

	logger.Info("closing snapshot", zap.String("date", date))
}

func (s *Scheduler) cacheWarmup() {
	logger.Info("cache warmup task")
}

func (s *Scheduler) historyCleanup() {
	logger.Info("history cleanup task")
}

func (s *Scheduler) llmBatch() {
	logger.Info("LLM batch task")
}

func isTradingDay(t time.Time) bool {
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return false
	}
	return true
}

func getPrevDate(date string) string {
	t, _ := time.Parse("2006-01-02", date)
	return t.AddDate(0, 0, -1).Format("2006-01-02")
}

func (s *Scheduler) GetCron() *cron.Cron {
	return s.cron
}
