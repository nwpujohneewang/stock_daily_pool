package classify

import (
	"context"
	crand "crypto/rand"
	"math/big"
	"stock/dal/cache"
	"stock/dal/repo"
	"stock/model/dal_model"
)

// Heat classify (online path)
//
// This file implements a lightweight "heat-based" topic classification:
// - Heat universe: stocks with pct_chg > HeatRisingThreshold (default 3%) contribute to topic heat stats.
// - Classification universe: stocks with pct_chg > ClassifyThreshold (default 5%) get a final topic.
//
// Assignment order & rules:
//  1. Limit-up stocks first:
//     manual relation > recent relation (<= 7 days) > best topic by today's heat ranking.
//  2. Strong (non-limit-up) stocks next:
//     try topics already claimed by limit-up stocks > otherwise fall back to heat ranking.
//
// Note: We intentionally keep this logic count-based (limit-up count / strong count / participation),
// so the behavior is stable and easy to reason about.
type StockQuoteInput struct {
	TsCode        string
	ChangePercent float64
	IsLimitUp     bool
}

type HeatClassifyService struct {
	topicStockCountRepo      repo.TopicStockCountRepository
	topicHeatCache           cache.TopicHeatCacheInterface
	stockTopicsRelationCache cache.StockTopicsRelationCacheInterface
}

type topicHeatAccumulator struct {
	stockCount   int
	limitUpCount int
	risingCount  int
}

// classificationRunState holds per-run mutable state so the main flow reads as steps.
type classificationRunState struct {
	results               map[string][]dal_model.TopicRelation
	classified            map[string]bool
	limitUpAssignedTopics map[int64]struct{}
}

func NewHeatClassifyService() *HeatClassifyService {
	return &HeatClassifyService{
		topicStockCountRepo:      repo.NewTopicStockCountRepository(),
		topicHeatCache:           cache.NewTopicHeatCache(),
		stockTopicsRelationCache: cache.NewStockTopicsRelationCache(),
	}
}

func (s *ClassifyServiceImpl) ClassifyBySimpleHeat(ctx context.Context, stocks []StockQuoteInput, date string) (map[string][]dal_model.TopicRelation, error) {
	return NewHeatClassifyService().ClassifyBySimpleHeat(ctx, stocks, date)
}

// ClassifyBySimpleHeat is the online topic classification flow used by monitor/reclassify.
// It first builds today's topic heat snapshot, then classifies limit-up stocks before
// classifying the remaining strong stocks.
func (s *HeatClassifyService) ClassifyBySimpleHeat(ctx context.Context, stocks []StockQuoteInput, date string) (map[string][]dal_model.TopicRelation, error) {
	if len(stocks) == 0 {
		return nil, nil
	}

	// 1) Split the input by thresholds. Only the heat universe is used to build heat stats,
	// but only the classification universe produces final assignments.
	heatStocks, classifyStocks, tsCodes := splitStocksForHeatClassification(stocks)
	if len(heatStocks) == 0 || len(classifyStocks) == 0 {
		return nil, nil
	}

	// 2) Load candidate topic relations for involved stocks.
	relationsByStock, err := s.loadStockTopicsBatch(ctx, tsCodes)
	if err != nil {
		return nil, err
	}

	// 3) Compute today's per-topic heat snapshot (cached for debugging/inspection).
	heatMap, err := s.calcTopicHeat(ctx, heatStocks, relationsByStock, date)
	if err != nil {
		return nil, err
	}

	// 4) Assign topics in two passes: limit-up first, then strong (non-limit-up).
	state := newClassificationRunState(len(classifyStocks))
	classifyLimitUpStocks(classifyStocks, relationsByStock, heatMap, date, state)
	classifyStrongStocks(classifyStocks, relationsByStock, heatMap, date, state)

	if len(state.results) == 0 {
		return nil, nil
	}
	return state.results, nil
}

