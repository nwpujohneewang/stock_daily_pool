package classify

import (
	"context"
	crand "crypto/rand"
	"math/big"
	"stock/dal/cache"
	"stock/dal/repo"
	"stock/model/dal_model"
)

type StockQuoteInput struct {
	TsCode        string
	ChangePercent float64
	IsLimitUp     bool
}

type HeatClassifyService struct {
	topicStockCountRepo      repo.TopicStockCountRepository
	topicHeatCache           cache.TopicHeatCacheInterface
	stockTopicsRelationCache cache.StockTopicsRelationCacheInterface
	classificationCache      cache.ClassificationCacheInterface
}

type topicHeatAccumulator struct {
	stockCount   int
	limitUpCount int
	risingCount  int
	totalGain    float64
}

type topicGroupStats struct {
	stockCount   int
	limitUpCount int
	totalChange  float64
}

func NewHeatClassifyService() *HeatClassifyService {
	return &HeatClassifyService{
		topicStockCountRepo:      repo.NewTopicStockCountRepository(),
		topicHeatCache:           cache.NewTopicHeatCache(),
		stockTopicsRelationCache: cache.NewStockTopicsRelationCache(),
		classificationCache:      cache.NewClassificationCache(),
	}
}

func (s *ClassifyServiceImpl) ClassifyBySimpleHeat(ctx context.Context, stocks []StockQuoteInput, date string) (map[string][]dal_model.TopicRelation, error) {
	return NewHeatClassifyService().ClassifyBySimpleHeat(ctx, stocks, date)
}

func (s *HeatClassifyService) ClassifyBySimpleHeat(ctx context.Context, stocks []StockQuoteInput, date string) (map[string][]dal_model.TopicRelation, error) {
	if len(stocks) == 0 {
		return nil, nil
	}

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
	if len(heatStocks) == 0 || len(classifyStocks) == 0 {
		return nil, nil
	}

	relationsByStock, err := s.loadStockTopicsBatch(ctx, tsCodes)
	if err != nil {
		return nil, err
	}
	heatMap, err := s.calcTopicHeat(ctx, heatStocks, relationsByStock, date)
	if err != nil {
		return nil, err
	}

	results := make(map[string][]dal_model.TopicRelation)
	classified := make(map[string]bool, len(classifyStocks))
	limitUpAssignedTopics := make(map[int64]struct{})

	for _, stock := range classifyStocks {
		if !stock.IsLimitUp {
			continue
		}
		best, ok := pickBestTopicBySimpleHeatForLimitUp(relationsByStock[stock.TsCode], heatMap, date)
		if !ok {
			continue
		}
		results[stock.TsCode] = []dal_model.TopicRelation{best}
		classified[stock.TsCode] = true
		limitUpAssignedTopics[best.TopicID] = struct{}{}
	}

	for _, stock := range classifyStocks {
		if stock.IsLimitUp || classified[stock.TsCode] {
			continue
		}
		best, ok := pickBestTopicBySimpleHeatForStrong(relationsByStock[stock.TsCode], heatMap, limitUpAssignedTopics, date)
		if !ok {
			continue
		}
		results[stock.TsCode] = []dal_model.TopicRelation{best}
		classified[stock.TsCode] = true
	}

	if len(results) == 0 {
		return nil, nil
	}
	return results, nil
}

func sampleSupportFactor(totalRelatedStocks, stockCount int) float64 {
	if totalRelatedStocks <= 0 {
		return 0
	}
	support := float64(totalRelatedStocks) / float64(totalRelatedStocks+20)
	if stockCount > 0 {
		support *= float64(stockCount) / float64(stockCount+2)
	}
	if support < 0 {
		return 0
	}
	if support > 1 {
		return 1
	}
	return support
}

func effectiveOverallStrength(heatInfo *dal_model.TopicHeatInfo) float64 {
	if heatInfo == nil {
		return 0
	}
	if heatInfo.OverallStrength > 0 {
		return heatInfo.OverallStrength
	}
	strength := heatInfo.AvgGain
	if heatInfo.TotalRelatedStocks > 0 {
		strength *= sampleSupportFactor(heatInfo.TotalRelatedStocks, heatInfo.RisingCount)
	}
	return strength
}

