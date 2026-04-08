package dao

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"stock/model/dal_model"
)

var _ StockTopicRelationDAO = (*stockTopicRelationDAOImpl)(nil)

type StockTopicRelationDAO interface {
	GetByTsCode(ctx context.Context, tsCode string) ([]dal_model.StockTopicRelation, error)
	GetByTsCodeBatch(ctx context.Context, tsCodes []string) (map[string][]dal_model.StockTopicRelation, error)
	Upsert(ctx context.Context, m *dal_model.StockTopicRelation) error
	UpsertBatch(ctx context.Context, mappings []dal_model.StockTopicRelation) error
	BulkUpsertAccumulate(ctx context.Context, mappings []dal_model.StockTopicRelation) error
	BulkUpsertReplace(ctx context.Context, mappings []dal_model.StockTopicRelation) error
	GetTopicMappings(ctx context.Context, topicID int64) ([]dal_model.StockTopicRelation, error)
	Delete(ctx context.Context, tsCode string, topicID int64) error
	GetByTopicIDs(ctx context.Context, topicIDs []int64) ([]dal_model.StockTopicRelation, error)
}

type stockTopicRelationDAOImpl struct{}

func NewStockTopicRelationDAO() StockTopicRelationDAO {
	return &stockTopicRelationDAOImpl{}
}

func (r stockTopicRelationDAOImpl) GetByTsCode(ctx context.Context, tsCode string) ([]dal_model.StockTopicRelation, error) {
	var ms []dal_model.StockTopicRelation
	err := PostgresStockDB(ctx).Where("ts_code = ?", tsCode).Find(&ms).Error
	return ms, err
}

func (r stockTopicRelationDAOImpl) GetByTsCodeBatch(ctx context.Context, tsCodes []string) (map[string][]dal_model.StockTopicRelation, error) {
	if len(tsCodes) == 0 {
		return nil, nil
	}
	var ms []dal_model.StockTopicRelation
	err := PostgresStockDB(ctx).Where("ts_code IN ?", tsCodes).Find(&ms).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	result := make(map[string][]dal_model.StockTopicRelation, len(tsCodes))
	for _, m := range ms {
		result[m.TsCode] = append(result[m.TsCode], m)
	}
	return result, nil
}

func (r stockTopicRelationDAOImpl) Upsert(ctx context.Context, m *dal_model.StockTopicRelation) error {
	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ts_code"}, {Name: "topic_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"topic_name", "category", "hit_count", "last_seen_date", "updated_at",
		}),
	}).Create(m).Error
}

func (r stockTopicRelationDAOImpl) UpsertBatch(ctx context.Context, mappings []dal_model.StockTopicRelation) error {
	if len(mappings) == 0 {
		return nil
	}
	return PostgresStockDB(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "ts_code"}, {Name: "topic_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"topic_name", "category", "hit_count", "last_seen_date", "updated_at",
			}),
		}).
		CreateInBatches(mappings, 500).Error
}

func (r stockTopicRelationDAOImpl) GetTopicMappings(ctx context.Context, topicID int64) ([]dal_model.StockTopicRelation, error) {
	var ms []dal_model.StockTopicRelation
	err := PostgresStockDB(ctx).Where("topic_id = ?", topicID).Find(&ms).Error
	return ms, err
}

func (r stockTopicRelationDAOImpl) Delete(ctx context.Context, tsCode string, topicID int64) error {
	return PostgresStockDB(ctx).
		Where("ts_code = ? AND topic_id = ?", tsCode, topicID).
		Delete(&dal_model.StockTopicRelation{}).Error
}

func (r stockTopicRelationDAOImpl) GetByTopicIDs(ctx context.Context, topicIDs []int64) ([]dal_model.StockTopicRelation, error) {
	if len(topicIDs) == 0 {
		return nil, nil
	}
	var ms []dal_model.StockTopicRelation
	err := PostgresStockDB(ctx).Where("topic_id IN ?", topicIDs).Find(&ms).Error
	return ms, err
}

func (r stockTopicRelationDAOImpl) BulkUpsertAccumulate(ctx context.Context, mappings []dal_model.StockTopicRelation) error {
	if len(mappings) == 0 {
		return nil
	}

	now := Now()
	for i := range mappings {
		mappings[i].UpdatedAt = now
	}

	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ts_code"}, {Name: "topic_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"topic_name":     clause.Column{Table: "excluded", Name: "topic_name"},
			"category":       clause.Column{Table: "excluded", Name: "category"},
			"hit_count":      gorm.Expr("stock_topic_relations.hit_count + excluded.hit_count"),
			"last_seen_date": gorm.Expr("GREATEST(stock_topic_relations.last_seen_date, excluded.last_seen_date)"),
			"updated_at":     clause.Column{Table: "excluded", Name: "updated_at"},
		}),
	}).CreateInBatches(mappings, 500).Error
}

func (r stockTopicRelationDAOImpl) BulkUpsertReplace(ctx context.Context, mappings []dal_model.StockTopicRelation) error {
	if len(mappings) == 0 {
		return nil
	}

	now := Now()
	for i := range mappings {
		mappings[i].UpdatedAt = now
	}

	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ts_code"}, {Name: "topic_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"topic_name":      clause.Column{Table: "excluded", Name: "topic_name"},
			"category":        clause.Column{Table: "excluded", Name: "category"},
			"source":          clause.Column{Table: "excluded", Name: "source"},
			"hit_count":       clause.Column{Table: "excluded", Name: "hit_count"},
			"first_seen_date": clause.Column{Table: "excluded", Name: "first_seen_date"},
			"last_seen_date":  clause.Column{Table: "excluded", Name: "last_seen_date"},
			"updated_at":      clause.Column{Table: "excluded", Name: "updated_at"},
		}),
	}).CreateInBatches(mappings, 500).Error
}
