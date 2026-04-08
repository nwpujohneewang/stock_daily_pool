package dao

import (
	"context"

	"gorm.io/gorm/clause"

	"stock/model/dal_model"
)

var _ TopicStockCountDAO = (*topicStockCountDAOImpl)(nil)

type TopicStockCountDAO interface {
	UpsertBatch(ctx context.Context, counts []dal_model.TopicStockCount) error
	GetAll(ctx context.Context) ([]dal_model.TopicStockCount, error)
	AggregateFromRelations(ctx context.Context) ([]dal_model.TopicStockCount, error)
	GetByTopicIDs(ctx context.Context, topicIDs []int64) ([]dal_model.TopicStockCount, error)
}

type topicStockCountDAOImpl struct{}

func NewTopicStockCountDAO() TopicStockCountDAO {
	return &topicStockCountDAOImpl{}
}

func (r topicStockCountDAOImpl) UpsertBatch(ctx context.Context, counts []dal_model.TopicStockCount) error {
	if len(counts) == 0 {
		return nil
	}

	now := Now()
	for i := range counts {
		if counts[i].ComputedDate.IsZero() {
			counts[i].ComputedDate = now
		}
		if counts[i].CreatedAt.IsZero() {
			counts[i].CreatedAt = now
		}
		counts[i].UpdatedAt = now
	}

	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "topic_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"total_stock_count": clause.Column{Table: "excluded", Name: "total_stock_count"},
			"computed_date":     clause.Column{Table: "excluded", Name: "computed_date"},
			"updated_at":        clause.Column{Table: "excluded", Name: "updated_at"},
		}),
	}).CreateInBatches(counts, 500).Error
}

func (r topicStockCountDAOImpl) GetAll(ctx context.Context) ([]dal_model.TopicStockCount, error) {
	var counts []dal_model.TopicStockCount
	err := PostgresStockDB(ctx).Order("topic_id ASC").Find(&counts).Error
	return counts, err
}

func (r topicStockCountDAOImpl) GetByTopicIDs(ctx context.Context, topicIDs []int64) ([]dal_model.TopicStockCount, error) {
	if len(topicIDs) == 0 {
		return nil, nil
	}

	var counts []dal_model.TopicStockCount
	err := PostgresStockDB(ctx).
		Where("topic_id IN ?", topicIDs).
		Order("topic_id ASC").
		Find(&counts).Error
	return counts, err
}

func (r topicStockCountDAOImpl) AggregateFromRelations(ctx context.Context) ([]dal_model.TopicStockCount, error) {
	type row struct {
		TopicID         int64 `gorm:"column:topic_id"`
		TotalStockCount int   `gorm:"column:total_stock_count"`
	}

	var rows []row
	err := PostgresStockDB(ctx).
		Table("stock_topic_relations").
		Select("topic_id, COUNT(DISTINCT ts_code) AS total_stock_count").
		Group("topic_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	computedAt := Now()
	counts := make([]dal_model.TopicStockCount, 0, len(rows))
	for _, item := range rows {
		counts = append(counts, dal_model.TopicStockCount{
			TopicID:         item.TopicID,
			TotalStockCount: item.TotalStockCount,
			ComputedDate:    computedAt,
		})
	}
	return counts, nil
}
