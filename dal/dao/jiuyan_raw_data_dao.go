package dao

import (
	"context"
	"time"

	"gorm.io/gorm/clause"

	"stock/model/dal_model"
)

// AggregatedRawData 聚合后的原始数据
type AggregatedRawData struct {
	TopicName string
	StockCode string
	HitCount  int
	FirstSeen time.Time
	LastSeen  time.Time
}

type JiuyanRawDataDAO interface {
	UpsertBatch(ctx context.Context, records []dal_model.JiuyanRawData) error
	GetDistinctTopicNames(ctx context.Context, startDate, endDate string) ([]string, error)
	GetAggregatedData(ctx context.Context, startDate, endDate string) ([]AggregatedRawData, error)
}

type jiuyanRawDataDAOImpl struct{}

func NewJiuyanRawDataDAO() JiuyanRawDataDAO {
	return &jiuyanRawDataDAOImpl{}
}

func (r *jiuyanRawDataDAOImpl) UpsertBatch(ctx context.Context, records []dal_model.JiuyanRawData) error {
	if len(records) == 0 {
		return nil
	}
	return PostgresStockDB(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "date"}, {Name: "topic_name"}, {Name: "stock_code"}},
			DoUpdates: clause.AssignmentColumns([]string{"stock_name"}),
		}).
		CreateInBatches(records, 500).Error
}

func (r *jiuyanRawDataDAOImpl) GetDistinctTopicNames(ctx context.Context, startDate, endDate string) ([]string, error) {
	var topicNames []string
	err := PostgresStockDB(ctx).Table("jiuyan_raw_data").
		Select("DISTINCT topic_name").
		Where("date >= ? AND date <= ?", startDate, endDate).
		Pluck("topic_name", &topicNames).Error
	return topicNames, err
}

func (r *jiuyanRawDataDAOImpl) GetAggregatedData(ctx context.Context, startDate, endDate string) ([]AggregatedRawData, error) {
	var rows []AggregatedRawData
	err := PostgresStockDB(ctx).Table("jiuyan_raw_data").
		Select("topic_name, stock_code, COUNT(*) AS hit_count, MIN(date) AS first_seen, MAX(date) AS last_seen").
		Where("date >= ? AND date <= ?", startDate, endDate).
		Group("topic_name, stock_code").
		Find(&rows).Error
	return rows, err
}