func splitStocksForHeatClassification(stocks []StockQuoteInput) ([]StockQuoteInput, []StockQuoteInput, []string) {
	heatStocks := make([]StockQuoteInput, 0, len(stocks))
	classifyStocks := make([]StockQuoteInput, 0, len(stocks))
	tsCodes := make([]string, 0, len(stocks))
	seen := make(map[string]struct{}, len(stocks))

	for _, stock := range stocks {
		if stock.TsCode == "" {
			continue
		}
		if stock.ChangePercent > HeatRisingThreshold {
			heatStocks = append(heatStocks, stock)
			if _, ok := seen[stock.TsCode]; !ok {
				tsCodes = append(tsCodes, stock.TsCode)
				seen[stock.TsCode] = struct{}{}
			}
		}
		if stock.ChangePercent > ClassifyThreshold {
			classifyStocks = append(classifyStocks, stock)
		}
	}

	return heatStocks, classifyStocks, tsCodes
}

func newClassificationRunState(size int) *classificationRunState {
	return &classificationRunState{
		results:               make(map[string][]dal_model.TopicRelation),
		classified:            make(map[string]bool, size),
		limitUpAssignedTopics: make(map[int64]struct{}),
	}
}

// classifyLimitUpStocks assigns topics for limit-up stocks only.
// It also records which topics were chosen so strong stocks can optionally align to them.
func classifyLimitUpStocks(
	classifyStocks []StockQuoteInput,
	relationsByStock map[string][]dal_model.TopicRelation,
	heatMap map[int64]*dal_model.TopicHeatInfo,
	date string,
	state *classificationRunState,
) {
	for _, stock := range classifyStocks {
		if !stock.IsLimitUp {
			continue
		}
		best, ok := pickBestTopicBySimpleHeatForLimitUp(relationsByStock[stock.TsCode], heatMap, date)
		if !ok {
			continue
		}
		state.results[stock.TsCode] = []dal_model.TopicRelation{best}
		state.classified[stock.TsCode] = true
		state.limitUpAssignedTopics[best.TopicID] = struct{}{}
	}
}

// classifyStrongStocks assigns topics for strong stocks that are not limit-up.
func classifyStrongStocks(
	classifyStocks []StockQuoteInput,
	relationsByStock map[string][]dal_model.TopicRelation,
	heatMap map[int64]*dal_model.TopicHeatInfo,
	date string,
	state *classificationRunState,
) {
	for _, stock := range classifyStocks {
		if stock.IsLimitUp || state.classified[stock.TsCode] {
			continue
		}
		best, ok := pickBestTopicBySimpleHeatForStrong(relationsByStock[stock.TsCode], heatMap, state.limitUpAssignedTopics, date)
		if !ok {
			continue
		}
		state.results[stock.TsCode] = []dal_model.TopicRelation{best}
		state.classified[stock.TsCode] = true
	}
}

// Limit-up stocks use the strictest selection path:
// manual relation > recent relation > heat ranking.
func pickBestTopicBySimpleHeatForLimitUp(relations []dal_model.TopicRelation, heatMap map[int64]*dal_model.TopicHeatInfo, date string) (dal_model.TopicRelation, bool) {
	if manual, ok := pickLatestManualRelation(relations); ok {
		return manual, true
	}
	if recent, ok := pickRecentTopic(relations, date); ok {
		return recent, true
	}
	return pickBestTopicBySimpleHeat(relations, heatMap, nil, true)
}

// Strong stocks first try topics that were already chosen by limit-up stocks.
// If none fit, they fall back to the broader heat ranking.
func pickBestTopicBySimpleHeatForStrong(relations []dal_model.TopicRelation, heatMap map[int64]*dal_model.TopicHeatInfo, limitUpAssignedTopics map[int64]struct{}, date string) (dal_model.TopicRelation, bool) {
	if manual, ok := pickLatestManualRelation(relations); ok {
		return manual, true
	}
	if recent, ok := pickRecentTopic(relations, date); ok {
		return recent, true
	}

	bestFromAssigned, ok := pickBestTopicBySimpleHeat(relations, heatMap, func(topicID int64) bool {
		_, exists := limitUpAssignedTopics[topicID]
		return exists
	}, true)
	if ok {
		return bestFromAssigned, true
	}

	return pickBestTopicBySimpleHeat(relations, heatMap, nil, false)
}

