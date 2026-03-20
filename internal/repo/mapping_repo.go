package repo

import (
	"context"

	"gorm.io/gorm"
	"stock/internal/model"
)

type MappingRepo struct {
	db *gorm.DB
}

func NewMappingRepo(db *gorm.DB) *MappingRepo {
	return &MappingRepo{db: db}
}

func (r *MappingRepo) GetByTsCode(ctx context.Context, tsCode string) ([]model.StockTopicRelation, error) {
	var mappings []model.StockTopicRelation
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, ts_code, topic_id, source, confidence, hit_count, last_seen_date, first_seen_date, created_at, updated_at
		FROM stock_topic_relations WHERE ts_code = ?`, tsCode).Scan(&mappings).Error
	if err != nil {
		return nil, err
	}
	return mappings, nil
}

func (r *MappingRepo) Upsert(ctx context.Context, mapping *model.StockTopicRelation) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO stock_topic_relations (ts_code, topic_id, source, hit_count, first_seen_date, last_seen_date, updated_at)
		VALUES (?, ?, ?, 1, ?, ?, NOW())
		ON CONFLICT (ts_code, topic_id) DO UPDATE SET
			hit_count = stock_topic_relations.hit_count + 1,
			last_seen_date = GREATEST(stock_topic_relations.last_seen_date, EXCLUDED.last_seen_date),
			updated_at = NOW()
	`, mapping.TsCode, mapping.TopicID, mapping.Source, mapping.LastSeenDate).Error
}

func (r *MappingRepo) GetTopicMappings(ctx context.Context, topicID int64) ([]model.StockTopicRelation, error) {
	var mappings []model.StockTopicRelation
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, ts_code, topic_id, source, confidence, hit_count, last_seen_date, first_seen_date, created_at, updated_at
		FROM stock_topic_relations WHERE topic_id = ?`, topicID).Scan(&mappings).Error
	if err != nil {
		return nil, err
	}
	return mappings, nil
}

func (r *MappingRepo) Delete(ctx context.Context, tsCode string, topicID int64) error {
	return r.db.WithContext(ctx).Exec(`DELETE FROM stock_topic_relations WHERE ts_code = ? AND topic_id = ?`, tsCode, topicID).Error
}

func (r *MappingRepo) GetConceptMappingsByTopic(ctx context.Context, topicID int64) ([]model.TopicConcept, error) {
	var mappings []model.TopicConcept
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, concept_name, concept_code, topic_id, match_type, created_at
		FROM topic_concepts WHERE topic_id = ?`, topicID).Scan(&mappings).Error
	if err != nil {
		return nil, err
	}
	return mappings, nil
}

func (r *MappingRepo) GetConceptMapping(ctx context.Context, conceptName string) (*model.TopicConcept, error) {
	var m model.TopicConcept
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, concept_name, concept_code, topic_id, match_type, created_at
		FROM topic_concepts WHERE concept_name = ?`, conceptName).Scan(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MappingRepo) CreateConceptMapping(ctx context.Context, conceptName, conceptCode string, topicID int64, matchType string) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO topic_concepts (concept_name, concept_code, topic_id, match_type)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (concept_name, topic_id) DO NOTHING
	`, conceptName, conceptCode, topicID, matchType).Error
}
