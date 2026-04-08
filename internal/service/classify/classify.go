package classify

import (
	"context"
	"stock/dal/cache"
	"stock/dal/repo"
	"stock/internal/pkg/attribution"
	"stock/model/dal_model"
	"time"
)

type ClassifyServiceImpl struct {
	weightMode attribution.WeightMode
}

func NewClassifyService() *ClassifyServiceImpl {
	return &ClassifyServiceImpl{
		weightMode: attribution.WeightModeRecent,
	}
}

func NewClassifyServiceWithMode(mode attribution.WeightMode) *ClassifyServiceImpl {
	return &ClassifyServiceImpl{
		weightMode: mode,
	}
}

func (s *ClassifyServiceImpl) ClassifyStock(ctx context.Context, tsCode string, date string, quoteTime time.Time) ([]dal_model.TopicRelation, error) {

	stockTopicRelationCache := cache.NewStockTopicsRelationCache()
	relations, err := stockTopicRelationCache.GetStockTopics(ctx, tsCode)
	if err != nil {
		return nil, err
	}

	if relations != nil && len(relations) > 0 {
		if latestManual, ok := pickLatestManualRelation(relations); ok {
			return []dal_model.TopicRelation{latestManual}, nil
		}
		attrInput := attribution.AttributionInput{
			TsCode:         tsCode,
			QuoteTime:      quoteTime,
			Date:           date,
			TopicRelations: relations,
			WeightMode:     s.weightMode,
		}
		strategy := attribution.NewStrategy(attrInput.WeightMode)
		attrOut, err := strategy.RunAttribution(ctx, attrInput)
		if err != nil || len(attrOut.FinalTopicIDs) == 0 {
			return relations, nil
		}
		result := s.attrOutputToTopicRelations(attrOut, relations)
		_ = attrOut.Confidence
		return result, nil
	}

	mappingRepo := repo.NewStockTopicRelationRepository()
	pgMappings, err := mappingRepo.GetByTsCode(ctx, tsCode)
	if err != nil {
		return nil, err
	}

	if len(pgMappings) > 0 {
		topicRepo := repo.NewTopicRepository()
		result := make([]dal_model.TopicRelation, len(pgMappings))
		for i, m := range pgMappings {
			topic, _ := topicRepo.GetByID(ctx, m.TopicID)
			topicName := ""
			if topic != nil {
				topicName = topic.Name
			}
			result[i] = dal_model.TopicRelation{
				TopicID:      m.TopicID,
				TopicName:    topicName,
				Category:     m.Category,
				Source:       m.Source,
				HitCount:     m.HitCount,
				LastSeenDate: m.LastSeenDate.Format("2006-01-02"),
				UpdatedAt:    m.UpdatedAt.Format(time.RFC3339),
			}
		}
		stockTopicRelationCache.SetStockTopics(ctx, tsCode, result)
		if latestManual, ok := pickLatestManualRelation(result); ok {
			return []dal_model.TopicRelation{latestManual}, nil
		}
		attrInput := attribution.AttributionInput{
			TsCode:         tsCode,
			QuoteTime:      quoteTime,
			Date:           date,
			TopicRelations: result,
			WeightMode:     s.weightMode,
		}
		strategy := attribution.NewStrategy(attrInput.WeightMode)
		attrOut, err := strategy.RunAttribution(ctx, attrInput)
		if err != nil || len(attrOut.FinalTopicIDs) == 0 {
			return result, nil
		}
		attrResult := s.attrOutputToTopicRelations(attrOut, result)
		_ = attrOut.Confidence
		return attrResult, nil
	}

	conceptCache := cache.NewConceptCache()
	concepts, err := conceptCache.GetStockConcepts(ctx, tsCode)
	if err != nil || concepts == nil {
		return nil, nil
	}

	// concepts found but no mapping
	return nil, nil
}

func (s *ClassifyServiceImpl) attrOutputToTopicRelations(attrOut attribution.AttributionOutput, allMappings []dal_model.TopicRelation) []dal_model.TopicRelation {
	mappingByTopicID := make(map[int64]dal_model.TopicRelation)
	for _, m := range allMappings {
		mappingByTopicID[m.TopicID] = m
	}
	result := make([]dal_model.TopicRelation, 0, len(attrOut.FinalTopicIDs))
	for _, tid := range attrOut.FinalTopicIDs {
		if m, ok := mappingByTopicID[tid]; ok {
			m.Confidence = attrOut.Confidence
			result = append(result, m)
		}
	}
	return result
}