// pickBestTopicBySimpleHeat picks a topic from the candidate relations using today's topic heat stats.
//
// preferLimitUp controls tie-breaking:
// - true: prioritize more limit-up confirmations (then strong count / participation / heat score).
// - false: prioritize overall heat score (then strong count / limit-up count / participation).
func pickBestTopicBySimpleHeat(relations []dal_model.TopicRelation, heatMap map[int64]*dal_model.TopicHeatInfo, allow func(topicID int64) bool, preferLimitUp bool) (dal_model.TopicRelation, bool) {
	var (
		best              dal_model.TopicRelation
		bestLimitUp       int
		bestStockCount    int
		bestParticipation float64
		bestHeatScore     float64
		found             bool
	)

	for _, relation := range relations {
		if IsFilteredTopic(relation.TopicName) {
			continue
		}
		if allow != nil && !allow(relation.TopicID) {
			continue
		}
		heatInfo := heatMap[relation.TopicID]
		if heatInfo == nil {
			continue
		}
		if !found {
			best, bestLimitUp, bestStockCount, bestParticipation, bestHeatScore =
				relation, heatInfo.LimitUpCount, heatInfo.StockCount, heatInfo.ParticipationRate, heatInfo.HeatScore
			found = true
			continue
		}
		if shouldReplacePickedTopic(heatInfo, bestLimitUp, bestStockCount, bestParticipation, bestHeatScore, preferLimitUp) {
			best, bestLimitUp, bestStockCount, bestParticipation, bestHeatScore =
				relation, heatInfo.LimitUpCount, heatInfo.StockCount, heatInfo.ParticipationRate, heatInfo.HeatScore
		}
	}

	return best, found
}

// shouldReplacePickedTopic defines the ranking order between two candidate topics.
func shouldReplacePickedTopic(
	heatInfo *dal_model.TopicHeatInfo,
	bestLimitUp int,
	bestStockCount int,
	bestParticipation float64,
	bestHeatScore float64,
	preferLimitUp bool,
) bool {
	if preferLimitUp {
		return heatInfo.LimitUpCount > bestLimitUp ||
			(heatInfo.LimitUpCount == bestLimitUp && heatInfo.StockCount > bestStockCount) ||
			(heatInfo.LimitUpCount == bestLimitUp && heatInfo.StockCount == bestStockCount && heatInfo.ParticipationRate > bestParticipation) ||
			(heatInfo.LimitUpCount == bestLimitUp && heatInfo.StockCount == bestStockCount && heatInfo.ParticipationRate == bestParticipation && heatInfo.HeatScore > bestHeatScore)
	}

	return heatInfo.HeatScore > bestHeatScore ||
		(heatInfo.HeatScore == bestHeatScore && heatInfo.StockCount > bestStockCount) ||
		(heatInfo.HeatScore == bestHeatScore && heatInfo.StockCount == bestStockCount && heatInfo.LimitUpCount > bestLimitUp) ||
		(heatInfo.HeatScore == bestHeatScore && heatInfo.StockCount == bestStockCount && heatInfo.LimitUpCount == bestLimitUp && heatInfo.ParticipationRate > bestParticipation)
}

