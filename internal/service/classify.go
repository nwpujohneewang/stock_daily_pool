package service

import (
	"context"
	"encoding/json"
	"log"
	"stock/dal/db"
	"stock/dal/redis"
	"stock/internal/pkg/attribution"
	"stock/model/dal_model"
	"time"
)

type ClassifyService struct {
	logger     *log.Logger
	weightMode attribution.WeightMode
}

func NewClassifyService() *ClassifyService {
	return &ClassifyService{
		logger:     log.Default(),
		weightMode: attribution.WeightModeNormal,
	}
}

func NewClassifyServiceWithMode(mode attribution.WeightMode) *ClassifyService {
	return &ClassifyService{
		logger:     log.Default(),
		weightMode: mode,
	}
}

func (s *ClassifyService) ClassifyStock(ctx context.Context, tsCode string, date string, quoteTime time.Time) ([]dal_model.TopicMapping, error) {

	mappingCache := redis.NewMappingCache()
	mappings, err := mappingCache.GetStockTopics(ctx, tsCode)
	if err != nil {
		return nil, err
	}

	if mappings != nil && len(mappings) > 0 {
		for _, m := range mappings {
			if m.Source == "manual" {
				s.saveEvidence(ctx, date, tsCode, m.TopicID, "L1_REDIS", "MANUAL", mappings, "", 1.0)
				return []dal_model.TopicMapping{m}, nil
			}
		}
		attrInput := attribution.AttributionInput{
			TsCode:            tsCode,
			QuoteTime:         quoteTime,
			Date:              date,
			CandidateMappings: mappings,
			WeightMode:        s.weightMode,
		}
		attrOut, err := attribution.RunAttribution(ctx, attrInput)
		if err != nil || len(attrOut.FinalTopicIDs) == 0 {
			s.saveEvidence(ctx, date, tsCode, mappings[0].TopicID, "L1_REDIS", "JIUYAN_ATTR", mappings, "", 0.5)
			return mappings, nil
		}
		result := s.attrOutputToTopicMappings(attrOut, mappings)
		confidence := attrOut.Confidence
		s.saveEvidence(ctx, date, tsCode, result[0].TopicID, "L1_REDIS", "JIUYAN_ATTR", mappings, "", confidence)
		return result, nil
	}

	mappingRepo := db.NewMappingRepository()
	pgMappings, err := mappingRepo.GetByTsCode(ctx, tsCode)
	if err != nil {
		return nil, err
	}

	if len(pgMappings) > 0 {
		topicRepo := db.NewTopicRepository()
		result := make([]dal_model.TopicMapping, len(pgMappings))
		for i, m := range pgMappings {
			topic, _ := topicRepo.GetByID(ctx, m.TopicID)
			topicName := ""
			if topic != nil {
				topicName = topic.Name
			}
			result[i] = dal_model.TopicMapping{
				TopicID:      m.TopicID,
				TopicName:    topicName,
				Source:       m.Source,
				HitCount:     m.HitCount,
				LastSeenDate: m.LastSeenDate.Format("2006-01-02"),
			}
		}
		mappingCache.SetStockTopics(ctx, tsCode, result)
		attrInput := attribution.AttributionInput{
			TsCode:            tsCode,
			QuoteTime:         quoteTime,
			Date:              date,
			CandidateMappings: result,
			WeightMode:        s.weightMode,
		}
		attrOut, err := attribution.RunAttribution(ctx, attrInput)
		if err != nil || len(attrOut.FinalTopicIDs) == 0 {
			s.saveEvidence(ctx, date, tsCode, result[0].TopicID, "L2_PG_JIUYAN", "JIUYAN_ATTR", result, "", 0.5)
			return result, nil
		}
		attrResult := s.attrOutputToTopicMappings(attrOut, result)
		confidence := attrOut.Confidence
		s.saveEvidence(ctx, date, tsCode, attrResult[0].TopicID, "L2_PG_JIUYAN", "JIUYAN_ATTR", result, "", confidence)
		return attrResult, nil
	}

	conceptCache := redis.NewConceptCache()
	concepts, err := conceptCache.GetStockConcepts(ctx, tsCode)
	if err != nil || concepts == nil {
		s.saveEvidence(ctx, date, tsCode, 0, "L3_PG_CONCEPT", "CONCEPT_ATTR", nil, "no concepts found", 0.0)
		return nil, nil
	}

	s.saveEvidence(ctx, date, tsCode, 0, "L3_PG_CONCEPT", "CONCEPT_ATTR", nil, "concepts found but no mapping", 0.0)
	return nil, nil
}

