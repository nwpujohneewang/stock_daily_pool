package classify

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"stock/dal/dao"
	"stock/dal/repo"
	"stock/internal/pkg/utils"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"stock/config"
	"stock/dal/cache"
	"stock/external/llm"
	"stock/internal/pkg/logger"
	"stock/model/dal_model"
)

const llmBatchSize = 200

type LLMClassifyService struct {
	llmClient     *llm.Client
	sectorFetcher *SectorFetcher
	llmCache      cache.LLMClassificationCacheInterface
	stockTopics   cache.StockTopicsRelationCacheInterface
	cfg           *config.LLMConfig
}

func NewLLMClassifyService(
	llmClient *llm.Client,
	cfg *config.LLMConfig,
) *LLMClassifyService {
	return &LLMClassifyService{
		llmClient:     llmClient,
		sectorFetcher: NewSectorFetcher(),
		llmCache:      cache.NewLLMClassificationCache(),
		stockTopics:   cache.NewStockTopicsRelationCache(),
		cfg:           cfg,
	}
}

// RunClassification runs the full two-phase LLM classification and writes results to cache.
// Called as a goroutine from monitor.ProcessTick; single-batch failures are logged but do not abort.
func (s *LLMClassifyService) RunClassification(
	ctx context.Context,
	date string,
	quotes []*dal_model.StockQuote,
	stockMap map[string]*dal_model.StockBasicInfo,
	limitUpSet map[string]bool,
) error {
	start := time.Now()

	limitUpStocks, strongStocks := s.filterStocks(quotes, stockMap, limitUpSet)
	if len(limitUpStocks) == 0 && len(strongStocks) == 0 {
		return nil
	}

	sectorChanges, err := s.sectorFetcher.FetchSectorChanges(ctx, date)
	if err != nil {
		logger.Warn("fetch sector changes failed", zap.Error(err))
		sectorChanges = map[string]float64{}
	}

	// Load topic relations for all involved stocks.
	allCodes := make([]string, 0, len(limitUpStocks)+len(strongStocks))
	for _, q := range limitUpStocks {
		allCodes = append(allCodes, q.TsCode)
	}
	for _, q := range strongStocks {
		allCodes = append(allCodes, q.TsCode)
	}
	relationsByStock, err := s.stockTopics.GetStockTopicsBatch(ctx, allCodes)
	if err != nil {
		return fmt.Errorf("load stock topics: %w", err)
	}

	// Incremental filtering: skip already-classified / manual-annotated / recent last_seen stocks.
	// Load existing results once and reuse for both groups.
	existingResults, hit, err := s.llmCache.GetClassificationResult(ctx, date)
	if err != nil {
		logger.Warn("get llm classification result failed", zap.Error(err))
	}
	limitUpStocks = filterIncremental(limitUpStocks, relationsByStock, existingResults, hit, date)
	strongStocks = filterIncremental(strongStocks, relationsByStock, existingResults, hit, date)
	if len(limitUpStocks) == 0 && len(strongStocks) == 0 {
		return nil
	}

	// Phase 1: classify limit-up stocks.
	limitUpResults, assignedTopics := s.classifyBatch(ctx, limitUpStocks, stockMap, relationsByStock, sectorChanges, nil, true)

	// Phase 2: classify strong (non-limit-up) stocks, guided by confirmed limit-up topics.
	// Deduplicate assignedTopics to avoid bloating the Phase 2 prompt.
	strongResults, _ := s.classifyBatch(ctx, strongStocks, stockMap, relationsByStock, sectorChanges, dedupeStrings(assignedTopics), false)

	merged := make(map[string][]dal_model.TopicRelation, len(limitUpResults)+len(strongResults))
	for k, v := range limitUpResults {
		merged[k] = v
	}
	for k, v := range strongResults {
		merged[k] = v
	}

	enrichLLMResults(ctx, merged, relationsByStock)

	if err := s.llmCache.MergeClassificationResult(ctx, date, merged); err != nil {
		logger.Warn("merge llm classification result failed", zap.Error(err))
	}

	logger.Info("llm classify completed",
		zap.Int("limit_up_count", len(limitUpResults)),
		zap.Int("strong_count", len(strongResults)),
		zap.Int("total", len(merged)),
		zap.Duration("elapsed", time.Since(start)),
	)
	return nil
}

