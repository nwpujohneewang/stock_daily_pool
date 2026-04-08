package dao

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"stock/model/dal_model"
)

var _ TopicDictionaryDAO = (*topicDictionaryDAOImpl)(nil)

type TopicDictionaryDAO interface {
	GetByRawTopicName(ctx context.Context, rawName string) (*dal_model.TopicDictionary, error)
	GetByNormalizedName(ctx context.Context, normalizedName string) ([]dal_model.TopicDictionary, error)
	GetByCategory(ctx context.Context, category string) ([]dal_model.TopicDictionary, error)
	GetAll(ctx context.Context) ([]dal_model.TopicDictionary, error)
	GetAllMap(ctx context.Context) (map[string]dal_model.TopicDictionary, error)
	Create(ctx context.Context, dict dal_model.TopicDictionary) (*dal_model.TopicDictionary, error)
	Update(ctx context.Context, dict dal_model.TopicDictionary) error
	BatchCreate(ctx context.Context, dicts []dal_model.TopicDictionary) error
	Delete(ctx context.Context, id int64) error
}

type topicDictionaryDAOImpl struct{}

func NewTopicDictionaryDAO() TopicDictionaryDAO {
	return &topicDictionaryDAOImpl{}
}

func (r topicDictionaryDAOImpl) GetByRawTopicName(ctx context.Context, rawName string) (*dal_model.TopicDictionary, error) {
	var dict dal_model.TopicDictionary
	err := PostgresStockDB(ctx).Where("raw_topic_name = ?", rawName).First(&dict).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &dict, nil
}

func (r topicDictionaryDAOImpl) GetByNormalizedName(ctx context.Context, normalizedName string) ([]dal_model.TopicDictionary, error) {
	var results []dal_model.TopicDictionary
	err := PostgresStockDB(ctx).Where("normalized_name = ?", normalizedName).Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("query by normalized_name: %w", err)
	}
	return results, nil
}

func (r topicDictionaryDAOImpl) GetByCategory(ctx context.Context, category string) ([]dal_model.TopicDictionary, error) {
	var results []dal_model.TopicDictionary
	err := PostgresStockDB(ctx).Where("category = ?", category).Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("query by category: %w", err)
	}
	return results, nil
}

func (r topicDictionaryDAOImpl) GetAll(ctx context.Context) ([]dal_model.TopicDictionary, error) {
	var results []dal_model.TopicDictionary
	err := PostgresStockDB(ctx).Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("query all topic_dictionary: %w", err)
	}
	return results, nil
}

func (r topicDictionaryDAOImpl) GetAllMap(ctx context.Context) (map[string]dal_model.TopicDictionary, error) {
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

func (r topicDictionaryDAOImpl) Create(ctx context.Context, dict dal_model.TopicDictionary) (*dal_model.TopicDictionary, error) {
	err := PostgresStockDB(ctx).Create(&dict).Error
	if err != nil {
		return nil, fmt.Errorf("insert topic_dictionary: %w", err)
	}
	return &dict, nil
}

func (r topicDictionaryDAOImpl) Update(ctx context.Context, dict dal_model.TopicDictionary) error {
	return PostgresStockDB(ctx).Model(&dal_model.TopicDictionary{}).Where("id = ?", dict.ID).Updates(map[string]interface{}{
		"raw_topic_name":  dict.RawTopicName,
		"normalized_name": dict.NormalizedName,
		"category":        dict.Category,
		"updated_at":      "NOW()",
	}).Error
}

func (r topicDictionaryDAOImpl) BatchCreate(ctx context.Context, dicts []dal_model.TopicDictionary) error {
	if len(dicts) == 0 {
		return nil
	}
	err := PostgresStockDB(ctx).CreateInBatches(dicts, 100).Error
	if err != nil {
		return fmt.Errorf("batch insert topic_dictionary: %w", err)
	}
	return nil
}

func (r topicDictionaryDAOImpl) Delete(ctx context.Context, id int64) error {
	return PostgresStockDB(ctx).Delete(&dal_model.TopicDictionary{}, id).Error
}
