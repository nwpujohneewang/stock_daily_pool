// internal/service/topic_query_service.go
package topic

import (
	"context"
	"fmt"

	"stock/dal/repo"
)

type TopicQueryServiceImpl struct{}

func NewTopicQueryService() *TopicQueryServiceImpl {
	return &TopicQueryServiceImpl{}
}

func (s *TopicQueryServiceImpl) List(ctx context.Context, params TopicQueryParams) (*TopicListResult, error) {
	topicRepo := repo.NewTopicRepository()
	topics, total, err := topicRepo.List(ctx, params.Keyword, params.Page, params.PageSize)
	if err != nil {
		return nil, fmt.Errorf("list topics: %w", err)
	}

	items := make([]TopicQueryResult, 0, len(topics))
	for _, t := range topics {
		items = append(items, TopicQueryResult{
			ID:              t.ID,
			Name:            t.Name,
			Category:        t.Category,
			Source:          t.Source,
			OccurrenceCount: t.OccurrenceCount,
			FirstSeenDate:   t.FirstSeenDate,
			LastSeenDate:    t.LastSeenDate,
		})
	}

	return &TopicListResult{
		Items: items,
		Total: total,
	}, nil
}

func (s *TopicQueryServiceImpl) GetByID(ctx context.Context, id int64) (*TopicQueryResult, error) {
	topicRepo := repo.NewTopicRepository()
	topic, err := topicRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get topic by id: %w", err)
	}
	if topic == nil {
		return nil, ErrTopicNotFound
	}

	return &TopicQueryResult{
		ID:              topic.ID,
		Name:            topic.Name,
		Category:        topic.Category,
		Source:          topic.Source,
		OccurrenceCount: topic.OccurrenceCount,
		FirstSeenDate:   topic.FirstSeenDate,
		LastSeenDate:    topic.LastSeenDate,
	}, nil
}