func (s *LLMClassifyService) filterStocks(
	quotes []*dal_model.StockQuote,
	stockMap map[string]*dal_model.StockBasicInfo,
	limitUpSet map[string]bool,
) (limitUp, strong []*dal_model.StockQuote) {
	for _, q := range quotes {
		info, ok := stockMap[q.TsCode]
		if !ok || info.IsST {
			continue
		}
		if limitUpSet[q.TsCode] {
			limitUp = append(limitUp, q)
		} else if q.PctChg > ClassifyThreshold {
			strong = append(strong, q)
		}
	}
	return
}

// classifyBatch splits stocks into batches of llmBatchSize, calls KIMI in parallel
// (bounded by cfg.MaxConcurrent), and returns (results, assignedTopicNames, error).
func (s *LLMClassifyService) classifyBatch(
	ctx context.Context,
	stocks []*dal_model.StockQuote,
	stockMap map[string]*dal_model.StockBasicInfo,
	relationsByStock map[string][]dal_model.TopicRelation,
	sectorChanges map[string]float64,
	confirmedTopics []string,
	limitUp bool,
) (map[string][]dal_model.TopicRelation, []string) {
	if len(stocks) == 0 {
		return nil, nil
	}

	batches := splitIntoBatches(stocks, llmBatchSize)
	maxConcurrent := s.cfg.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}

	sem := make(chan struct{}, maxConcurrent)
	type batchResult struct {
		items []llm.BatchClassifyItem
	}
	resultsCh := make(chan batchResult, len(batches))

	sectorTop20 := validSectors(sectorChanges)

	var wg sync.WaitGroup
	for _, batch := range batches {
		wg.Add(1)
		b := batch
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			systemPrompt := buildSystemPrompt()
			userPrompt := buildUserPrompt(b, stockMap, relationsByStock, sectorTop20, confirmedTopics, limitUp)

			raw, err := s.llmClient.ChatCompletion(ctx, systemPrompt, userPrompt)
			if err != nil {
				logger.Warn("kimi batch classify failed", zap.Error(err), zap.Int("batch_size", len(b)))
				return
			}

			logger.Info("kimi response", zap.String("val = ", string(raw)))

			var items []llm.BatchClassifyItem
			if err := json.Unmarshal([]byte(cleanLLMJSON(raw)), &items); err != nil {
				logger.Warn("parse kimi response failed", zap.Error(err))
				return
			}
			resultsCh <- batchResult{items: items}
		}()
	}
	wg.Wait()
	close(resultsCh)

	results := make(map[string][]dal_model.TopicRelation)
	var assignedTopicNames []string
	for br := range resultsCh {
		for _, item := range br.items {
			if item.TsCode == "" || item.TopicName == "" {
				continue
			}
			results[item.TsCode] = []dal_model.TopicRelation{
				{TopicName: item.TopicName, Confidence: item.Confidence},
			}
			assignedTopicNames = append(assignedTopicNames, item.TopicName)
		}
	}
	return results, assignedTopicNames
}