func topicPriorityScore(heatInfo *dal_model.TopicHeatInfo, historyScore float64) float64 {
	if heatInfo == nil {
		return 0
	}
	return float64(heatInfo.LimitUpCount)*1000 + float64(heatInfo.StockCount)*100 + effectiveOverallStrength(heatInfo) + historyScore
}

func pickBestTopicBySimpleHeatForLimitUp(relations []dal_model.TopicRelation, heatMap map[int64]*dal_model.TopicHeatInfo, date string) (dal_model.TopicRelation, bool) {
	if manual, ok := pickLatestManualRelation(relations); ok {
		return manual, true
	}
	if recent, ok := pickRecentTopic(relations, date); ok {
		return recent, true
	}
	return pickBestTopicBySimpleHeat(relations, heatMap, nil, true)
}

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

func pickBestTopicBySimpleHeat(relations []dal_model.TopicRelation, heatMap map[int64]*dal_model.TopicHeatInfo, allow func(topicID int64) bool, preferLimitUp bool) (dal_model.TopicRelation, bool) {
	var (
		best        dal_model.TopicRelation
		bestLimitUp int
		bestAvgGain float64
		found       bool
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
			best = relation
			bestLimitUp = heatInfo.LimitUpCount
			bestAvgGain = heatInfo.AvgGain
			found = true
			continue
		}
		if preferLimitUp {
			if heatInfo.LimitUpCount > bestLimitUp || (heatInfo.LimitUpCount == bestLimitUp && heatInfo.AvgGain > bestAvgGain) {
				best = relation
				bestLimitUp = heatInfo.LimitUpCount
				bestAvgGain = heatInfo.AvgGain
			}
			continue
		}
		if heatInfo.AvgGain > bestAvgGain {
			best = relation
			bestLimitUp = heatInfo.LimitUpCount
			bestAvgGain = heatInfo.AvgGain
		}
	}

	return best, found
}

func (s *HeatClassifyService) calcTopicHeat(ctx context.Context, stocks []StockQuoteInput, relationsByStock map[string][]dal_model.TopicRelation, date string) (map[int64]*dal_model.TopicHeatInfo, error) {
	totalCounts, err := s.topicStockCountRepo.GetAllCountMap(ctx)
	if err != nil {
		return nil, err
	}
	countsUnavailable := len(totalCounts) == 0

	stats := make(map[int64]*topicHeatAccumulator)
	missingTopicIDs := make(map[int64]struct{})
	for _, stock := range stocks {
		relations := relationsByStock[stock.TsCode]
		for _, relation := range relations {
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

			stat := stats[relation.TopicID]
			if stat == nil {
				stat = &topicHeatAccumulator{}
				stats[relation.TopicID] = stat
			}
			if stock.ChangePercent > ClassifyThreshold {
				stat.stockCount++
				if stock.IsLimitUp {
					stat.limitUpCount++
				}
			}
			if stock.ChangePercent > HeatRisingThreshold {
				stat.risingCount++
			}
			stat.totalGain += stock.ChangePercent
		}
	}

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

	heatMap := make(map[int64]*dal_model.TopicHeatInfo, len(stats))
	for topicID, stat := range stats {
		if stat.risingCount == 0 {
			continue
		}

		avgGain := stat.totalGain / float64(stat.risingCount)

		info := &dal_model.TopicHeatInfo{
			TopicID:      topicID,
			StockCount:   stat.stockCount,
			LimitUpCount: stat.limitUpCount,
			RisingCount:  stat.risingCount,
			AvgGain:      avgGain,
		}
		if countsUnavailable {
			info.OverallStrength = avgGain
			info.HeatScore = float64(stat.limitUpCount)*1000 + float64(stat.stockCount)*100 + info.OverallStrength
		} else {
			info.TotalRelatedStocks = totalCounts[topicID]
			if info.TotalRelatedStocks <= 0 {
				continue
			}
			info.ParticipationRate = float64(stat.stockCount) / float64(info.TotalRelatedStocks)
			info.OverallStrength = avgGain * sampleSupportFactor(info.TotalRelatedStocks, stat.risingCount)
			info.HeatScore = float64(stat.limitUpCount)*1000 + float64(stat.stockCount)*100 + info.OverallStrength
		}
		heatMap[topicID] = info
	}

	if err := s.topicHeatCache.SetAll(ctx, date, heatMap); err != nil {
		return nil, err
	}
	return heatMap, nil
}

