package db

import (
	"context"
	"stock/model/dal_model"

	"gorm.io/gorm/clause"
)

type ConceptDetailRepoImpl struct{}

func NewConceptDetailRepository() *ConceptDetailRepoImpl {
	return &ConceptDetailRepoImpl{}
}

func (r ConceptDetailRepoImpl) Upsert(ctx context.Context, tsCode, conceptName, conceptCode string) error {
	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "ts_code"}, {Name: "concept_name"}},
		DoNothing: true,
	}).Create(&dal_model.TushareConceptDetail{
		TsCode:      tsCode,
		ConceptName: conceptName,
		ConceptCode: conceptCode,
		Source:      "tushare",
	}).Error
}

func (r ConceptDetailRepoImpl) GetByStock(ctx context.Context, tsCode string) ([]dal_model.TushareConceptDetail, error) {
	var details []dal_model.TushareConceptDetail
	err := PostgresStockDB(ctx).Select("id, ts_code, concept_name, concept_code, source").
		Where("ts_code = ?", tsCode).Find(&details).Error
	if err != nil {
		return nil, err
	}
	return details, nil
}

func (r ConceptDetailRepoImpl) GetByConcept(ctx context.Context, conceptName string) ([]dal_model.TushareConceptDetail, error) {
	var details []dal_model.TushareConceptDetail
	err := PostgresStockDB(ctx).Select("id, ts_code, concept_name, concept_code, source").
		Where("concept_name = ?", conceptName).Find(&details).Error
	if err != nil {
		return nil, err
	}
	return details, nil
}

func (r ConceptDetailRepoImpl) GetByConceptCode(ctx context.Context, conceptCode string) ([]dal_model.TushareConceptDetail, error) {
	var details []dal_model.TushareConceptDetail
	err := PostgresStockDB(ctx).Select("id, ts_code, concept_name, concept_code, source").
		Where("concept_code = ?", conceptCode).Find(&details).Error
	if err != nil {
		return nil, err
	}
	return details, nil
}