func enrichLLMResults(ctx context.Context, results map[string][]dal_model.TopicRelation, relationsByStock map[string][]dal_model.TopicRelation) {
	if len(results) == 0 {
		return
	}

	type enrichInfo struct {
		TopicID  int64
		Category string
	}
	enrichMap := make(map[string]enrichInfo)
	var missingNames []string

	for tsCode, relations := range results {
		for _, r := range relations {
			if r.TopicID != 0 || r.TopicName == "" {
				continue
			}
			mapKey := tsCode + "|" + r.TopicName
			if _, exists := enrichMap[mapKey]; exists {
				continue
			}
			found := false
			if existingRelations, ok := relationsByStock[tsCode]; ok {
				for _, rel := range existingRelations {
					if rel.TopicName == r.TopicName {
						enrichMap[mapKey] = enrichInfo{TopicID: rel.TopicID, Category: rel.Category}
						found = true
						break
					}
				}
			}
			if !found {
				missingNames = append(missingNames, r.TopicName)
			}
		}
	}

	if len(missingNames) > 0 {
		dictCache := cache.NewTopicDictionaryCache()
		dictMap, cacheHit := dictCache.GetAllMap(ctx)
		if !cacheHit {
			dictDAO := dao.NewTopicDictionaryDAO()
			dbMap, err := dictDAO.GetAllMap(ctx)
			if err != nil {
				logger.Warn("enrichLLMResults: load topic_dictionary failed", zap.Error(err))
			} else {
				dictMap = dbMap
				_ = dictCache.SetAllMap(ctx, dbMap)
			}
		}

		topicRepo := repo.NewTopicRepository()
		deduped := dedupeStrings(missingNames)

		for _, name := range deduped {
			mapKey := "|" + name
			if _, exists := enrichMap[mapKey]; exists {
				continue
			}
			if dict, ok := dictMap[name]; ok {
				enrichMap[mapKey] = enrichInfo{Category: dict.Category}
			}
		}

		var stillMissing []string
		for _, name := range deduped {
			mapKey := "|" + name
			if info, exists := enrichMap[mapKey]; exists && info.Category != "" {
				topic, err := topicRepo.GetByName(ctx, name)
				if err == nil && topic != nil {
					enrichMap[mapKey] = enrichInfo{TopicID: topic.ID, Category: info.Category}
				} else {
					stillMissing = append(stillMissing, name)
				}
			} else {
				stillMissing = append(stillMissing, name)
			}
		}

		//if len(stillMissing) > 0 {
		//	dictDAO := dao.NewTopicDictionaryDAO()
		//	now := time.Now()
		//	for _, name := range stillMissing {
		//		dict := dal_model.TopicDictionary{
		//			RawTopicName:   name,
		//			NormalizedName: name,
		//			Category:       "",
		//			Source:         2,
		//			CreatedAt:      now,
		//			UpdatedAt:      now,
		//		}
		//		created, err := dictDAO.Create(ctx, dict)
		//		if err != nil {
		//			logger.Warn("enrichLLMResults: insert topic_dictionary failed",
		//				zap.String("name", name), zap.Error(err))
		//			continue
		//		}
		//		mapKey := "|" + name
		//		enrichMap[mapKey] = enrichInfo{Category: created.Category}
		//		_ = dictCache.SetAllMap(ctx, map[string]dal_model.TopicDictionary{name: *created})
		//	}
		//}
	}

	for tsCode, relations := range results {
		for i, r := range relations {
			if r.TopicID != 0 || r.TopicName == "" {
				continue
			}
			mapKey := tsCode + "|" + r.TopicName
			if info, ok := enrichMap[mapKey]; ok {
				results[tsCode][i].TopicID = info.TopicID
				if info.Category != "" {
					results[tsCode][i].Category = info.Category
				}
			} else {
				mapKey = "|" + r.TopicName
				if info, ok := enrichMap[mapKey]; ok {
					if info.TopicID != 0 {
						results[tsCode][i].TopicID = info.TopicID
					}
					if info.Category != "" {
						results[tsCode][i].Category = info.Category
					}
				}
			}
		}
	}
}

func buildSystemPrompt() string {
	return `## 角色
你是一个A股短线投资专家，擅长打板套利，尤其擅长对股票进行热点分类，在确认龙头后，打板中游或者跟风股票进行套利

## 任务
将股票进行题材分类，方便第二天对前一天未涨停但是涨幅超过5的股票进行打板。
对于每一只股票你都需要按照以下的步骤进行分类：
第一步：根据输入的板块的涨跌幅分析出今天的热点板块
第二步：列举出公司的相关业务
第三步：结合板块信息、公司业务信息，从股票前置被关联的topics中选出最相关的topic作为分类结果
**注意：** 只从该股票被关联的topics中选取，不要自己新建

## 输入
下面对输入参数进行说明
sector_changes：板块涨幅变化，结构体List
  - name：板块名称
  - pct_chg：板块涨幅
stocks：股票List
  - ts_code ：股票代码
  - name：股票名
  - topics ：这个股票关联的主题，前置链路手动关联
confirmed_limitup_topics：涨停股票分类后的主题List

## 输出
只需要输出
- ts_code ： 股票code
- topic_name： 分类名称
- confidence：分类置信度，0.0-1.0之间的浮点数，表示你对分类结果的确信程度
严格输出JSON数组，格式：[{"ts_code":"...","topic_name":"...","confidence":0.8}]`
}

type sectorEntry struct {
	Name   string  `json:"name"`
	PctChg float64 `json:"pct_chg"`
}

