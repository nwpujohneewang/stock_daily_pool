package monitor

import (
	"context"
	"stock/internal/pkg/utils"
	"sync"
	"time"

	"stock/config"
	"stock/dal/cache"
	"stock/dal/repo"
	"stock/internal/external/tushare"
	"stock/internal/pkg/limiter"
	"stock/internal/pkg/logger"
	"stock/internal/service/classify"
	"stock/internal/service/quote_fetcher"
	"stock/model/dal_model"

	"go.uber.org/zap"
)

var instance *MonitorServiceImpl

type MonitorServiceImpl struct {
	tushareClient   *tushare.Client
	cfg             *config.MonitorConfig
	boardRules      map[dal_model.BoardCode]*dal_model.BoardRule
	stockStates     map[string]int
	classifiedToday map[string]bool
	notifyCallback  func(date string)
	running         bool
	lastTickAt      time.Time
	stocksProcessed int
	mu              sync.Mutex
	cancelFunc      context.CancelFunc
}

type MonitorStatus struct {
	Running         bool      `json:"running"`
	LastTickAt      time.Time `json:"last_tick_at"`
	StocksProcessed int       `json:"stocks_processed"`
}

func (s *MonitorServiceImpl) SetNotifyCallback(fn func(date string)) {
	s.notifyCallback = fn
}

func NewMonitorService(tushareClient *tushare.Client, cfg *config.MonitorConfig) *MonitorServiceImpl {
	return &MonitorServiceImpl{
		tushareClient:   tushareClient,
		cfg:             cfg,
		boardRules:      initBoardRules(),
		stockStates:     make(map[string]int),
		classifiedToday: make(map[string]bool),
	}
}

func Init(tushareClient *tushare.Client, cfg *config.MonitorConfig) {
	instance = &MonitorServiceImpl{
		tushareClient:   tushareClient,
		cfg:             cfg,
		boardRules:      initBoardRules(),
		stockStates:     make(map[string]int),
		classifiedToday: make(map[string]bool),
	}
}

func GetInstance() *MonitorServiceImpl {
	return instance
}

func (s *MonitorServiceImpl) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *MonitorServiceImpl) GetStatus() MonitorStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return MonitorStatus{
		Running:         s.running,
		LastTickAt:      s.lastTickAt,
		StocksProcessed: s.stocksProcessed,
	}
}

func (s *MonitorServiceImpl) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancelFunc != nil {
		s.cancelFunc()
		s.cancelFunc = nil
	}
	s.running = false
}

func (s *MonitorServiceImpl) ResetClassifiedToday() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.classifiedToday = make(map[string]bool)
	s.stockStates = make(map[string]int)
}