func pickLatestManualRelation(relations []dal_model.TopicRelation) (dal_model.TopicRelation, bool) {
	var (
		latest dal_model.TopicRelation
		found  bool
		bestAt time.Time
	)
	for _, relation := range relations {
		if relation.Source != "manual" {
			continue
		}
		updatedAt, err := time.Parse(time.RFC3339, relation.UpdatedAt)
		if err != nil {
			updatedAt = time.Time{}
		}
		if !found || updatedAt.After(bestAt) {
			latest = relation
			bestAt = updatedAt
			found = true
		}
	}
	return latest, found
}

func (s *ClassifyServiceImpl) ClassifyStockBatch(ctx context.Context, tsCodes []string, date string, quoteTime time.Time) (map[string][]dal_model.TopicRelation, error) {
	relationCache := cache.NewStockTopicsRelationCache()

	l1Results, err := relationCache.GetStockTopicsBatch(ctx, tsCodes)
	if err != nil {
		return nil, err
	}

	l2Needed := make([]string, 0, len(tsCodes))
	for _, tc := range tsCodes {
		if mappings, ok := l1Results[tc]; !ok || len(mappings) == 0 {
			l2Needed = append(l2Needed, tc)
		}
	}

	if len(l2Needed) > 0 {
		topicRelationRepo := repo.NewStockTopicRelationRepository()
		l2Raw, err := topicRelationRepo.GetByTsCodeBatch(ctx, l2Needed)
		if err != nil {
			return nil, err
		}

		// Collect all unique topic IDs
		topicIDSet := make(map[int64]struct{})
		for _, relations := range l2Raw {
			for _, rel := range relations {
				topicIDSet[rel.TopicID] = struct{}{}
			}
		}
		topicIDs := make([]int64, 0, len(topicIDSet))
		for id := range topicIDSet {
			topicIDs = append(topicIDs, id)
		}

		// Batch query topic names
		topicRepo := repo.NewTopicRepository()
		topicNameMap, err := topicRepo.GetNamesByIDs(ctx, topicIDs)
		if err != nil {
			return nil, err
		}

		// Build mappings with topic names
		l2Results := make(map[string][]dal_model.TopicRelation, len(l2Raw))
		for tc, relations := range l2Raw {
			mappings := make([]dal_model.TopicRelation, 0, len(relations))
			for _, rel := range relations {
				mappings = append(mappings, dal_model.TopicRelation{
					TopicID:      rel.TopicID,
					TopicName:    topicNameMap[rel.TopicID],
					Category:     rel.Category,
					Source:       rel.Source,
					HitCount:     rel.HitCount,
					LastSeenDate: rel.LastSeenDate.Format("2006-01-02"),
				})
			}
			l2Results[tc] = mappings
			l1Results[tc] = mappings
		}

		// Batch writeback to cache
		if len(l2Results) > 0 {
			_ = relationCache.SetStockTopicsBatch(ctx, l2Results)
		}
	}

	for tc, mappings := range l1Results {
		if len(mappings) == 0 {
			continue
		}

		var result []dal_model.TopicRelation
		if latestManual, ok := pickLatestManualRelation(mappings); ok {
			result = []dal_model.TopicRelation{latestManual}
			//s.saveEvidence(ctx, date, tc, latestManual.TopicID, "L1_REDIS", "MANUAL", mappings, "", 1.0)
		} else {
			attrInput := attribution.AttributionInput{
				TsCode:         tc,
				QuoteTime:      quoteTime,
				Date:           date,
				TopicRelations: mappings,
				WeightMode:     s.weightMode,
			}
			strategy := attribution.NewStrategy(attrInput.WeightMode)
			attrOut, err := strategy.RunAttribution(ctx, attrInput)
			if err != nil || len(attrOut.FinalTopicIDs) == 0 {
				if len(mappings) > 0 {
					//s.saveEvidence(ctx, date, tc, mappings[0].TopicID, "L1_REDIS", "JIUYAN_ATTR", mappings, "", 0.5)
					result = mappings[:1]
				}
			} else {
				result = s.attrOutputToTopicRelations(attrOut, mappings)
				//s.saveEvidence(ctx, date, tc, result[0].TopicID, "L1_REDIS", "JIUYAN_ATTR", mappings, "", attrOut.Confidence)
			}
		}
		l1Results[tc] = result
	}

	return l1Results, nil
}

func (s *ClassifyServiceImpl) NormalizeTopicName(ctx context.Context, rawName string) (int64, string, bool, error) {
	topicRepo := repo.NewTopicRepository()
	topic, err := topicRepo.GetByName(ctx, rawName)
	if err == nil && topic != nil {
		return topic.ID, topic.Name, false, nil
	}
	return 0, "", true, nil
}
