package repo

import (
	"context"
	"stock/dal/cache"
	"stock/dal/dao"
	"stock/model/dal_model"
	"time"
)

type StockTopicRelationRepository interface {
	GetByTsCode(ctx context.Context, tsCode string) ([]dal_model.StockTopicRelation, error)
	GetByTsCodeBatch(ctx context.Context, tsCodes []string) (map[string][]dal_model.StockTopicRelation, error)
	Upsert(ctx context.Context, m *dal_model.StockTopicRelation) error
	UpsertBatch(ctx context.Context, mappings []dal_model.StockTopicRelation) error
	BulkUpsertAccumulate(ctx context.Context, mappings []dal_model.StockTopicRelation) error
	BulkUpsertReplace(ctx context.Context, mappings []dal_model.StockTopicRelation) error
	GetTopicMappings(ctx context.Context, topicID int64) ([]dal_model.StockTopicRelation, error)
	Delete(ctx context.Context, tsCode string, topicID int64) error
	GetByTopicIDs(ctx context.Context, topicIDs []int64) ([]dal_model.StockTopicRelation, error)
	WarmupByTsCodes(ctx context.Context, tsCodes []string) error
}

type stockTopicRelationRepoImpl struct {
	dao   dao.StockTopicRelationDAO
	cache cache.StockTopicsRelationCacheInterface
}

func NewStockTopicRelationRepository() StockTopicRelationRepository {
	return &stockTopicRelationRepoImpl{
		dao:   dao.NewStockTopicRelationDAO(),
		cache: cache.NewStockTopicsRelationCache(),
	}
}

func (r *stockTopicRelationRepoImpl) GetByTsCode(ctx context.Context, tsCode string) ([]dal_model.StockTopicRelation, error) {
	cached, err := r.cache.GetStockTopics(ctx, tsCode)
	if err != nil {
		return nil, err
	}
	if len(cached) > 0 {
		return topicRelationsToStockTopicRelations(tsCode, cached), nil
	}

	relations, err := r.dao.GetByTsCode(ctx, tsCode)
	if err != nil {
		return nil, err
	}
	if len(relations) > 0 {
		_ = r.cache.SetStockTopics(ctx, tsCode, stockTopicRelationsToTopicRelations(relations))
	}
	return relations, nil
}