// calcTopicHeat builds today's topic heat snapshot using count-style indicators only.
func (s *HeatClassifyService) calcTopicHeat(ctx context.Context, stocks []StockQuoteInput, relationsByStock map[string][]dal_model.TopicRelation, date string) (map[int64]*dal_model.TopicHeatInfo, error) {
	totalCounts, err := s.topicStockCountRepo.GetAllCountMap(ctx)
	if err != nil {
		return nil, err
	}
	countsUnavailable := len(totalCounts) == 0

	stats, missingTopicIDs := collectTopicHeatStats(stocks, relationsByStock, totalCounts, countsUnavailable)

	if !countsUnavailable && len(missingTopicIDs) > 0 {
		loadedCounts, err := s.loadMissingTopicCounts(ctx, missingTopicIDs)
		if err != nil {
			return nil, err
		}
		for topicID, count := range loadedCounts {
			totalCounts[topicID] = count
		}
		for topicID := range missingTopicIDs {
			if totalCounts[topicID] <= 0 {
				delete(stats, topicID)
			}
		}
	}

	heatMap := buildTopicHeatMap(stats, totalCounts, countsUnavailable)

	if err := s.topicHeatCache.SetAll(ctx, date, heatMap); err != nil {
		return nil, err
	}
	return heatMap, nil
}

// collectTopicHeatStats aggregates per-topic stats from the heat universe.
func collectTopicHeatStats(
	stocks []StockQuoteInput,
	relationsByStock map[string][]dal_model.TopicRelation,
	totalCounts map[int64]int,
	countsUnavailable bool,
) (map[int64]*topicHeatAccumulator, map[int64]struct{}) {
	stats := make(map[int64]*topicHeatAccumulator)
	missingTopicIDs := make(map[int64]struct{})

	for _, stock := range stocks {
		for _, relation := range relationsByStock[stock.TsCode] {
			if IsFilteredTopic(relation.TopicName) {
				continue
			}
			if !countsUnavailable {
				totalRelated, ok := totalCounts[relation.TopicID]
				if !ok {
					missingTopicIDs[relation.TopicID] = struct{}{}
				} else if totalRelated <= 0 {
					continue
				}
			}

			stat := ensureTopicHeatAccumulator(stats, relation.TopicID)
			if stock.ChangePercent > ClassifyThreshold {
				stat.stockCount++
				if stock.IsLimitUp {
					stat.limitUpCount++
				}
			}
			if stock.ChangePercent > HeatRisingThreshold {
				stat.risingCount++
			}
		}
	}

	return stats, missingTopicIDs
}

// ensureTopicHeatAccumulator returns the accumulator for a topic, creating it if needed.
func ensureTopicHeatAccumulator(stats map[int64]*topicHeatAccumulator, topicID int64) *topicHeatAccumulator {
	stat := stats[topicID]
	if stat == nil {
		stat = &topicHeatAccumulator{}
		stats[topicID] = stat
	}
	return stat
}

// buildTopicHeatMap converts raw per-topic accumulators into TopicHeatInfo snapshots.
func buildTopicHeatMap(
	stats map[int64]*topicHeatAccumulator,
	totalCounts map[int64]int,
	countsUnavailable bool,
) map[int64]*dal_model.TopicHeatInfo {
	heatMap := make(map[int64]*dal_model.TopicHeatInfo, len(stats))

	for topicID, stat := range stats {
		if stat.risingCount == 0 {
			continue
		}

		info := &dal_model.TopicHeatInfo{
			TopicID:      topicID,
			StockCount:   stat.stockCount,
			LimitUpCount: stat.limitUpCount,
			RisingCount:  stat.risingCount,
		}
		if countsUnavailable {
			info.HeatScore = fallbackHeatScore(stat)
		} else {
			info.TotalRelatedStocks = totalCounts[topicID]
			if info.TotalRelatedStocks <= 0 {
				continue
			}
			info.ParticipationRate = float64(stat.stockCount) / float64(info.TotalRelatedStocks)
			info.HeatScore = participationHeatScore(stat, info.ParticipationRate)
		}
		heatMap[topicID] = info
	}

	return heatMap
}

