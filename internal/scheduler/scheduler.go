package scheduler

import (
	"context"
	"log"
	"stock/dal/redis"
	"time"

	"github.com/robfig/cron/v3"
	"stock/config"
	"stock/dal/db"
	"stock/internal/service"
)

type Scheduler struct {
	cron           *cron.Cron
	cfg            *config.SchedulerConfig
	monitorService *service.MonitorService
	logger         *log.Logger
}

func NewScheduler(
	cfg *config.SchedulerConfig,
	monitorService *service.MonitorService,
) *Scheduler {
	return &Scheduler{
		cron:           cron.New(cron.WithSeconds()),
		cfg:            cfg,
		monitorService: monitorService,
		logger:         log.Default(),
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

	s.logger.Printf("running pre-market init for %s", date)

	poolCache := redis.NewPoolCache()
	_ = poolCache.RemoveLimitUp(ctx, getPrevDate(date), "")
	_ = poolCache.RemoveAbove5(ctx, getPrevDate(date), "")

	stockRepo := db.NewStockRepository()
	stocks, err := stockRepo.GetActiveStocks(ctx)
	if err != nil {
		s.logger.Printf("get active stocks: %v", err)
		return
	}

	s.logger.Printf("loaded %d active stocks", len(stocks))
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
		s.logger.Printf("realtime collect: %v", err)
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
	s.logger.Println("jiuyan sync task")
}

func (s *Scheduler) conceptSync() {
	s.logger.Println("concept sync task")
}

func (s *Scheduler) closingSnapshot() {
	date := time.Now().Format("2006-01-02")

	if !isTradingDay(time.Now()) {
		return
	}

	s.logger.Printf("closing snapshot for %s", date)
}

func (s *Scheduler) cacheWarmup() {
	s.logger.Println("cache warmup task")
}

func (s *Scheduler) historyCleanup() {
	s.logger.Println("history cleanup task")
}

func (s *Scheduler) llmBatch() {
	s.logger.Println("LLM batch task")
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
