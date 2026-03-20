package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"stock/internal/cache"
	"stock/internal/config"
	"stock/internal/model"
	"stock/internal/pkg/limiter"
	"stock/internal/pkg/shard"
	"stock/internal/repo"
)

type MonitorService struct {
	db           *repo.StockRepo
	quoteCache   *cache.QuoteCache
	poolCache    *cache.PoolCache
	boardRepo    *repo.BoardRepo
	poolRepo     *repo.PoolRepo
	quoteFetcher *QuoteFetcher
	classifySvc  *ClassifyService
	alertSvc     *AlertService
	cfg          *config.MonitorConfig
	boardRules   map[model.BoardCode]*model.BoardRule
	stockStates  map[string]int
	mu           sync.RWMutex
	logger       *log.Logger
}

func NewMonitorService(
	db *repo.StockRepo,
	quoteCache *cache.QuoteCache,
	poolCache *cache.PoolCache,
	boardRepo *repo.BoardRepo,
	poolRepo *repo.PoolRepo,
	quoteFetcher *QuoteFetcher,
	classifySvc *ClassifyService,
	alertSvc *AlertService,
	cfg *config.MonitorConfig,
) *MonitorService {
	return &MonitorService{
		db:           db,
		quoteCache:   quoteCache,
		poolCache:    poolCache,
		boardRepo:    boardRepo,
		poolRepo:     poolRepo,
		quoteFetcher: quoteFetcher,
		classifySvc:  classifySvc,
		alertSvc:     alertSvc,
		cfg:          cfg,
		boardRules:   initBoardRules(),
		stockStates:  make(map[string]int),
		logger:       log.Default(),
	}
}

func initBoardRules() map[model.BoardCode]*model.BoardRule {
	return map[model.BoardCode]*model.BoardRule{
		model.BoardMain: {BoardCode: model.BoardMain, BoardName: "主板", LimitUpRatio: 0.10, LimitDownRatio: -0.10},
		model.BoardGEM:  {BoardCode: model.BoardGEM, BoardName: "创业板", LimitUpRatio: 0.20, LimitDownRatio: -0.20},
		model.BoardSTAR: {BoardCode: model.BoardSTAR, BoardName: "科创板", LimitUpRatio: 0.20, LimitDownRatio: -0.20},
		model.BoardBSE:  {BoardCode: model.BoardBSE, BoardName: "北交所", LimitUpRatio: 0.30, LimitDownRatio: -0.30},
	}
}

func (s *MonitorService) DetectBoard(tsCode string) model.BoardCode {
	return limiter.DetectBoard(tsCode[:6])
}

func (s *MonitorService) ProcessTick(ctx context.Context, date string) error {
	if err := s.quoteFetcher.FetchAllQuotes(ctx); err != nil {
		s.logger.Printf("fetch quotes failed: %v", err)
	}

	stocks, err := s.db.GetActiveStocks(ctx)
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

func (s *MonitorService) processShard(ctx context.Context, date string, batch shard.ShardBatch) {
	for _, tsCode := range batch.TsCodes {
		stock, err := s.db.GetByTsCode(ctx, tsCode)
		if err != nil {
			s.logger.Printf("get stock %s: %v", tsCode, err)
			continue
		}

		if stock.IsST {
			continue
		}

		quote, err := s.quoteCache.Get(ctx, tsCode)
		if err != nil || quote == nil {
			continue
		}

		board := s.DetectBoard(tsCode)
		rule := s.boardRules[board]

		s.mu.RLock()
		prevState := s.stockStates[tsCode]
		s.mu.RUnlock()

		input := model.DetectInput{
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
			s.poolCache.AddLimitUp(ctx, date, tsCode)
			if output.IsFirstLimitUp {
				s.poolCache.SetFirstLimitTime(ctx, date, tsCode, output.FirstLimitTime.Format("15:04:05"))
				if s.classifySvc != nil && s.alertSvc != nil {
					s.triggerClassifyAndAlert(ctx, date, tsCode, stock.Name, quote, output)
				}
			}
		} else if output.IsAbove5Pct {
			s.poolCache.AddAbove5(ctx, date, tsCode)
		}
	}
}

func (s *MonitorService) triggerClassifyAndAlert(ctx context.Context, date, tsCode, stockName string, quote *model.StockQuote, output model.DetectOutput) {
	go func() {
		classifyCtx := context.Background()
		topics, err := s.classifySvc.ClassifyStock(classifyCtx, tsCode)
		if err != nil {
			s.logger.Printf("classify stock %s: %v", tsCode, err)
			return
		}
		if len(topics) == 0 {
			return
		}

		prevDate := prevTradingDay(date)
		prevPool, err := s.poolRepo.GetByTsCodeAndDate(classifyCtx, prevDate, tsCode)
		if err != nil {
			s.logger.Printf("get prev pool %s %s: %v", tsCode, prevDate, err)
		}

		var prevDayPct float64
		var prevDayAbove5 bool
		var prevDayLimitUp bool
		if prevPool != nil {
			if prevPool.ChangePct != nil {
				prevDayPct = *prevPool.ChangePct
			}
			prevDayLimitUp = prevPool.PoolType == model.PoolTypeLimitUp
			prevDayAbove5 = prevPool.PoolType == model.PoolTypeAbove5 || prevPool.PoolType == model.PoolTypeLimitUp
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
		if _, err := s.alertSvc.CheckAndAlert(classifyCtx, alertInput); err != nil {
			s.logger.Printf("check alert %s: %v", tsCode, err)
		}
	}()
}

func prevTradingDay(date string) string {
	t, _ := time.Parse("2006-01-02", date)
	t = t.AddDate(0, 0, -1)
	return t.Format("2006-01-02")
}

func (s *MonitorService) Start(ctx context.Context) {
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
				s.logger.Printf("process tick: %v", err)
			}
		}
	}
}