func (s *ClassifyService) attrOutputToTopicMappings(attrOut attribution.AttributionOutput, allMappings []dal_model.TopicMapping) []dal_model.TopicMapping {
	mappingByTopicID := make(map[int64]dal_model.TopicMapping)
	for _, m := range allMappings {
		mappingByTopicID[m.TopicID] = m
	}
	result := make([]dal_model.TopicMapping, 0, len(attrOut.FinalTopicIDs))
	for _, tid := range attrOut.FinalTopicIDs {
		if m, ok := mappingByTopicID[tid]; ok {
			m.Confidence = attrOut.Confidence
			result = append(result, m)
		}
	}
	return result
}

func (s *ClassifyService) saveEvidence(ctx context.Context, date, tsCode string, topicID int64, layer, strategy string, candidates []dal_model.TopicMapping, evidenceText string, confidence float64) {
	evidenceRepo := db.NewEvidenceRepository()
	if evidenceRepo == nil {
		return
	}
	candidateScores, _ := json.Marshal(candidates)
	parsedDate, _ := time.Parse("2006-01-02", date)
	e := dal_model.ClassificationAuditLog{
		Date:            parsedDate,
		TsCode:          tsCode,
		TopicID:         &topicID,
		ClassifyLayer:   layer,
		Strategy:        strategy,
		CandidateScores: candidateScores,
		EvidenceText:    &evidenceText,
		Confidence:      &confidence,
	}
	evidenceRepo.Create(ctx, e)
}

func (s *ClassifyService) ClassifyStockBatch(ctx context.Context, tsCodes []string, date string, quoteTime time.Time) (map[string][]dal_model.TopicMapping, error) {
	mappingCache := redis.NewMappingCache()

	l1Results, err := mappingCache.GetStockTopicsBatch(ctx, tsCodes)
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
		mappingRepo := db.NewMappingRepository()
		l2Raw, err := mappingRepo.GetByTsCodeBatch(ctx, l2Needed)
		if err != nil {
			return nil, err
		}

		topicRepo := db.NewTopicRepository()
		topicNameMap := make(map[int64]string)

		for tc, relations := range l2Raw {
			mappings := make([]dal_model.TopicMapping, 0, len(relations))
			for _, rel := range relations {
				name := topicNameMap[rel.TopicID]
				if name == "" {
					if t, _ := topicRepo.GetByID(ctx, rel.TopicID); t != nil {
						topicNameMap[rel.TopicID] = t.Name
						name = t.Name
					}
				}
				mappings = append(mappings, dal_model.TopicMapping{
					TopicID:      rel.TopicID,
					TopicName:    name,
					Source:       rel.Source,
					HitCount:     rel.HitCount,
					LastSeenDate: rel.LastSeenDate.Format("2006-01-02"),
				})
			}
			l1Results[tc] = mappings
		}
	}

	for tc, mappings := range l1Results {
		if len(mappings) == 0 {
			continue
		}

		var hasManual *dal_model.TopicMapping
		for i := range mappings {
			if mappings[i].Source == "manual" {
				hasManual = &mappings[i]
				break
			}
		}

		var result []dal_model.TopicMapping
		if hasManual != nil {
			result = []dal_model.TopicMapping{*hasManual}
			s.saveEvidence(ctx, date, tc, hasManual.TopicID, "L1_REDIS", "MANUAL", mappings, "", 1.0)
		} else {
			attrInput := attribution.AttributionInput{
				TsCode:            tc,
				QuoteTime:         quoteTime,
				Date:              date,
				CandidateMappings: mappings,
				WeightMode:        s.weightMode,
			}
			attrOut, err := attribution.RunAttribution(ctx, attrInput)
			if err != nil || len(attrOut.FinalTopicIDs) == 0 {
				if len(mappings) > 0 {
					s.saveEvidence(ctx, date, tc, mappings[0].TopicID, "L1_REDIS", "JIUYAN_ATTR", mappings, "", 0.5)
					result = mappings[:1]
				}
			} else {
				result = s.attrOutputToTopicMappings(attrOut, mappings)
				s.saveEvidence(ctx, date, tc, result[0].TopicID, "L1_REDIS", "JIUYAN_ATTR", mappings, "", attrOut.Confidence)
			}
		}
		l1Results[tc] = result
	}

	return l1Results, nil
}

func (s *ClassifyService) NormalizeTopicName(ctx context.Context, rawName string) (int64, string, bool, error) {
	topicRepo := db.NewTopicRepository()
	topic, err := topicRepo.GetByName(ctx, rawName)
	if err == nil && topic != nil {
		return topic.ID, topic.Name, false, nil
	}
	return 0, "", true, nil
}
