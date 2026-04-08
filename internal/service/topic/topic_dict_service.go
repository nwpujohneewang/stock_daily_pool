// internal/service/topic/topic_dict_service.go
package topic

import (
	"context"
	"fmt"

	"stock/dal/repo"
	"stock/model/dal_model"
)

type TopicDictServiceImpl struct{}

func NewTopicDictService() *TopicDictServiceImpl {
	return &TopicDictServiceImpl{}
}

func (s *TopicDictServiceImpl) List(ctx context.Context) ([]TopicDictResult, error) {
	topicDictRepo := repo.NewTopicDictionaryRepository()
	dicts, err := topicDictRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all topic dictionaries: %w", err)
	}

	results := make([]TopicDictResult, 0, len(dicts))
	for _, d := range dicts {
		results = append(results, TopicDictResult{
			ID:             d.ID,
			RawTopicName:   d.RawTopicName,
			NormalizedName: d.NormalizedName,
			Category:       d.Category,
		})
	}
	return results, nil
}

func (s *TopicDictServiceImpl) Create(ctx context.Context, params TopicDictParams) (*TopicDictResult, error) {
	topicDictRepo := repo.NewTopicDictionaryRepository()

	dict := dal_model.TopicDictionary{
		RawTopicName:   params.RawTopicName,
		NormalizedName: params.NormalizedName,
		Category:       params.Category,
	}

	created, err := topicDictRepo.Create(ctx, dict)
	if err != nil {
		return nil, fmt.Errorf("create topic dictionary: %w", err)
	}

	return &TopicDictResult{
		ID:             created.ID,
		RawTopicName:   created.RawTopicName,
		NormalizedName: created.NormalizedName,
		Category:       created.Category,
	}, nil
}

func (s *TopicDictServiceImpl) Update(ctx context.Context, id int64, params TopicDictParams) error {
	topicDictRepo := repo.NewTopicDictionaryRepository()

	dict := dal_model.TopicDictionary{
		ID:             id,
		RawTopicName:   params.RawTopicName,
		NormalizedName: params.NormalizedName,
		Category:       params.Category,
	}

	if err := topicDictRepo.Update(ctx, dict); err != nil {
		return fmt.Errorf("update topic dictionary: %w", err)
	}
	return nil
}

func (s *TopicDictServiceImpl) Delete(ctx context.Context, id int64) error {
	topicDictRepo := repo.NewTopicDictionaryRepository()
	if err := topicDictRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete topic dictionary: %w", err)
	}
	return nil
}
