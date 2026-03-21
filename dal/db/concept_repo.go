package db

import (
	"context"
	"stock/model/dal_model"
)

var _ ConceptRepository = (*ConceptRepoImpl)(nil)

type ConceptRepository interface {
	Upsert(ctx context.Context, conceptName, conceptCode string) error
	GetAll(ctx context.Context) ([]dal_model.TushareConcept, error)
	GetByCode(ctx context.Context, conceptCode string) (*dal_model.TushareConcept, error)
	List(ctx context.Context, keyword string, page, pageSize int) ([]dal_model.TushareConcept, int64, error)
	GetUnmapped(ctx context.Context) ([]string, error)
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
