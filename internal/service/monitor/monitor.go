package monitor

import (
	"context"
	"stock/external/llm"
	"stock/external/tushare"
	"stock/internal/pkg/utils"
	"sync"
	"time"

	"stock/config"
	"stock/dal/cache"
	"stock/dal/repo"
	"stock/internal/pkg/limiter"
	"stock/internal/pkg/logger"
	"stock/internal/service/classify"
	"stock/internal/service/quote_fetcher"
	"stock/model/dal_model"

	"go.uber.org/zap"
)

var instance *MonitorServiceImpl

type MonitorServiceImpl struct {
	tushareClient      *tushare.Client
	llmClassifyService *classify.LLMClassifyService
	cfg                *config.MonitorConfig
	boardRules         map[dal_model.BoardCode]*dal_model.BoardRule
	stockStates        map[string]int
	notifyCallback     func(date string)
	running            bool
	lastTickAt         time.Time
	stocksProcessed    int
	mu                 sync.Mutex
	tickMu             sync.Mutex
	cancelFunc         context.CancelFunc
	// Snapshot of last ProcessTick data for the independent LLM Ticker.
	snapshotMu     sync.RWMutex
	lastDate       string
	lastQuotes     []*dal_model.StockQuote
	lastStockMap   map[string]*dal_model.StockBasicInfo
	lastLimitUpSet map[string]bool
}

type MonitorStatus struct {
	Running         bool      `json:"running"`
	LastTickAt      time.Time `json:"last_tick_at"`
	StocksProcessed int       `json:"stocks_processed"`
}

func (s *MonitorServiceImpl) SetNotifyCallback(fn func(date string)) {
	s.notifyCallback = fn
}

func NewMonitorService(tushareClient *tushare.Client, llmClient *llm.Client, llmCfg *config.LLMConfig, cfg *config.MonitorConfig) *MonitorServiceImpl {
	svc := &MonitorServiceImpl{
		tushareClient: tushareClient,
		cfg:           cfg,
		boardRules:    initBoardRules(),
		stockStates:   make(map[string]int),
	}
	if llmClient != nil && llmCfg != nil {
		svc.llmClassifyService = classify.NewLLMClassifyService(llmClient, llmCfg)
	}
	return svc
}

func Init(tushareClient *tushare.Client, llmClient *llm.Client, llmCfg *config.LLMConfig, cfg *config.MonitorConfig) {
	instance = NewMonitorService(tushareClient, llmClient, llmCfg, cfg)
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
	s.stockStates = make(map[string]int)
}

func (s *MonitorServiceImpl) GetLLMClassifyService() *classify.LLMClassifyService {
	return s.llmClassifyService
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
	s.tickMu.Lock()
	defer s.tickMu.Unlock()

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
	poolCache := cache.NewPoolCache()
	var quoteTime time.Time
	limitUpSet := make(map[string]bool, len(quotes))

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
			limitUpSet[quote.TsCode] = true
			if output.IsFirstLimitUp {
				poolCache.SetFirstLimitTime(ctx, date, quote.TsCode, output.FirstLimitTime.Format("15:04:05"))
			}
		} else if output.IsAbove5Pct {
			above5Codes = append(above5Codes, quote.TsCode)
			if output.CurrentState == dal_model.LimitStateOpened {
				removeFromLimitUp = append(removeFromLimitUp, quote.TsCode)
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

	// Step 1: Seed classification cache with LLM results from the previous tick (preferred source).
	if s.llmClassifyService != nil {
		llmCache := cache.NewLLMClassificationCache()
		if llmResults, hit, _ := llmCache.GetClassificationResult(ctx, date); hit && len(llmResults) > 0 {
			classificationCache := cache.NewClassificationCache()
			if mergeErr := classificationCache.MergeClassificationResult(ctx, date, llmResults); mergeErr != nil {
				logger.Warn("merge llm classification result failed", zap.Error(mergeErr))
			}
		}
	}

	// Step 2: Heat classification fills gaps for stocks not covered by LLM.
	allHeatInputs := s.buildHeatInputs(quotes, stockMap, limitUpSet)
	if len(allHeatInputs) > 0 {
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

	// Update LLM Ticker snapshot with the latest tick data.
	s.snapshotMu.Lock()
	s.lastDate = date
	s.lastQuotes = quotes
	s.lastStockMap = stockMap
	s.lastLimitUpSet = limitUpSet
	s.snapshotMu.Unlock()

	if s.notifyCallback != nil {
		s.notifyCallback(date)
	}

	return nil
}

func (s *MonitorServiceImpl) buildHeatInputs(quotes []*dal_model.StockQuote, stockMap map[string]*dal_model.StockBasicInfo, limitUpSet map[string]bool) []classify.StockQuoteInput {
	heatInputs := make([]classify.StockQuoteInput, 0, len(quotes))
	for _, quote := range quotes {
		stock, exists := stockMap[quote.TsCode]
		if !exists || stock.IsST {
			continue
		}
		heatInputs = append(heatInputs, classify.StockQuoteInput{
			TsCode:        quote.TsCode,
			ChangePercent: quote.PctChg,
			IsLimitUp:     limitUpSet[quote.TsCode],
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

	// Independent 10-minute LLM classification ticker, decoupled from ProcessTick.
	if s.llmClassifyService != nil {
		llmTicker := time.NewTicker(10 * time.Minute)
		llmSvc := s.llmClassifyService
		go func() {
			defer llmTicker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-llmTicker.C:
					now := time.Now()
					if !utils.IsTradingDay(now) || !s.isTradingTime(now) {
						continue
					}
					s.snapshotMu.RLock()
					date := s.lastDate
					quotes := s.lastQuotes
					stockMap := s.lastStockMap
					limitUpSet := s.lastLimitUpSet
					s.snapshotMu.RUnlock()
					if date == "" || len(quotes) == 0 {
						continue
					}
					if err := llmSvc.RunClassification(ctx, date, quotes, stockMap, limitUpSet); err != nil {
						logger.Warn("llm classify failed", zap.Error(err))
					}
				}
			}
		}()
	}

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
