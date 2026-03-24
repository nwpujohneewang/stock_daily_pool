package db

import (
	"context"
	"fmt"
	"stock/model/dal_model"
)

var _ TopicDictionaryRepository = (*TopicDictionaryRepoImpl)(nil)

type TopicDictionaryRepository interface {
	GetByRawTopicName(ctx context.Context, rawName string) (*dal_model.TopicDictionary, error)
	GetByNormalizedName(ctx context.Context, normalizedName string) ([]dal_model.TopicDictionary, error)
	GetByCategory(ctx context.Context, category string) ([]dal_model.TopicDictionary, error)
	GetAll(ctx context.Context) ([]dal_model.TopicDictionary, error)
	GetAllMap(ctx context.Context) (map[string]dal_model.TopicDictionary, error)
	Create(ctx context.Context, dict dal_model.TopicDictionary) (*dal_model.TopicDictionary, error)
	BatchCreate(ctx context.Context, dicts []dal_model.TopicDictionary) error
	Delete(ctx context.Context, id int64) error
}

type TopicDictionaryRepoImpl struct{}

func NewTopicDictionaryRepository() *TopicDictionaryRepoImpl {
	return &TopicDictionaryRepoImpl{}
}

func (r TopicDictionaryRepoImpl) GetByRawTopicName(ctx context.Context, rawName string) (*dal_model.TopicDictionary, error) {
	var dict dal_model.TopicDictionary
	err := PostgresStockDB(ctx).Where("raw_topic_name = ?", rawName).First(&dict).Error
	if err != nil {
		return nil, err
	}
	return &dict, nil
}

func (r TopicDictionaryRepoImpl) GetByNormalizedName(ctx context.Context, normalizedName string) ([]dal_model.TopicDictionary, error) {
	var results []dal_model.TopicDictionary
	err := PostgresStockDB(ctx).Where("normalized_name = ?", normalizedName).Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("query by normalized_name: %w", err)
	}
	return results, nil
}

func (r TopicDictionaryRepoImpl) GetByCategory(ctx context.Context, category string) ([]dal_model.TopicDictionary, error) {
	var results []dal_model.TopicDictionary
	err := PostgresStockDB(ctx).Where("category = ?", category).Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("query by category: %w", err)
	}
	return results, nil
}

func (r TopicDictionaryRepoImpl) GetAll(ctx context.Context) ([]dal_model.TopicDictionary, error) {
	var results []dal_model.TopicDictionary
	err := PostgresStockDB(ctx).Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("query all topic_dictionary: %w", err)
	}
	return results, nil
}

func (r TopicDictionaryRepoImpl) GetAllMap(ctx context.Context) (map[string]dal_model.TopicDictionary, error) {
	var rows []dal_model.TopicDictionary
	if err := PostgresStockDB(ctx).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("query all topic_dictionary: %w", err)
	}
	result := make(map[string]dal_model.TopicDictionary)
	for _, row := range rows {
		result[row.RawTopicName] = row
	}
	return result, nil
}

func (r TopicDictionaryRepoImpl) Create(ctx context.Context, dict dal_model.TopicDictionary) (*dal_model.TopicDictionary, error) {
	err := PostgresStockDB(ctx).Create(&dict).Error
	if err != nil {
		return nil, fmt.Errorf("insert topic_dictionary: %w", err)
	}
	return &dict, nil
}

func (r TopicDictionaryRepoImpl) BatchCreate(ctx context.Context, dicts []dal_model.TopicDictionary) error {
	if len(dicts) == 0 {
		return nil
	}
	err := PostgresStockDB(ctx).CreateInBatches(dicts, 100).Error
	if err != nil {
		return fmt.Errorf("batch insert topic_dictionary: %w", err)
	}
	return nil
}

func (r TopicDictionaryRepoImpl) Delete(ctx context.Context, id int64) error {
	return PostgresStockDB(ctx).Delete(&dal_model.TopicDictionary{}, id).Error
}