// fallbackHeatScore is used when topic total related stock counts are unavailable.
func fallbackHeatScore(stat *topicHeatAccumulator) float64 {
	return float64(stat.limitUpCount)*1000 + float64(stat.stockCount)*100 + float64(stat.risingCount)
}

// participationHeatScore incorporates participation rate (strong_count / total_related) when available.
func participationHeatScore(stat *topicHeatAccumulator, participationRate float64) float64 {
	return float64(stat.limitUpCount)*1000 + float64(stat.stockCount)*100 + participationRate*100
}

func (s *HeatClassifyService) loadMissingTopicCounts(ctx context.Context, missingTopicIDs map[int64]struct{}) (map[int64]int, error) {
	if len(missingTopicIDs) == 0 {
		return nil, nil
	}
	if s.topicStockCountRepo == nil {
		return nil, nil
	}

	topicIDs := make([]int64, 0, len(missingTopicIDs))
	for topicID := range missingTopicIDs {
		topicIDs = append(topicIDs, topicID)
	}

	return s.topicStockCountRepo.GetCountMapByTopicIDs(ctx, topicIDs)
}

func (s *HeatClassifyService) loadStockTopicsBatch(ctx context.Context, tsCodes []string) (map[string][]dal_model.TopicRelation, error) {
	results, err := s.stockTopicsRelationCache.GetStockTopicsBatch(ctx, tsCodes)
	if err != nil {
		return nil, err
	}
	if results == nil {
		results = make(map[string][]dal_model.TopicRelation, len(tsCodes))
	}

	missing := make([]string, 0, len(tsCodes))
	for _, tsCode := range tsCodes {
		if len(results[tsCode]) == 0 {
			missing = append(missing, tsCode)
		}
	}
	if len(missing) == 0 {
		return results, nil
	}

	topicRelationRepo := repo.NewStockTopicRelationRepository()
	rawRelations, err := topicRelationRepo.GetByTsCodeBatch(ctx, missing)
	if err != nil {
		return nil, err
	}

	writeback := make(map[string][]dal_model.TopicRelation)
	for tsCode, relations := range rawRelations {
		items := make([]dal_model.TopicRelation, 0, len(relations))
		for _, relation := range relations {
			lastSeenDate := ""
			if relation.LastSeenDate != nil {
				lastSeenDate = relation.LastSeenDate.Format("2006-01-02")
			}
			items = append(items, dal_model.TopicRelation{
				TopicID:      relation.TopicID,
				TopicName:    relation.TopicName,
				Category:     relation.Category,
				Source:       relation.Source,
				HitCount:     relation.HitCount,
				LastSeenDate: lastSeenDate,
			})
		}
		results[tsCode] = items
		writeback[tsCode] = items
	}
	if len(writeback) > 0 {
		_ = s.stockTopicsRelationCache.SetStockTopicsBatch(ctx, writeback)
	}

	return results, nil
}

// If a stock touched a topic recently, reuse that relation before relying on today's heat ranking.
func pickRecentTopic(relations []dal_model.TopicRelation, date string) (dal_model.TopicRelation, bool) {
	recent := make([]dal_model.TopicRelation, 0, len(relations))
	bestHitCount := -1
	for _, relation := range relations {
		if IsFilteredTopic(relation.TopicName) || !IsRecentTopic(relation.LastSeenDate, date) {
			continue
		}
		if relation.HitCount > bestHitCount {
			recent = []dal_model.TopicRelation{relation}
			bestHitCount = relation.HitCount
			continue
		}
		if relation.HitCount == bestHitCount {
			recent = append(recent, relation)
		}
	}
	if len(recent) == 0 {
		return dal_model.TopicRelation{}, false
	}
	if len(recent) == 1 {
		return recent[0], true
	}
	idx, err := crand.Int(crand.Reader, big.NewInt(int64(len(recent))))
	if err != nil {
		return recent[0], true
	}
	return recent[idx.Int64()], true
}
