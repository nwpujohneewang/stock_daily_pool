package repo

import (
	"context"
	"stock/dal/cache"
	"stock/dal/dao"
	"stock/model/dal_model"
)

type TopicDictionaryRepository interface {
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

type topicDictionaryRepoImpl struct {
	dao dao.TopicDictionaryDAO
}

func NewTopicDictionaryRepository() TopicDictionaryRepository {
	return &topicDictionaryRepoImpl{dao: dao.NewTopicDictionaryDAO()}
}

func (r *topicDictionaryRepoImpl) GetByRawTopicName(ctx context.Context, rawName string) (*dal_model.TopicDictionary, error) {
	dictMap, err := r.GetAllMap(ctx)
	if err == nil {
		if dict, ok := dictMap[rawName]; ok {
			copied := dict
			return &copied, nil
		}
	}
	return r.dao.GetByRawTopicName(ctx, rawName)
}

func (r *topicDictionaryRepoImpl) GetByNormalizedName(ctx context.Context, normalizedName string) ([]dal_model.TopicDictionary, error) {
	dictMap, err := r.GetAllMap(ctx)
	if err == nil {
		result := make([]dal_model.TopicDictionary, 0)
		for _, dict := range dictMap {
			if dict.NormalizedName == normalizedName {
				result = append(result, dict)
			}
		}
		return result, nil
	}
	return r.dao.GetByNormalizedName(ctx, normalizedName)
}

func (r *topicDictionaryRepoImpl) GetByCategory(ctx context.Context, category string) ([]dal_model.TopicDictionary, error) {
	dictMap, err := r.GetAllMap(ctx)
	if err == nil {
		result := make([]dal_model.TopicDictionary, 0)
		for _, dict := range dictMap {
			if dict.Category == category {
				result = append(result, dict)
			}
		}
		return result, nil
	}
	return r.dao.GetByCategory(ctx, category)
}

func (r *topicDictionaryRepoImpl) GetAll(ctx context.Context) ([]dal_model.TopicDictionary, error) {
	topicDictCache := cache.NewTopicDictionaryCache()
	if dicts, found := topicDictCache.GetAll(ctx); found {
		return dicts, nil
	}
	dicts, err := r.dao.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	_ = topicDictCache.SetAll(ctx, dicts)
	return dicts, nil
}

func (r *topicDictionaryRepoImpl) GetAllMap(ctx context.Context) (map[string]dal_model.TopicDictionary, error) {
	topicDictCache := cache.NewTopicDictionaryCache()
	if dictMap, found := topicDictCache.GetAllMap(ctx); found {
		return dictMap, nil
	}
	dictMap, err := r.dao.GetAllMap(ctx)
	if err != nil {
		return nil, err
	}
	_ = topicDictCache.SetAllMap(ctx, dictMap)
	return dictMap, nil
}

func (r *topicDictionaryRepoImpl) Create(ctx context.Context, dict dal_model.TopicDictionary) (*dal_model.TopicDictionary, error) {
	created, err := r.dao.Create(ctx, dict)
	if err != nil {
		return nil, err
	}
	_ = r.invalidateCaches(ctx)
	return created, nil
}

func (r *topicDictionaryRepoImpl) Update(ctx context.Context, dict dal_model.TopicDictionary) error {
	if err := r.dao.Update(ctx, dict); err != nil {
		return err
	}
	return r.invalidateCaches(ctx)
}

func (r *topicDictionaryRepoImpl) BatchCreate(ctx context.Context, dicts []dal_model.TopicDictionary) error {
	if err := r.dao.BatchCreate(ctx, dicts); err != nil {
		return err
	}
	return r.invalidateCaches(ctx)
}

func (r *topicDictionaryRepoImpl) Delete(ctx context.Context, id int64) error {
	if err := r.dao.Delete(ctx, id); err != nil {
		return err
	}
	return r.invalidateCaches(ctx)
}

func (r *topicDictionaryRepoImpl) invalidateCaches(ctx context.Context) error {
	topicDictCache := cache.NewTopicDictionaryCache()
	_ = topicDictCache.ClearAll(ctx)
	_ = topicDictCache.ClearAllMap(ctx)
	return nil
}
