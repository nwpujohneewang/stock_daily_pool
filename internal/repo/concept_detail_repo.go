package repo

import (
	"context"

	"gorm.io/gorm"
)

type ConceptDetail struct {
	ID          int64  `gorm:"column:id"`
	TsCode      string `gorm:"column:ts_code"`
	ConceptName string `gorm:"column:concept_name"`
	ConceptCode string `gorm:"column:concept_code"`
	Source      string `gorm:"column:source"`
}

type ConceptDetailRepo struct {
	db *gorm.DB
}

func NewConceptDetailRepo(db *gorm.DB) *ConceptDetailRepo {
	return &ConceptDetailRepo{db: db}
}

func (r *ConceptDetailRepo) Upsert(ctx context.Context, tsCode, conceptName, conceptCode string) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO tushare_concept_details (ts_code, concept_name, concept_code, source)
		VALUES (?, ?, ?, 'tushare')
		ON CONFLICT (ts_code, concept_name) DO NOTHING
	`, tsCode, conceptName, conceptCode).Error
}

func (r *ConceptDetailRepo) GetByStock(ctx context.Context, tsCode string) ([]ConceptDetail, error) {
	var details []ConceptDetail
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, ts_code, concept_name, concept_code, source FROM tushare_concept_details WHERE ts_code = ?
	`, tsCode).Scan(&details).Error
	if err != nil {
		return nil, err
	}
	return details, nil
}

func (r *ConceptDetailRepo) GetByConcept(ctx context.Context, conceptName string) ([]ConceptDetail, error) {
	var details []ConceptDetail
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, ts_code, concept_name, concept_code, source FROM tushare_concept_details WHERE concept_name = ?
	`, conceptName).Scan(&details).Error
	if err != nil {
		return nil, err
	}
	return details, nil
}

func (r *ConceptDetailRepo) GetByConceptCode(ctx context.Context, conceptCode string) ([]ConceptDetail, error) {
	var details []ConceptDetail
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, ts_code, concept_name, concept_code, source FROM tushare_concept_details WHERE concept_code = ?
	`, conceptCode).Scan(&details).Error
	if err != nil {
		return nil, err
	}
	return details, nil
}
