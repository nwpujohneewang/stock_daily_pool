package db

import (
	"context"
	"stock/model/dal_model"

	"gorm.io/gorm/clause"
)

var _ ConceptRepository = (*ConceptRepoImpl)(nil)

type ConceptRepository interface {
	Upsert(ctx context.Context, conceptName, conceptCode string) error
	GetAll(ctx context.Context) ([]dal_model.TushareConcept, error)
	GetByCode(ctx context.Context, conceptCode string) (*dal_model.TushareConcept, error)
	List(ctx context.Context, keyword string, page, pageSize int) ([]dal_model.TushareConcept, int64, error)
	GetUnmapped(ctx context.Context) ([]string, error)
	GetConceptMappingsByTopic(ctx context.Context, topicID int64) ([]dal_model.TopicConcept, error)
	GetConceptMapping(ctx context.Context, conceptName string) (*dal_model.TopicConcept, error)
	CreateConceptMapping(ctx context.Context, conceptName, conceptCode string, topicID int64, matchType string) error
}

type ConceptRepoImpl struct{}

func NewConceptRepository() *ConceptRepoImpl {
	return &ConceptRepoImpl{}
}

func (r ConceptRepoImpl) Upsert(ctx context.Context, conceptName, conceptCode string) error {
	sql := `INSERT INTO concepts (concept_name, concept_code) VALUES (?, ?) ON CONFLICT (concept_code) DO UPDATE SET concept_name = EXCLUDED.concept_name, updated_at = NOW()`
	return PostgresStockDB(ctx).Exec(sql, conceptName, conceptCode).Error
}

func (r ConceptRepoImpl) GetAll(ctx context.Context) ([]dal_model.TushareConcept, error) {
	var concepts []dal_model.TushareConcept
	if err := PostgresStockDB(ctx).Find(&concepts).Error; err != nil {
		return nil, err
	}
	return concepts, nil
}

func (r ConceptRepoImpl) GetByCode(ctx context.Context, conceptCode string) (*dal_model.TushareConcept, error) {
	var c dal_model.TushareConcept
	if err := PostgresStockDB(ctx).Where("concept_code = ?", conceptCode).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r ConceptRepoImpl) List(ctx context.Context, keyword string, page, pageSize int) ([]dal_model.TushareConcept, int64, error) {
	var concepts []dal_model.TushareConcept
	var total int64
	q := PostgresStockDB(ctx).Model(&dal_model.TushareConcept{})
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("concept_name ILIKE ?", like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Offset((page - 1) * pageSize).Limit(pageSize).Find(&concepts).Error; err != nil {
		return nil, 0, err
	}
	return concepts, total, nil
}

func (r ConceptRepoImpl) GetUnmapped(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (r ConceptRepoImpl) GetConceptMappingsByTopic(ctx context.Context, topicID int64) ([]dal_model.TopicConcept, error) {
	var mappings []dal_model.TopicConcept
	err := PostgresStockDB(ctx).Where("topic_id = ?", topicID).Find(&mappings).Error
	if err != nil {
		return nil, err
	}
	return mappings, nil
}

func (r ConceptRepoImpl) GetConceptMapping(ctx context.Context, conceptName string) (*dal_model.TopicConcept, error) {
	var m dal_model.TopicConcept
	err := PostgresStockDB(ctx).Where("concept_name = ?", conceptName).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r ConceptRepoImpl) CreateConceptMapping(ctx context.Context, conceptName, conceptCode string, topicID int64, matchType string) error {
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