func (r *stockTopicRelationRepoImpl) GetByTsCodeBatch(ctx context.Context, tsCodes []string) (map[string][]dal_model.StockTopicRelation, error) {
	if len(tsCodes) == 0 {
		return nil, nil
	}

	cached, err := r.cache.GetStockTopicsBatch(ctx, tsCodes)
	if err != nil {
		return nil, err
	}
	if cached == nil {
		cached = make(map[string][]dal_model.TopicRelation, len(tsCodes))
	}

	result := make(map[string][]dal_model.StockTopicRelation, len(tsCodes))
	missing := make([]string, 0)
	for _, tsCode := range tsCodes {
		if relations := cached[tsCode]; len(relations) > 0 {
			result[tsCode] = topicRelationsToStockTopicRelations(tsCode, relations)
			continue
		}
		missing = append(missing, tsCode)
	}
	if len(missing) == 0 {
		return result, nil
	}

	loaded, err := r.dao.GetByTsCodeBatch(ctx, missing)
	if err != nil {
		return nil, err
	}
	writeback := make(map[string][]dal_model.TopicRelation, len(loaded))
	for tsCode, relations := range loaded {
		if len(relations) == 0 {
			continue
		}
		result[tsCode] = relations
		writeback[tsCode] = stockTopicRelationsToTopicRelations(relations)
	}
	if len(writeback) > 0 {
		if err := r.cache.SetStockTopicsBatch(ctx, writeback); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (r *stockTopicRelationRepoImpl) Upsert(ctx context.Context, m *dal_model.StockTopicRelation) error {
	if err := r.dao.Upsert(ctx, m); err != nil {
		return err
	}
	return r.refreshTsCodes(ctx, []string{m.TsCode})
}

func (r *stockTopicRelationRepoImpl) UpsertBatch(ctx context.Context, mappings []dal_model.StockTopicRelation) error {
	if err := r.dao.UpsertBatch(ctx, mappings); err != nil {
		return err
	}
	return r.refreshTsCodes(ctx, uniqueTsCodes(mappings))
}

func (r *stockTopicRelationRepoImpl) BulkUpsertAccumulate(ctx context.Context, mappings []dal_model.StockTopicRelation) error {
	if err := r.dao.BulkUpsertAccumulate(ctx, mappings); err != nil {
		return err
	}
	return r.refreshTsCodes(ctx, uniqueTsCodes(mappings))
}

func (r *stockTopicRelationRepoImpl) BulkUpsertReplace(ctx context.Context, mappings []dal_model.StockTopicRelation) error {
	if err := r.dao.BulkUpsertReplace(ctx, mappings); err != nil {
		return err
	}
	return r.refreshTsCodes(ctx, uniqueTsCodes(mappings))
}

func (r *stockTopicRelationRepoImpl) GetTopicMappings(ctx context.Context, topicID int64) ([]dal_model.StockTopicRelation, error) {
	return r.dao.GetTopicMappings(ctx, topicID)
}

func (r *stockTopicRelationRepoImpl) Delete(ctx context.Context, tsCode string, topicID int64) error {
	if err := r.dao.Delete(ctx, tsCode, topicID); err != nil {
		return err
	}
	return r.warmupSingle(ctx, tsCode)
}

func (r *stockTopicRelationRepoImpl) GetByTopicIDs(ctx context.Context, topicIDs []int64) ([]dal_model.StockTopicRelation, error) {
	return r.dao.GetByTopicIDs(ctx, topicIDs)
}

func (r *stockTopicRelationRepoImpl) WarmupByTsCodes(ctx context.Context, tsCodes []string) error {
	_, err := r.GetByTsCodeBatch(ctx, tsCodes)
	return err
}

func (r *stockTopicRelationRepoImpl) refreshTsCodes(ctx context.Context, tsCodes []string) error {
	if len(tsCodes) == 0 {
		return nil
	}
	loaded, err := r.dao.GetByTsCodeBatch(ctx, tsCodes)
	if err != nil {
		return err
	}
	writeback := make(map[string][]dal_model.TopicRelation, len(tsCodes))
	for _, tsCode := range tsCodes {
		writeback[tsCode] = stockTopicRelationsToTopicRelations(loaded[tsCode])
	}
	return r.cache.SetStockTopicsBatch(ctx, writeback)
}

func (r *stockTopicRelationRepoImpl) warmupSingle(ctx context.Context, tsCode string) error {
	_, err := r.GetByTsCode(ctx, tsCode)
	return err
}

func uniqueTsCodes(mappings []dal_model.StockTopicRelation) []string {
	seen := make(map[string]struct{}, len(mappings))
	result := make([]string, 0, len(mappings))
	for _, mapping := range mappings {
		if mapping.TsCode == "" {
			continue
		}
		if _, ok := seen[mapping.TsCode]; ok {
			continue
		}
		seen[mapping.TsCode] = struct{}{}
		result = append(result, mapping.TsCode)
	}
	return result
}

func stockTopicRelationsToTopicRelations(relations []dal_model.StockTopicRelation) []dal_model.TopicRelation {
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
			UpdatedAt:    relation.UpdatedAt.Format(time.RFC3339),
		})
	}
	return items
}

func topicRelationsToStockTopicRelations(tsCode string, relations []dal_model.TopicRelation) []dal_model.StockTopicRelation {
	items := make([]dal_model.StockTopicRelation, 0, len(relations))
	for _, relation := range relations {
		var lastSeenDate *time.Time
		if relation.LastSeenDate != "" {
			if parsed, err := time.Parse("2006-01-02", relation.LastSeenDate); err == nil {
				lastSeenDate = &parsed
			}
		}
		var updatedAt time.Time
		if relation.UpdatedAt != "" {
			if parsed, err := time.Parse(time.RFC3339, relation.UpdatedAt); err == nil {
				updatedAt = parsed
			}
		}
		items = append(items, dal_model.StockTopicRelation{
			TsCode:       tsCode,
			TopicID:      relation.TopicID,
			TopicName:    relation.TopicName,
			Category:     relation.Category,
			Source:       relation.Source,
			HitCount:     relation.HitCount,
			LastSeenDate: lastSeenDate,
			UpdatedAt:    updatedAt,
		})
	}
	return items
}
