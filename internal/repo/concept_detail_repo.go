package repo

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "ts_code"}, {Name: "concept_name"}},
		DoNothing: true,
	}).Create(&ConceptDetail{
		TsCode:      tsCode,
		ConceptName: conceptName,
		ConceptCode: conceptCode,
		Source:      "tushare",
	}).Error
}

func (r *ConceptDetailRepo) GetByStock(ctx context.Context, tsCode string) ([]ConceptDetail, error) {
	var details []ConceptDetail
	err := r.db.WithContext(ctx).Select("id, ts_code, concept_name, concept_code, source").
		Where("ts_code = ?", tsCode).Find(&details).Error
	if err != nil {
		return nil, err
	}
	return details, nil
}

func (r *ConceptDetailRepo) GetByConcept(ctx context.Context, conceptName string) ([]ConceptDetail, error) {
	var details []ConceptDetail
	err := r.db.WithContext(ctx).Select("id, ts_code, concept_name, concept_code, source").
		Where("concept_name = ?", conceptName).Find(&details).Error
	if err != nil {
		return nil, err
	}
	return details, nil
}

func (r *ConceptDetailRepo) GetByConceptCode(ctx context.Context, conceptCode string) ([]ConceptDetail, error) {
	var details []ConceptDetail
	err := r.db.WithContext(ctx).Select("id, ts_code, concept_name, concept_code, source").
		Where("concept_code = ?", conceptCode).Find(&details).Error
	if err != nil {
		return nil, err
	}
	return details, nil
}