func (s *MonitorServiceImpl) ManualTick(ctx context.Context, date string) error {
	err := s.ProcessTick(ctx, date)
	s.mu.Lock()
	s.lastTickAt = time.Now()
	s.mu.Unlock()
	return err
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
	quoteFetcher := quote_fetcher.NewQuoteFetcher(s.tushareClient)
	quotes, err := quoteFetcher.FetchAllQuotes(ctx)
	if err != nil {
		logger.Warn("fetch quotes failed", zap.Error(err))
		return err
	}

	stockRepo := repo.NewStockRepository()
	activeStocks, err := stockRepo.GetActiveStocks(ctx)
	if err != nil {
		logger.Warn("fetch active stocks failed", zap.Error(err))
		return err
	}

	stockMap := make(map[string]*dal_model.StockBasicInfo, len(activeStocks))
	for i := range activeStocks {
		stockMap[activeStocks[i].TsCode] = &activeStocks[i]
	}

	var limitUpCodes, above5Codes []string
	var removeFromLimitUp, removeFromAbove5 []string
	var newEntries []string
	poolCache := cache.NewPoolCache()
	var quoteTime time.Time

	for _, quote := range quotes {
		stock, exists := stockMap[quote.TsCode]
		if !exists {
			continue
		}
		if stock.IsST {
			continue
		}

		if quoteTime.IsZero() {
			quoteTime = quote.UpdateTime
		}

		board := s.DetectBoard(quote.TsCode)
		rule := s.boardRules[board]
		prevState := s.stockStates[quote.TsCode]

		input := dal_model.DetectInput{
			TsCode:       quote.TsCode,
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
		s.stockStates[quote.TsCode] = output.CurrentState

		if output.IsLimitUp {
			limitUpCodes = append(limitUpCodes, quote.TsCode)
			removeFromAbove5 = append(removeFromAbove5, quote.TsCode)
			if output.IsFirstLimitUp {
				poolCache.SetFirstLimitTime(ctx, date, quote.TsCode, output.FirstLimitTime.Format("15:04:05"))
				newEntries = append(newEntries, quote.TsCode)
			}
		} else if output.IsAbove5Pct {
			above5Codes = append(above5Codes, quote.TsCode)
			if output.CurrentState == dal_model.LimitStateOpened {
				removeFromLimitUp = append(removeFromLimitUp, quote.TsCode)
			}
			// First time entering above5 pool
			if prevState == dal_model.LimitStateNone || prevState == dal_model.LimitStateOpened {
				newEntries = append(newEntries, quote.TsCode)
			}
		} else {
			removeFromLimitUp = append(removeFromLimitUp, quote.TsCode)
			removeFromAbove5 = append(removeFromAbove5, quote.TsCode)
		}
	}

	// Batch pool operations
	if err := poolCache.RemoveLimitUpBatch(ctx, date, removeFromLimitUp); err != nil {
		logger.Warn("batch remove limit-up failed", zap.Error(err), zap.Int("count", len(removeFromLimitUp)))
	}
	if err := poolCache.RemoveAbove5Batch(ctx, date, removeFromAbove5); err != nil {
		logger.Warn("batch remove above5 failed", zap.Error(err), zap.Int("count", len(removeFromAbove5)))
	}
	if err := poolCache.AddLimitUpBatch(ctx, date, limitUpCodes); err != nil {
		logger.Warn("batch add limit-up failed", zap.Error(err), zap.Int("count", len(limitUpCodes)))
	}
	if err := poolCache.AddAbove5Batch(ctx, date, above5Codes); err != nil {
		logger.Warn("batch add above5 failed", zap.Error(err), zap.Int("count", len(above5Codes)))
	}

	allHeatInputs := s.buildHeatInputs(quotes, stockMap)
	if len(allHeatInputs) > 0 {
		if quoteTime.IsZero() {
			quoteTime = time.Now()
		}
		classifySvc := classify.NewClassifyService()
		classifyResults, err := classifySvc.ClassifyBySimpleHeat(ctx, allHeatInputs, date)
		if err != nil {
			logger.Warn("heat classify failed", zap.Error(err), zap.Int("count", len(allHeatInputs)))
		}

		if len(classifyResults) > 0 {
			classificationCache := cache.NewClassificationCache()
			if mergeErr := classificationCache.MergeClassificationResult(ctx, date, classifyResults); mergeErr != nil {
				logger.Warn("merge classification result failed", zap.Error(mergeErr))
			}
			logger.Info("batch classified stocks", zap.Int("count", len(classifyResults)))
		}
	}

	s.mu.Lock()
	s.stocksProcessed = len(quotes)
	s.mu.Unlock()

	if s.notifyCallback != nil {
		s.notifyCallback(date)
	}

	return nil
}

func (s *MonitorServiceImpl) buildHeatInputs(quotes []*dal_model.StockQuote, stockMap map[string]*dal_model.StockBasicInfo) []classify.StockQuoteInput {
	heatInputs := make([]classify.StockQuoteInput, 0, len(quotes))
	for _, quote := range quotes {
		stock, exists := stockMap[quote.TsCode]
		if !exists || stock.IsST {
			continue
		}

		board := s.DetectBoard(quote.TsCode)
		rule := s.boardRules[board]
		output := limiter.DetectLimitUp(dal_model.DetectInput{
			TsCode:       quote.TsCode,
			StockName:    stock.Name,
			CurrentPrice: quote.Price,
			PreClose:     quote.PreClose,
			ChangePct:    quote.PctChg,
			BoardCode:    board,
			IsST:         stock.IsST,
			QuoteTime:    quote.UpdateTime,
		}, rule)
		heatInputs = append(heatInputs, classify.StockQuoteInput{
			TsCode:        quote.TsCode,
			ChangePercent: quote.PctChg,
			IsLimitUp:     output.IsLimitUp,
		})
	}
	return heatInputs
}

func (s *MonitorServiceImpl) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	ctx, s.cancelFunc = context.WithCancel(ctx)
	s.running = true
	s.mu.Unlock()

	ticker := time.NewTicker(time.Duration(s.cfg.IntervalSec) * time.Second)
	logger.Info("start monitor service")
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
			return
		case <-ticker.C:
			now := time.Now()
			if !utils.IsTradingDay(now) {
				continue
			}
			if !s.isTradingTime(now) {
				continue
			}
			date := now.Format("2006-01-02")
			if err := s.ProcessTick(ctx, date); err != nil {
				logger.Warn("process tick failed", zap.Error(err))
			}
			s.mu.Lock()
			s.lastTickAt = now
			s.mu.Unlock()
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
