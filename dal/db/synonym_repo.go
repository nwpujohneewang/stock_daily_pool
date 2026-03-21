package db

import (
	"context"
	"fmt"

	"stock/internal/model"
)

var _ SynonymRepository = (*SynonymRepoImpl)(nil)

type SynonymRepository interface {
	GetByTopicID(ctx context.Context, topicID int64) ([]model.TopicSynonym, error)
	GetBySynonym(ctx context.Context, synonym string) (*model.TopicSynonym, error)
	Create(ctx context.Context, synonym model.TopicSynonym) (*model.TopicSynonym, error)
	Delete(ctx context.Context, id int64) error
	BatchCreate(ctx context.Context, synonyms []model.TopicSynonym) error
	GetAllSynonymsMap(ctx context.Context) (map[string]int64, error)
}

type SynonymRepoImpl struct{}

func NewSynonymRepository() *SynonymRepoImpl {
	return &SynonymRepoImpl{}
}

func (r SynonymRepoImpl) GetByTopicID(ctx context.Context, topicID int64) ([]model.TopicSynonym, error) {
	var results []model.TopicSynonym
	err := PostgresStockDB(ctx).Where("topic_id = ?", topicID).Order("id").Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("query synonyms: %w", err)
	}
	return results, nil
}

func (r SynonymRepoImpl) GetBySynonym(ctx context.Context, synonym string) (*model.TopicSynonym, error) {
	var s model.TopicSynonym
	err := PostgresStockDB(ctx).Where("synonym = ?", synonym).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r SynonymRepoImpl) Create(ctx context.Context, synonym model.TopicSynonym) (*model.TopicSynonym, error) {
	err := PostgresStockDB(ctx).Create(&synonym).Error
	if err != nil {
		return nil, fmt.Errorf("insert synonym: %w", err)
	}
	return &synonym, nil
}

func (r SynonymRepoImpl) Delete(ctx context.Context, id int64) error {
	return PostgresStockDB(ctx).Delete(&model.TopicSynonym{}, id).Error
}

func (r SynonymRepoImpl) BatchCreate(ctx context.Context, synonyms []model.TopicSynonym) error {
	if len(synonyms) == 0 {
		return nil
	}
	err := PostgresStockDB(ctx).CreateInBatches(synonyms, 100).Error
	if err != nil {
		return fmt.Errorf("batch insert synonym: %w", err)
	}
	return nil
}

func (r SynonymRepoImpl) GetAllSynonymsMap(ctx context.Context) (map[string]int64, error) {
	type synonymRow struct {
		Synonym string
		TopicID int64
	}
	var rows []synonymRow
	if err := PostgresStockDB(ctx).Select("synonym, topic_id").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("query all synonyms: %w", err)
	}
	result := make(map[string]int64)
	for _, row := range rows {
		result[row.Synonym] = row.TopicID
	}
	return result, nil
}
