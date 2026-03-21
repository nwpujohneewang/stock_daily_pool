package repo

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	err := r.db.WithContext(ctx).Where("ts_code = ?", tsCode).Find(&mappings).Error
	if err != nil {
		return nil, err
	}
	return mappings, nil
}

func (r *MappingRepo) Upsert(ctx context.Context, mapping *model.StockTopicRelation) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ts_code"}, {Name: "topic_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"hit_count", "last_seen_date", "updated_at",
		}),
	}).Create(mapping).Error
}

func (r *MappingRepo) GetTopicMappings(ctx context.Context, topicID int64) ([]model.StockTopicRelation, error) {
	var mappings []model.StockTopicRelation
	err := r.db.WithContext(ctx).Where("topic_id = ?", topicID).Find(&mappings).Error
	if err != nil {
		return nil, err
	}
	return mappings, nil
}

func (r *MappingRepo) Delete(ctx context.Context, tsCode string, topicID int64) error {
	return r.db.WithContext(ctx).Where("ts_code = ? AND topic_id = ?", tsCode, topicID).Delete(&model.StockTopicRelation{}).Error
}

func (r *MappingRepo) GetConceptMappingsByTopic(ctx context.Context, topicID int64) ([]model.TopicConcept, error) {
	var mappings []model.TopicConcept
	err := r.db.WithContext(ctx).Where("topic_id = ?", topicID).Find(&mappings).Error
	if err != nil {
		return nil, err
	}
	return mappings, nil
}

func (r *MappingRepo) GetConceptMapping(ctx context.Context, conceptName string) (*model.TopicConcept, error) {
	var m model.TopicConcept
	err := r.db.WithContext(ctx).Where("concept_name = ?", conceptName).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MappingRepo) CreateConceptMapping(ctx context.Context, conceptName, conceptCode string, topicID int64, matchType string) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "concept_name"}, {Name: "topic_id"}},
		DoNothing: true,
	}).Create(&model.TopicConcept{
		ConceptName: conceptName,
		ConceptCode: conceptCode,
		TopicID:     topicID,
		MatchType:   matchType,
	}).Error
}
