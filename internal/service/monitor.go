package service

import (
	"context"
	"fmt"
	"stock/config"
	"stock/dal/db"
	"stock/dal/redis"
	"stock/internal/external/tushare"
	"stock/internal/pkg/limiter"
	"stock/internal/pkg/logger"
	"stock/internal/pkg/shard"
	"stock/model/dal_model"
	"sync"
	"time"

	"go.uber.org/zap"
)

type MonitorServiceImpl struct {
	tushareClient *tushare.Client
	cfg           *config.MonitorConfig
	boardRules    map[dal_model.BoardCode]*dal_model.BoardRule
	stockStates   map[string]int
	mu            sync.RWMutex
}

var _ MonitorServiceInterface = (*MonitorServiceImpl)(nil)

func NewMonitorService(tushareClient *tushare.Client, cfg *config.MonitorConfig) *MonitorServiceImpl {
	return &MonitorServiceImpl{
		tushareClient: tushareClient,
		cfg:           cfg,
		boardRules:    initBoardRules(),
		stockStates:   make(map[string]int),
	}
}

func initBoardRules() map[dal_model.BoardCode]*dal_model.BoardRule {
	return map[dal_model.BoardCode]*dal_model.BoardRule{
		dal_model.BoardMain: {BoardCode: dal_model.BoardMain, BoardName: "主板", LimitUpRatio: 0.10, LimitDownRatio: -0.10},
		dal_model.BoardGEM:  {BoardCode: dal_model.BoardGEM, BoardName: "创业板", LimitUpRatio: 0.20, LimitDownRatio: -0.20},
		dal_model.BoardSTAR: {BoardCode: dal_model.BoardSTAR, BoardName: "科创板", LimitUpRatio: 0.20, LimitDownRatio: -0.20},
		dal_model.BoardBSE:  {BoardCode: dal_model.BoardBSE, BoardName: "北交所", LimitUpRatio: 0.30, LimitDownRatio: -0.30},
	}
}

func (s *MonitorServiceImpl) DetectBoard(tsCode string) dal_model.BoardCode {
	return limiter.DetectBoard(tsCode[:6])
}

func (s *MonitorServiceImpl) ProcessTick(ctx context.Context, date string) error {
	quoteFetcher := NewQuoteFetcher(s.tushareClient, s.cfg.ShardCount)
	if err := quoteFetcher.FetchAllQuotes(ctx); err != nil {
		logger.Warn("fetch quotes failed", zap.Error(err))
	}

	stockRepo := db.NewStockRepository()
	stocks, err := stockRepo.GetActiveStocks(ctx)
	if err != nil {
		return fmt.Errorf("get active stocks: %w", err)
	}

	tsCodes := make([]string, len(stocks))
	for i, stock := range stocks {
		tsCodes[i] = stock.TsCode
	}

	batches := shard.BuildShardBatches(tsCodes, s.cfg.ShardCount)

	var wg sync.WaitGroup
	for _, batch := range batches {
		wg.Add(1)
		go func(batch shard.ShardBatch) {
			defer wg.Done()
			s.processShard(ctx, date, batch)
		}(batch)
	}
	wg.Wait()

	return nil
}

func (s *MonitorServiceImpl) processShard(ctx context.Context, date string, batch shard.ShardBatch) {
	stockRepo := db.NewStockRepository()

	for _, tsCode := range batch.TsCodes {
		stock, err := stockRepo.GetByTsCode(ctx, tsCode)
		if err != nil {
			logger.Warn("get stock failed", zap.String("ts_code", tsCode), zap.Error(err))
			continue
		}

		if stock.IsST {
			continue
		}

		quoteCache := redis.NewQuoteCache()
		quote, err := quoteCache.Get(ctx, tsCode)
		if err != nil || quote == nil {
			continue
		}

		board := s.DetectBoard(tsCode)
		rule := s.boardRules[board]

		s.mu.RLock()
		prevState := s.stockStates[tsCode]
		s.mu.RUnlock()

		input := dal_model.DetectInput{
			TsCode:       tsCode,
			StockName:    stock.Name,
			CurrentPrice: quote.Price,
			PreClose:     quote.PreClose,
			ChangePct:    quote.PctChg,
			Volume:       float64(quote.Vol),
			BoardCode:    board,
			IsST:         stock.IsST,
			QuoteTime:    quote.UpdateTime,
			PrevState:    prevState,
		}

		output := limiter.DetectLimitUp(input, rule)

		s.mu.Lock()
		s.stockStates[tsCode] = output.CurrentState
		s.mu.Unlock()

		if output.IsLimitUp {
			poolCache := redis.NewPoolCache()
			poolCache.AddLimitUp(ctx, date, tsCode)
			if output.IsFirstLimitUp {
				poolCache.SetFirstLimitTime(ctx, date, tsCode, output.FirstLimitTime.Format("15:04:05"))
				s.triggerClassifyAndAlert(ctx, date, tsCode, stock.Name, quote, output)
			}
		} else if output.IsAbove5Pct {
			poolCache := redis.NewPoolCache()
			poolCache.AddAbove5(ctx, date, tsCode)
		}
	}
}

