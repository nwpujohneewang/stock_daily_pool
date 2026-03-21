package db

import (
	"context"
	"stock/model/dal_model"
	"time"

	"gorm.io/gorm/clause"
)

var _ StockTopicRelationRepository = (*StockTopicRelationRepoImpl)(nil)

type StockTopicRelationRepository interface {
	GetByTsCode(ctx context.Context, tsCode string) ([]dal_model.StockTopicRelation, error)
	Upsert(ctx context.Context, m *dal_model.StockTopicRelation) error
	UpsertBatch(ctx context.Context, mappings []dal_model.StockTopicRelation) error
	GetTopicMappings(ctx context.Context, topicID int64) ([]dal_model.StockTopicRelation, error)
	Delete(ctx context.Context, tsCode string, topicID int64) error
	GetByTopicIDs(ctx context.Context, topicIDs []int64) ([]dal_model.StockTopicRelation, error)
}

type StockTopicRelationRepoImpl struct{}

func NewStockTopicRelationRepository() *StockTopicRelationRepoImpl {
	return &StockTopicRelationRepoImpl{}
}

func (r StockTopicRelationRepoImpl) GetByTsCode(ctx context.Context, tsCode string) ([]dal_model.StockTopicRelation, error) {
	var ms []dal_model.StockTopicRelation
	err := PostgresStockDB(ctx).Where("ts_code = ?", tsCode).Find(&ms).Error
	return ms, err
}

func (r StockTopicRelationRepoImpl) Upsert(ctx context.Context, m *dal_model.StockTopicRelation) error {
	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ts_code"}, {Name: "topic_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"topic_name", "hit_count", "last_seen_date", "updated_at",
		}),
	}).Create(m).Error
}

func (r StockTopicRelationRepoImpl) UpsertBatch(ctx context.Context, mappings []dal_model.StockTopicRelation) error {
	if len(mappings) == 0 {
		return nil
	}
	return PostgresStockDB(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "ts_code"}, {Name: "topic_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"topic_name", "hit_count", "last_seen_date", "updated_at",
			}),
		}).
		CreateInBatches(mappings, 500).Error
}

func (r StockTopicRelationRepoImpl) GetTopicMappings(ctx context.Context, topicID int64) ([]dal_model.StockTopicRelation, error) {
	var ms []dal_model.StockTopicRelation
	err := PostgresStockDB(ctx).Where("topic_id = ?", topicID).Find(&ms).Error
	return ms, err
}

func (r StockTopicRelationRepoImpl) Delete(ctx context.Context, tsCode string, topicID int64) error {
	return PostgresStockDB(ctx).
		Where("ts_code = ? AND topic_id = ?", tsCode, topicID).
		Delete(&dal_model.StockTopicRelation{}).Error
}

func (r StockTopicRelationRepoImpl) GetByTopicIDs(ctx context.Context, topicIDs []int64) ([]dal_model.StockTopicRelation, error) {
	if len(topicIDs) == 0 {
		return nil, nil
	}
	var ms []dal_model.StockTopicRelation
	err := PostgresStockDB(ctx).Where("topic_id IN ?", topicIDs).Find(&ms).Error
	return ms, err
}

func (r StockTopicRelationRepoImpl) BulkUpsertAccumulate(ctx context.Context, mappings []dal_model.StockTopicRelation) error {
	if len(mappings) == 0 {
		return nil
	}

	valueStrings := make([]string, 0, len(mappings))
	valueArgs := make([]interface{}, 0, len(mappings)*8)
	now := time.Now()

	for _, m := range mappings {
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?)")
		valueArgs = append(valueArgs, m.TsCode, m.TopicID, m.Source, m.TopicName, m.HitCount, m.LastSeenDate, m.FirstSeenDate, now)
	}

	sql := "INSERT INTO stock_topic_relations (ts_code, topic_id, source, topic_name, hit_count, last_seen_date, first_seen_date, updated_at) VALUES " +
		joinStrings(valueStrings) +
		" ON CONFLICT (ts_code, topic_id) DO UPDATE SET " +
		"topic_name = EXCLUDED.topic_name," +
		"hit_count = stock_topic_relations.hit_count + EXCLUDED.hit_count," +
		"last_seen_date = GREATEST(stock_topic_relations.last_seen_date, EXCLUDED.last_seen_date)," +
		"updated_at = NOW()"

	return PostgresStockDB(ctx).Exec(sql, valueArgs...).Error
}

func joinStrings(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += "," + strs[i]
	}
	return result
}
