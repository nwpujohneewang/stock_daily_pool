package dao

import (
	"context"
	"time"

	"gorm.io/gorm/clause"

	"stock/model/dal_model"
)

var _ LLMClassifyResultDAO = (*llmClassifyResultDAOImpl)(nil)

type LLMClassifyResultDAO interface {
	GetByDateAndTsCodes(ctx context.Context, date string, tsCodes []string) ([]dal_model.LLMClassifyResult, error)
	UpsertBatch(ctx context.Context, results []dal_model.LLMClassifyResult) error
}

type llmClassifyResultDAOImpl struct{}

func NewLLMClassifyResultDAO() LLMClassifyResultDAO {
	return &llmClassifyResultDAOImpl{}
}

func (d *llmClassifyResultDAOImpl) GetByDateAndTsCodes(ctx context.Context, date string, tsCodes []string) ([]dal_model.LLMClassifyResult, error) {
	var results []dal_model.LLMClassifyResult
	err := PostgresStockDB(ctx).
		Where("date = ? AND ts_code IN ?", date, tsCodes).
		Find(&results).Error
	return results, err
}

func (d *llmClassifyResultDAOImpl) UpsertBatch(ctx context.Context, results []dal_model.LLMClassifyResult) error {
	if len(results) == 0 {
		return nil
	}
	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "ts_code"}, {Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{"topic", "reason"}),
	}).CreateInBatches(results, 100).Error
}

func ParseDate(dateStr string) time.Time {
	t, _ := time.Parse("2006-01-02", dateStr)
	return t
}