func (s *HeatClassifyService) classifyLimitUp(stocks []StockQuoteInput, relationsByStock map[string][]dal_model.TopicRelation, heatMap map[int64]*dal_model.TopicHeatInfo, date string, results map[string][]dal_model.TopicRelation, classified map[string]bool, activeTopics map[int64]struct{}) {
	for _, stock := range stocks {
		if !stock.IsLimitUp {
			continue
		}
		best, score, ok := pickBestTopic(relationsByStock[stock.TsCode], heatMap, activeTopics, false, date)
		if !ok {
			continue
		}
		best.Confidence = score
		results[stock.TsCode] = []dal_model.TopicRelation{best}
		classified[stock.TsCode] = true
		activeTopics[best.TopicID] = struct{}{}
	}
	for topicID := range collectStrongHeatTopics(heatMap) {
		activeTopics[topicID] = struct{}{}
	}
}

func (s *HeatClassifyService) classifyNonLimitUp(stocks []StockQuoteInput, relationsByStock map[string][]dal_model.TopicRelation, heatMap map[int64]*dal_model.TopicHeatInfo, date string, results map[string][]dal_model.TopicRelation, classified map[string]bool, activeTopics map[int64]struct{}) {
	if len(activeTopics) == 0 {
		return
	}
	for _, stock := range stocks {
		if stock.IsLimitUp || classified[stock.TsCode] {
			continue
		}
		best, score, ok := pickBestTopic(relationsByStock[stock.TsCode], heatMap, activeTopics, true, date)
		if !ok || score <= 0.1 {
			continue
		}
		best.Confidence = score
		results[stock.TsCode] = []dal_model.TopicRelation{best}
		classified[stock.TsCode] = true
	}
}

func (s *HeatClassifyService) classifyRemaining(stocks []StockQuoteInput, relationsByStock map[string][]dal_model.TopicRelation, heatMap map[int64]*dal_model.TopicHeatInfo, date string, results map[string][]dal_model.TopicRelation, classified map[string]bool) {
	for _, stock := range stocks {
		if classified[stock.TsCode] {
			continue
		}
		best, score, ok := pickBestTopic(relationsByStock[stock.TsCode], heatMap, nil, false, date)
		if !ok {
			continue
		}
		best.Confidence = score
		results[stock.TsCode] = []dal_model.TopicRelation{best}
		classified[stock.TsCode] = true
	}
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

func collectStrongHeatTopics(heatMap map[int64]*dal_model.TopicHeatInfo) map[int64]struct{} {
	activeTopics := make(map[int64]struct{})
	for topicID, heatInfo := range heatMap {
		if heatInfo == nil {
			continue
		}
		if heatInfo.StockCount >= 2 && heatInfo.HeatScore >= ActiveHeatThreshold {
			activeTopics[topicID] = struct{}{}
		}
	}
	return activeTopics
}

func pickBestTopic(relations []dal_model.TopicRelation, heatMap map[int64]*dal_model.TopicHeatInfo, activeTopics map[int64]struct{}, restrictActive bool, date string) (dal_model.TopicRelation, float64, bool) {
	if manual, ok := pickLatestManualRelation(relations); ok {
		return manual, 1, true
	}
	if recent, ok := pickRecentTopic(relations, date); ok {
		return recent, 1, true
	}

	var (
		best      dal_model.TopicRelation
		bestScore float64
		found     bool
	)

	for _, relation := range relations {
		if IsFilteredTopic(relation.TopicName) {
			continue
		}
		if restrictActive {
			if _, ok := activeTopics[relation.TopicID]; !ok {
				continue
			}
		}
		heatInfo := heatMap[relation.TopicID]
		if heatInfo == nil {
			continue
		}
		score := topicPriorityScore(heatInfo, CalcHistoryScore(relation.LastSeenDate, relation.HitCount, date))
		if !found || score > bestScore {
			best = relation
			bestScore = score
			found = true
		}
	}

	return best, bestScore, found
}