func (s *MonitorService) isTradingTime(t time.Time) bool {
	h, m := t.Hour(), t.Minute()
	total := h*60 + m
	start := 9*60 + 25
	end := 15*60 + 1
	return total >= start && total <= end
}

type ClassifyService struct {
	mappingCache *cache.MappingCache
	conceptCache *cache.ConceptCache
	topicRepo    *repo.TopicRepo
	mappingRepo  *repo.MappingRepo
	evidenceRepo *repo.EvidenceRepo
	logger       *log.Logger
}

func NewClassifyService(
	mappingCache *cache.MappingCache,
	conceptCache *cache.ConceptCache,
	topicRepo *repo.TopicRepo,
	mappingRepo *repo.MappingRepo,
	evidenceRepo *repo.EvidenceRepo,
) *ClassifyService {
	return &ClassifyService{
		mappingCache: mappingCache,
		conceptCache: conceptCache,
		topicRepo:    topicRepo,
		mappingRepo:  mappingRepo,
		evidenceRepo: evidenceRepo,
		logger:       log.Default(),
	}
}

func (s *ClassifyService) ClassifyStock(ctx context.Context, tsCode string) ([]model.TopicMapping, error) {
	date := time.Now().Format("2006-01-02")

	mappings, err := s.mappingCache.GetStockTopics(ctx, tsCode)
	if err != nil {
		return nil, err
	}

	if mappings != nil && len(mappings) > 0 {
		for _, m := range mappings {
			if m.Source == "manual" {
				s.saveEvidence(ctx, date, tsCode, m.TopicID, "L1_REDIS", "MANUAL", mappings, "", 1.0)
				return []model.TopicMapping{m}, nil
			}
		}
		s.saveEvidence(ctx, date, tsCode, mappings[0].TopicID, "L1_REDIS", "JIUYAN_ATTR", mappings, "", 0.8)
		return mappings, nil
	}

	pgMappings, err := s.mappingRepo.GetByTsCode(ctx, tsCode)
	if err != nil {
		return nil, err
	}

	if len(pgMappings) > 0 {
		result := make([]model.TopicMapping, len(pgMappings))
		for i, m := range pgMappings {
			topic, _ := s.topicRepo.GetByID(ctx, m.TopicID)
			topicName := ""
			if topic != nil {
				topicName = topic.Name
			}
			result[i] = model.TopicMapping{
				TopicID:      m.TopicID,
				TopicName:    topicName,
				Source:       m.Source,
				HitCount:     m.HitCount,
				LastSeenDate: m.LastSeenDate.Format("2006-01-02"),
			}
		}
		s.mappingCache.SetStockTopics(ctx, tsCode, result)
		s.saveEvidence(ctx, date, tsCode, pgMappings[0].TopicID, "L2_PG_JIUYAN", "JIUYAN_ATTR", result, "", 0.8)
		return result, nil
	}

	concepts, err := s.conceptCache.GetStockConcepts(ctx, tsCode)
	if err != nil || concepts == nil {
		s.saveEvidence(ctx, date, tsCode, 0, "L3_PG_CONCEPT", "CONCEPT_ATTR", nil, "no concepts found", 0.0)
		return nil, nil
	}

	s.saveEvidence(ctx, date, tsCode, 0, "L3_PG_CONCEPT", "CONCEPT_ATTR", nil, "concepts found but no mapping", 0.0)
	return nil, nil
}

func (s *ClassifyService) saveEvidence(ctx context.Context, date, tsCode string, topicID int64, layer, strategy string, candidates []model.TopicMapping, evidenceText string, confidence float64) {
	if s.evidenceRepo == nil {
		return
	}
	candidateScores, _ := json.Marshal(candidates)
	parsedDate, _ := time.Parse("2006-01-02", date)
	e := model.ClassificationAuditLog{
		Date:            parsedDate,
		TsCode:          tsCode,
		TopicID:         &topicID,
		ClassifyLayer:   layer,
		Strategy:        strategy,
		CandidateScores: candidateScores,
		EvidenceText:    &evidenceText,
		Confidence:      &confidence,
	}
	s.evidenceRepo.Create(ctx, e)
}

func (s *ClassifyService) NormalizeTopicName(ctx context.Context, rawName string) (int64, string, bool, error) {
	topic, err := s.topicRepo.GetByName(ctx, rawName)
	if err == nil && topic != nil {
		return topic.ID, topic.Name, false, nil
	}

	return 0, "", true, nil
}