func (s *MonitorServiceImpl) triggerClassifyAndAlert(ctx context.Context, date, tsCode, stockName string, quote *dal_model.StockQuote, output dal_model.DetectOutput) {
	go func() {
		classifyCtx := context.Background()
		classifySvc := NewClassifyService()
		topics, err := classifySvc.ClassifyStock(classifyCtx, tsCode, date, quote.UpdateTime)
		if err != nil {
			logger.Warn("classify stock failed", zap.String("ts_code", tsCode), zap.Error(err))
			return
		}
		if len(topics) == 0 {
			return
		}

		prevDate := prevTradingDay(date)
		poolRepo := db.NewPoolRepository()
		prevPool, err := poolRepo.GetByTsCodeAndDate(classifyCtx, prevDate, tsCode)
		if err != nil {
			logger.Warn("get prev pool failed", zap.String("ts_code", tsCode), zap.String("prev_date", prevDate), zap.Error(err))
		}

		var prevDayPct float64
		var prevDayAbove5 bool
		var prevDayLimitUp bool
		if prevPool != nil {
			if prevPool.ChangePct != nil {
				prevDayPct = *prevPool.ChangePct
			}
			prevDayLimitUp = prevPool.PoolType == dal_model.PoolTypeLimitUp
			prevDayAbove5 = prevPool.PoolType == dal_model.PoolTypeAbove5 || prevPool.PoolType == dal_model.PoolTypeLimitUp
		}

		topicIDs := make([]int64, len(topics))
		topicNames := make([]string, len(topics))
		for i, t := range topics {
			topicIDs[i] = t.TopicID
			topicNames[i] = t.TopicName
		}
		alertInput := AlertCheckInput{
			TsCode:         tsCode,
			StockName:      stockName,
			TopicIDs:       topicIDs,
			TopicNames:     topicNames,
			TriggerPrice:   output.LimitUpPrice,
			TriggerTime:    *output.FirstLimitTime,
			Date:           date,
			ChangePct:      quote.PctChg,
			PrevDayPct:     prevDayPct,
			PrevDayAbove5:  prevDayAbove5,
			PrevDayLimitUp: prevDayLimitUp,
		}
		alertSvc := NewAlertService(s.cfg)
		if _, err := alertSvc.CheckAndAlert(classifyCtx, alertInput); err != nil {
			logger.Warn("check alert failed", zap.String("ts_code", tsCode), zap.Error(err))
		}
	}()
}

func prevTradingDay(date string) string {
	t, _ := time.Parse("2006-01-02", date)
	t = t.AddDate(0, 0, -1)
	return t.Format("2006-01-02")
}

func (s *MonitorServiceImpl) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(s.cfg.IntervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			if !s.isTradingTime(now) {
				continue
			}
			date := now.Format("2006-01-02")
			if err := s.ProcessTick(ctx, date); err != nil {
				logger.Warn("process tick failed", zap.Error(err))
			}
		}
	}
}

func (s *MonitorServiceImpl) isTradingTime(t time.Time) bool {
	h, m := t.Hour(), t.Minute()
	total := h*60 + m
	start := 9*60 + 25
	end := 15*60 + 1
	return total >= start && total <= end
}
