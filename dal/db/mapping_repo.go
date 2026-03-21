package db

import (
	"context"
	"stock/model/dal_model"

	"gorm.io/gorm/clause"
)

var _ MappingRepository = (*MappingRepoImpl)(nil)

type MappingRepository interface {
	GetByTsCode(ctx context.Context, tsCode string) ([]dal_model.StockTopicRelation, error)
	Upsert(ctx context.Context, mapping *dal_model.StockTopicRelation) error
	GetTopicMappings(ctx context.Context, topicID int64) ([]dal_model.StockTopicRelation, error)
	Delete(ctx context.Context, tsCode string, topicID int64) error
	GetConceptMappingsByTopic(ctx context.Context, topicID int64) ([]dal_model.TopicConcept, error)
	GetConceptMapping(ctx context.Context, conceptName string) (*dal_model.TopicConcept, error)
	CreateConceptMapping(ctx context.Context, conceptName, conceptCode string, topicID int64, matchType string) error
}

type MappingRepoImpl struct{}

func NewMappingRepository() *MappingRepoImpl {
	return &MappingRepoImpl{}
}

func (r MappingRepoImpl) GetByTsCode(ctx context.Context, tsCode string) ([]dal_model.StockTopicRelation, error) {
	var mappings []dal_model.StockTopicRelation
	err := PostgresStockDB(ctx).Where("ts_code = ?", tsCode).Find(&mappings).Error
	if err != nil {
		return nil, err
	}
	return mappings, nil
}

func (r MappingRepoImpl) Upsert(ctx context.Context, mapping *dal_model.StockTopicRelation) error {
	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ts_code"}, {Name: "topic_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"hit_count", "last_seen_date", "updated_at",
		}),
	}).Create(mapping).Error
}

func (r MappingRepoImpl) GetTopicMappings(ctx context.Context, topicID int64) ([]dal_model.StockTopicRelation, error) {
	var mappings []dal_model.StockTopicRelation
	err := PostgresStockDB(ctx).Where("topic_id = ?", topicID).Find(&mappings).Error
	if err != nil {
		return nil, err
	}
	return mappings, nil
}

func (r MappingRepoImpl) Delete(ctx context.Context, tsCode string, topicID int64) error {
	return PostgresStockDB(ctx).Where("ts_code = ? AND topic_id = ?", tsCode, topicID).Delete(&dal_model.StockTopicRelation{}).Error
}

func (r MappingRepoImpl) GetConceptMappingsByTopic(ctx context.Context, topicID int64) ([]dal_model.TopicConcept, error) {
	var mappings []dal_model.TopicConcept
	err := PostgresStockDB(ctx).Where("topic_id = ?", topicID).Find(&mappings).Error
	if err != nil {
		return nil, err
	}
	return mappings, nil
}

func (r MappingRepoImpl) GetConceptMapping(ctx context.Context, conceptName string) (*dal_model.TopicConcept, error) {
	var m dal_model.TopicConcept
	err := PostgresStockDB(ctx).Where("concept_name = ?", conceptName).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r MappingRepoImpl) CreateConceptMapping(ctx context.Context, conceptName, conceptCode string, topicID int64, matchType string) error {
	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "concept_name"}, {Name: "topic_id"}},
		DoNothing: true,
	}).Create(&dal_model.TopicConcept{
		ConceptName: conceptName,
		ConceptCode: conceptCode,
		TopicID:     topicID,
		MatchType:   matchType,
	}).Error
}