type stockEntry struct {
	TsCode string   `json:"ts_code"`
	Name   string   `json:"name"`
	Topics []string `json:"topics"`
}

type userPromptPayload struct {
	AnalysisTarget         string        `json:"analysis_target"`
	SectorChanges          []sectorEntry `json:"sector_changes"`
	Stocks                 []stockEntry  `json:"stocks"`
	ConfirmedLimitUpTopics []string      `json:"confirmed_limitup_topics,omitempty"`
}

func buildUserPrompt(
	batch []*dal_model.StockQuote,
	stockMap map[string]*dal_model.StockBasicInfo,
	relationsByStock map[string][]dal_model.TopicRelation,
	sectorTop20 []sectorEntry,
	confirmedTopics []string,
	limitUp bool,
) string {
	stocks := make([]stockEntry, 0, len(batch))
	for _, q := range batch {
		name := q.TsCode
		if info, ok := stockMap[q.TsCode]; ok {
			name = info.Name
		}
		relations := relationsByStock[q.TsCode]
		topics := make([]string, 0, len(relations))
		for _, r := range relations {
			if r.TopicName != "" {
				topics = append(topics, r.TopicName)
			}
		}
		stocks = append(stocks, stockEntry{
			TsCode: q.TsCode,
			Name:   name,
			Topics: topics,
		})
	}

	analysisTarget := "以下股票为当日涨停股票，请重点分析其涨停原因和所属热点题材"
	if !limitUp {
		analysisTarget = "以下股票为当日涨幅超过5%但未涨停的股票，请分析其上涨原因和所属热点题材"
	}

	payload := userPromptPayload{
		AnalysisTarget:         analysisTarget,
		SectorChanges:          sectorTop20,
		Stocks:                 stocks,
		ConfirmedLimitUpTopics: confirmedTopics,
	}
	b, _ := json.Marshal(payload)
	return string(b)
}

func validSectors(changes map[string]float64) []sectorEntry {
	validConcepts := utils.FilterOverZeroAndNonThemeConcepts(changes)
	entries := make([]sectorEntry, 0, len(validConcepts))
	for name, pct := range validConcepts {
		entries = append(entries, sectorEntry{Name: name, PctChg: pct})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].PctChg > entries[j].PctChg })
	return entries
}

func splitIntoBatches(stocks []*dal_model.StockQuote, size int) [][]*dal_model.StockQuote {
	var batches [][]*dal_model.StockQuote
	for i := 0; i < len(stocks); i += size {
		end := i + size
		if end > len(stocks) {
			end = len(stocks)
		}
		batches = append(batches, stocks[i:end])
	}
	return batches
}

func cleanLLMJSON(s string) string {
	s = strings.TrimSpace(s)
	// Extract content from a markdown code block wherever it appears in the response.
	if idx := strings.Index(s, "```json"); idx != -1 {
		s = s[idx+7:]
	} else if idx := strings.Index(s, "```"); idx != -1 {
		s = s[idx+3:]
	}
	// Strip the closing fence.
	if idx := strings.LastIndex(s, "```"); idx != -1 {
		s = s[:idx]
	}
	return strings.TrimSpace(s)
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, exists := seen[s]; !exists {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

// filterIncremental removes stocks that should be skipped this round:
// already classified today, manually annotated, or have a last_seen within 7 days.
func filterIncremental(
	stocks []*dal_model.StockQuote,
	relationsByStock map[string][]dal_model.TopicRelation,
	existingResults map[string][]dal_model.TopicRelation,
	hit bool,
	date string,
) []*dal_model.StockQuote {
	filtered := make([]*dal_model.StockQuote, 0, len(stocks))
	for _, q := range stocks {
		if hit && len(existingResults[q.TsCode]) > 0 {
			continue
		}
		relations := relationsByStock[q.TsCode]
		if _, ok := pickLatestManualRelation(relations); ok {
			continue
		}
		if hasRecentLastSeen(relations, date) {
			continue
		}
		filtered = append(filtered, q)
	}
	return filtered
}

func hasRecentLastSeen(relations []dal_model.TopicRelation, date string) bool {
	for _, r := range relations {
		if IsRecentTopic(r.LastSeenDate, date) {
			return true
		}
	}
	return false
}
